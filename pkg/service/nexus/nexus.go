package nexus

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/dchest/uniuri"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/v2/pkg/controller/helper"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
)

var log = ctrl.Log.WithName("nexus_service")

const (
	imgFolder = "img"
	nexusIcon = "nexus.svg"
)

// NexusService interface for Nexus EDP component
type Service interface {
	Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, bool, error)
	ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error)
	ClientForNexusChild(ctx context.Context, child Child) (*nexus.Client, error)
}

type Client interface {
	IsNexusRestApiReady() (bool, int, error)
	DeclareDefaultScripts(listOfScripts map[string]string) error
	AreDefaultScriptsDeclared(listOfScripts map[string]string) (bool, error)
	RunScript(scriptName string, parameters map[string]interface{}) ([]byte, error)
}

// NewNexusService function that returns NexusService implementation
func NewService(platformService platform.PlatformService, client client.Client, scheme *runtime.Scheme) Service {
	return ServiceImpl{
		platformService:      platformService,
		client:               client,
		keycloakHelper:       keycloakHelper.MakeHelper(client, scheme),
		runningInClusterFunc: helper.RunningInCluster,
		clientBuilder: func(url string, user string, password string) Client {
			return nexus.Init(url, user, password)
		},
	}
}

// NexusServiceImpl struct fo Nexus EDP Component
type ServiceImpl struct {
	platformService      platform.PlatformService
	client               client.Client
	keycloakHelper       *keycloakHelper.Helper
	runningInClusterFunc func() bool
	clientBuilder        func(url string, user string, password string) Client
}

// IsDeploymentReady check if deployment for Nexus is ready
func (s ServiceImpl) IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error) {
	return s.platformService.IsDeploymentReady(instance)
}

func (s ServiceImpl) getNexusRestApiUrl(instance v1alpha1.Nexus) (string, error) {
	basePath := ""
	if len(instance.Spec.BasePath) > 0 {
		basePath = fmt.Sprintf("/%v", instance.Spec.BasePath)
	}
	u := fmt.Sprintf("http://%v.%v:%v%v/%v", instance.Name, instance.Namespace, nexusDefaultSpec.NexusPort, basePath, nexusDefaultSpec.NexusRestApiUrlPath)
	if s.runningInClusterFunc != nil {
		if !s.runningInClusterFunc() {
			eu, _, _, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
			if err != nil {
				return "", errors.Wrapf(err, "failed to get Route for %v/%v", instance.Namespace, instance.Name)
			}
			u = fmt.Sprintf("%v/%v", eu, nexusDefaultSpec.NexusRestApiUrlPath)
		}
	} else {
		return "", errors.New("missing runningInClusterFunc")
	}

	return u, nil
}

func (s ServiceImpl) getNexusAdminPassword(instance v1alpha1.Nexus) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	nexusAdminCredentials, err := s.platformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
	}
	return string(nexusAdminCredentials["password"]), nil
}

func (s ServiceImpl) setAnnotation(instance *v1alpha1.Nexus, key string, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}
	} else {
		instance.ObjectMeta.Annotations[key] = value
	}
}

// Integration performs integration Nexus with other EDP components
func (s ServiceImpl) Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {

	if instance.Spec.KeycloakSpec.Enabled {
		keycloakClient, err := s.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to get Keycloak client data!")
		}

		keycloakRealm, err := s.keycloakHelper.GetOwnerKeycloakRealm(keycloakClient.ObjectMeta)
		if err != nil {
			return &instance, nil
		}

		if keycloakRealm == nil {
			return &instance, errors.New("Keycloak Realm CR in not created yet!")
		}

		keycloak, err := s.keycloakHelper.GetOwnerKeycloak(keycloakRealm.ObjectMeta)
		if err != nil {
			errMsg := fmt.Sprintf("failed to get owner for %s/%s", keycloakClient.Namespace, keycloakClient.Name)
			return &instance, errors.Wrap(err, errMsg)
		}

		if keycloak == nil {
			return &instance, errors.New("Keycloak CR is not created yet")
		}

		_, host, scheme, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to get route")
		}

		var proxyConfig []string
		upstreamUrl := fmt.Sprintf("--upstream-url=http://127.0.0.1:%v", nexusDefaultSpec.NexusPort)
		if len(instance.Spec.BasePath) != 0 {
			upstreamUrl = fmt.Sprintf("%v/%v", upstreamUrl, instance.Spec.BasePath)
			proxyConfig = append(proxyConfig, fmt.Sprintf("--base-uri=/%v", instance.Spec.BasePath))
		}

		ru := fmt.Sprintf("--redirection-url=%v", fmt.Sprintf("%v://%v", scheme, host))
		id := fmt.Sprintf("--client-id=%v", keycloakClient.Spec.ClientId)
		secret := "--client-secret=42"
		du := fmt.Sprintf("--discovery-url=%s/auth/realms/%s", keycloak.Spec.Url, keycloakRealm.Spec.RealmName)
		uu := upstreamUrl
		listen := fmt.Sprintf("--listen=0.0.0.0:%d", nexusDefaultSpec.NexusKeycloakProxyPort)

		proxyConfig = append(
			proxyConfig,
			"--skip-openid-provider-tls-verify=true",
			du,
			id,
			secret,
			listen,
			ru,
			uu,
		)

		if len(instance.Spec.KeycloakSpec.Roles) > 0 {
			proxyConfig = append(proxyConfig,
				fmt.Sprintf("--resources=uri=/*|roles=%s|require-any-role=true",
					strings.Join(instance.Spec.KeycloakSpec.Roles, ",")))
		}

		err = s.platformService.AddKeycloakProxyToDeployConf(instance, proxyConfig)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to add Keycloak proxy")
		}

		keyCloakProxyPort := coreV1Api.ServicePort{
			Name:       "keycloak-proxy",
			Port:       nexusDefaultSpec.NexusKeycloakProxyPort,
			Protocol:   coreV1Api.ProtocolTCP,
			TargetPort: intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort},
		}
		if err = s.platformService.AddPortToService(instance, keyCloakProxyPort); err != nil {
			return &instance, errors.Wrap(err, "failed to add Keycloak proxy port to service")
		}

		if err = s.platformService.UpdateExternalTargetPath(instance, intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort}); err != nil {
			return &instance, errors.Wrap(err, "failed to update target port in Route")
		}
	} else {
		log.V(1).Info("Keycloak integration not enabled.")
	}

	return &instance, nil
}

// ExposeConfiguration performs exposing Nexus configuration for other EDP components
func (s ServiceImpl) ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	u, err := s.getNexusRestApiUrl(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "failed to get Nexus REST API URL")
	}

	nexusPassword, err := s.getNexusAdminPassword(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "failed to get Nexus admin password from secret")
	}

	nexusClient := nexus.Init(u, nexusDefaultSpec.NexusDefaultAdminUser, nexusPassword)

	nexusDefaultUsersToCreate, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix))
	if err != nil {
		return &instance, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	var newUserSecretName string
	var parsedUsers []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultUsersToCreate[nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix]), &parsedUsers)
	if err != nil {
		return &instance, errors.Wrapf(err, "cant umarshal %v", []byte(nexusDefaultUsersToCreate[nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix]))
	}

	newUser := map[string][]byte{}

	for _, userProperties := range parsedUsers {
		newUser["username"] = []byte(userProperties["username"].(string))
		newUser["first_name"] = []byte(userProperties["first_name"].(string))
		newUser["last_name"] = []byte(userProperties["last_name"].(string))
		newUser["password"] = []byte(uniuri.New())
		newUserSecretName = fmt.Sprintf("%s-%s", instance.Name, newUser["username"])

		err = s.platformService.CreateSecret(instance, newUserSecretName, newUser)
		if err != nil {
			return &instance, errors.Wrapf(err, "failed to create %s secret", newUserSecretName)
		}

		err = s.platformService.CreateJenkinsServiceAccount(instance.Namespace, newUserSecretName)
		if err != nil {
			return &instance, errors.Wrapf(err, "failed to create Jenkins service account %s", newUserSecretName)
		}

		data, err := s.platformService.GetSecretData(instance.Namespace, newUserSecretName)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to get CI user credentials")
		}

		userProperties["password"] = string(data["password"])

		_, err = nexusClient.RunScript("setup-user", userProperties)
		if err != nil {
			return &instance, errors.Wrapf(err, "failed to create user %v ", userProperties["username"])
		}
	}

	_ = s.client.Update(context.TODO(), &instance)

	if instance.Spec.KeycloakSpec.Enabled {
		webURL, _, _, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to get route from cluster")
		}

		keycloakClient := keycloakV1Api.KeycloakClient{}
		keycloakClient.Name = instance.Name
		keycloakClient.Namespace = instance.Namespace
		keycloakClient.Spec.ClientId = instance.Name
		keycloakClient.Spec.Public = true
		keycloakClient.Spec.WebUrl = webURL
		keycloakClient.Spec.AudRequired = true

		if instance.Spec.KeycloakSpec.Realm != "" {
			keycloakClient.Spec.TargetRealm = instance.Spec.KeycloakSpec.Realm
		}

		err = s.platformService.CreateKeycloakClient(&keycloakClient)
		if err != nil {
			return &instance, nil
		}
	}

	err = s.createEDPComponent(instance)

	return &instance, err
}

func (s ServiceImpl) createEDPComponent(nexus v1alpha1.Nexus) error {
	url, err := s.getUrl(nexus)
	if err != nil {
		return err
	}
	icon, err := s.getIcon()
	if err != nil {
		return err
	}
	return s.platformService.CreateEDPComponentIfNotExist(nexus, *url, *icon)
}

func (s ServiceImpl) getUrl(nexus v1alpha1.Nexus) (*string, error) {
	url, _, _, err := s.platformService.GetExternalUrl(nexus.Namespace, nexus.Name)
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (s ServiceImpl) getIcon() (*string, error) {
	p, err := platformHelper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, err
	}
	fp := fmt.Sprintf("%v/%v", p, nexusIcon)
	f, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	return &encoded, nil
}

// Configure performs self-configuration of Nexus
func (s ServiceImpl) Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, bool, error) {
	adminSecret := map[string][]byte{
		"user":     []byte(nexusDefaultSpec.NexusDefaultAdminUser),
		"password": []byte(nexusDefaultSpec.NexusDefaultAdminPassword),
	}

	err := s.platformService.CreateSecret(instance, instance.Name+"-admin-password", adminSecret)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to create Secret")
	}

	u, err := s.getNexusRestApiUrl(instance)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get Nexus REST API URL")
	}

	nexusPassword, err := s.getNexusAdminPassword(instance)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get Nexus admin password from secret")
	}

	nexusClient := s.clientBuilder(u, nexusDefaultSpec.NexusDefaultAdminUser, nexusPassword)

	if nexusApiIsReady, _, err := nexusClient.IsNexusRestApiReady(); err != nil {
		return &instance, false, errors.Wrap(err, "checking if Nexus REST API is ready has been failed")
	} else if !nexusApiIsReady {
		log.Info("Nexus REST API is not ready for configuration yet",
			"Namespace", instance.Namespace, "Name", instance.Name)
		return &instance, false, nil
	}

	nexusDefaultScriptsToCreate, err := s.platformService.GetConfigMapData(instance.Namespace,
		fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultScriptsConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	err = nexusClient.DeclareDefaultScripts(nexusDefaultScriptsToCreate)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to upload default scripts")
	}

	defaultScriptsAreDeclared, err := nexusClient.AreDefaultScriptsDeclared(nexusDefaultScriptsToCreate)
	if !defaultScriptsAreDeclared || err != nil {
		return &instance, false, errors.Wrap(err, "default scripts are not uploaded yet")
	}

	if nexusPassword == nexusDefaultSpec.NexusDefaultAdminPassword {
		updatePasswordParameters := map[string]interface{}{"new_password": uniuri.New()}

		nexusAdminPassword, err := s.platformService.GetSecret(instance.Namespace, instance.Name+"-admin-password")
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to get Nexus admin secret to update")
		}

		nexusAdminPassword.Data["password"] = []byte(updatePasswordParameters["new_password"].(string))
		err = s.platformService.UpdateSecret(nexusAdminPassword)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to update Nexus admin secret with new password")
		}

		_, err = nexusClient.RunScript("update-admin-password", updatePasswordParameters)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to update admin password")
		}

		passwordString := string(nexusAdminPassword.Data["password"])

		nexusClient = s.clientBuilder(u, nexusDefaultSpec.NexusDefaultAdminUser, passwordString)
	}
	nexusDefaultTasksToCreate, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	var parsedTasks []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultTasksToCreate[nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix]), &parsedTasks)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "cant unmarshal %v", []byte(nexusDefaultTasksToCreate[nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix]))
	}
	for _, taskParameters := range parsedTasks {
		_, err = nexusClient.RunScript("create-task", taskParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create task %v", taskParameters["name"])
		}
	}

	var emptyParameter map[string]interface{}
	_, err = nexusClient.RunScript("disable-outreach-capability", emptyParameter)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to run disable-outreach-capability scripts")
	}

	nexusCapabilities, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, "default-capabilities"))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	var nexusParsedCapabilities []map[string]interface{}
	err = json.Unmarshal([]byte(nexusCapabilities["default-capabilities"]), &nexusParsedCapabilities)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "Cant unmarshal %v", []byte(nexusCapabilities["default-capabilities"]))
	}

	for _, capability := range nexusParsedCapabilities {
		_, err = nexusClient.RunScript("setup-capability", capability)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to install default capabilities")
		}
	}

	enabledRealms := []map[string]interface{}{
		{"name": "NuGetApiKey"},
	}
	for _, realmName := range enabledRealms {
		_, err = nexusClient.RunScript("enable-realm", realmName)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to enable %v realm", enabledRealms)
		}
	}

	nexusDefaultRolesToCreate, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default roles from Config Map")
	}

	var parsedRoles []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultRolesToCreate[nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix]), &parsedRoles)

	if err != nil {
		return &instance, false, errors.Wrapf(err, "cant unmarshal %v", []byte(nexusDefaultRolesToCreate[nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix]))
	}

	for _, roleParameters := range parsedRoles {
		_, err := nexusClient.RunScript("setup-role", roleParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create role %v", roleParameters["name"])
		}
	}

	// Creating blob storage configuration from config map
	blobsConfig, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-blobs", instance.Name))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to get data from ConfigMap %v-blobs", instance.Name)
	}

	var parsedBlobsConfig []map[string]interface{}
	err = json.Unmarshal([]byte(blobsConfig["blobs"]), &parsedBlobsConfig)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to unmarshal blob ConfigMap")
	}

	for _, blob := range parsedBlobsConfig {
		_, err := nexusClient.RunScript("create-blobstore", blob)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create blob store %v", blob["name"])
		}
	}

	// Creating repositoriesToCreate from config map
	reposToCreate, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to get data from ConfigMap %v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix)
	}

	var parsedReposToCreate []map[string]interface{}
	err = json.Unmarshal([]byte(reposToCreate[nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix]), &parsedReposToCreate)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to unmarshal %v-%v ConfigMap", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix)
	}

	for _, repositoryToCreate := range parsedReposToCreate {
		repositoryName := repositoryToCreate["name"].(string)
		repositoryType := repositoryToCreate["repositoryType"].(string)
		_, err := nexusClient.RunScript(fmt.Sprintf("create-repo-%v", repositoryType), repositoryToCreate)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create repository %v", repositoryName)
		}
	}

	reposToDelete, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to get data from ConfigMap %v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix)
	}

	var parsedReposToDelete []map[string]interface{}
	err = json.Unmarshal([]byte(reposToDelete[nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix]), &parsedReposToDelete)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to unmarshal %v-%v ConfigMap", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix)
	}

	for _, repositoryToDelete := range parsedReposToDelete {
		_, err := nexusClient.RunScript("delete-repo", repositoryToDelete)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to delete repository %v", repositoryToDelete)
		}
	}

	for _, user := range instance.Spec.Users {
		setupUserParameters := map[string]interface{}{
			"username":   user.Username,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"email":      user.Email,
			"password":   uniuri.New(),
			"roles":      user.Roles,
		}

		_, err = nexusClient.RunScript("setup-user", setupUserParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create user %v", user.Username)
		}
	}

	if _, err := nexusClient.RunScript("setup-anonymous-access", map[string]interface{}{
		"anonymous_access": false,
	}); err != nil {
		return nil, false, errors.Wrap(err, "unable to setup anon access for nexus")
	}

	return &instance, true, nil
}
