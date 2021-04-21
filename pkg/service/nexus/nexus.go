package nexus

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/epam/edp-nexus-operator/v2/pkg/controller/helper"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/dchest/uniuri"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	keycloakV1Api "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/pkg/controller/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
	"github.com/pkg/errors"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = ctrl.Log.WithName("nexus_service")

const (
	imgFolder = "img"
	nexusIcon = "nexus.svg"
)

// NexusService interface for Nexus EDP component
type NexusService interface {
	Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, bool, error)
	ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error)
}

// NewNexusService function that returns NexusService implementation
func NewNexusService(platformService platform.PlatformService, client client.Client, scheme *runtime.Scheme) NexusService {
	return NexusServiceImpl{
		platformService: platformService,
		client:          client,
		keycloakHelper:  keycloakHelper.MakeHelper(client, scheme),
	}
}

// NexusServiceImpl struct fo Nexus EDP Component
type NexusServiceImpl struct {
	platformService platform.PlatformService
	client          client.Client
	nexusClient     nexus.NexusClient
	keycloakHelper  *keycloakHelper.Helper
}

// IsDeploymentReady check if deployment for Nexus is ready
func (n NexusServiceImpl) IsDeploymentReady(instance v1alpha1.Nexus) (*bool, error) {
	return n.platformService.IsDeploymentReady(instance)
}

func (n NexusServiceImpl) getNexusRestApiUrl(instance v1alpha1.Nexus) (string, error) {
	basePath := ""
	if len(instance.Spec.BasePath) > 0 {
		basePath = fmt.Sprintf("/%v", instance.Spec.BasePath)
	}
	u := fmt.Sprintf("http://%v.%v:%v%v/%v", instance.Name, instance.Namespace, nexusDefaultSpec.NexusPort, basePath, nexusDefaultSpec.NexusRestApiUrlPath)
	if !helper.RunningInCluster() {
		eu, _, _, err := n.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get Route for %v/%v", instance.Namespace, instance.Name)
		}
		u = fmt.Sprintf("%v/%v", eu, nexusDefaultSpec.NexusRestApiUrlPath)
	}
	return u, nil
}

func (n NexusServiceImpl) getNexusAdminPassword(instance v1alpha1.Nexus) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	nexusAdminCredentials, err := n.platformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
	}
	return string(nexusAdminCredentials["password"]), nil
}

func (n NexusServiceImpl) setAnnotation(instance *v1alpha1.Nexus, key string, value string) {
	if len(instance.Annotations) == 0 {
		instance.ObjectMeta.Annotations = map[string]string{
			key: value,
		}
	} else {
		instance.ObjectMeta.Annotations[key] = value
	}
}

// Integration performs integration Nexus with other EDP components
func (n NexusServiceImpl) Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {

	if instance.Spec.KeycloakSpec.Enabled {
		keycloakClient, err := n.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to get Keycloak client data!")
		}

		keycloakRealm, err := n.keycloakHelper.GetOwnerKeycloakRealm(keycloakClient.ObjectMeta)
		if err != nil {
			return &instance, nil
		}

		if keycloakRealm == nil {
			return &instance, errors.New("Keycloak Realm CR in not created yet!")
		}

		keycloak, err := n.keycloakHelper.GetOwnerKeycloak(keycloakRealm.ObjectMeta)
		if err != nil {
			errMsg := fmt.Sprintf("failed to get owner for %s/%s", keycloakClient.Namespace, keycloakClient.Name)
			return &instance, errors.Wrap(err, errMsg)
		}

		if keycloak == nil {
			return &instance, errors.New("Keycloak CR is not created yet")
		}

		_, host, scheme, err := n.platformService.GetExternalUrl(instance.Namespace, instance.Name)
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
		secret := fmt.Sprintf("--client-secret=42")
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
			"--resources=uri=/*|roles=developer,administrator|require-any-role=true",
		)

		err = n.platformService.AddKeycloakProxyToDeployConf(instance, proxyConfig)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to add Keycloak proxy")
		}

		keyCloakProxyPort := coreV1Api.ServicePort{
			Name:       "keycloak-proxy",
			Port:       nexusDefaultSpec.NexusKeycloakProxyPort,
			Protocol:   coreV1Api.ProtocolTCP,
			TargetPort: intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort},
		}
		if err = n.platformService.AddPortToService(instance, keyCloakProxyPort); err != nil {
			return &instance, errors.Wrap(err, "failed to add Keycloak proxy port to service")
		}

		if err = n.platformService.UpdateExternalTargetPath(instance, intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort}); err != nil {
			return &instance, errors.Wrap(err, "failed to update target port in Route")
		}
	} else {
		log.V(1).Info("Keycloak integration not enabled.")
	}

	return &instance, nil
}

// ExposeConfiguration performs exposing Nexus configuration for other EDP components
func (n NexusServiceImpl) ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	u, err := n.getNexusRestApiUrl(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "failed to get Nexus REST API URL")
	}

	nexusPassword, err := n.getNexusAdminPassword(instance)
	if err != nil {
		return &instance, errors.Wrap(err, "failed to get Nexus admin password from secret")
	}

	err = n.nexusClient.InitNewRestClient(&instance, u, nexusDefaultSpec.NexusDefaultAdminUser, nexusPassword)
	if err != nil {
		return &instance, errors.Wrap(err, "failed to initialize Nexus client")
	}

	nexusDefaultUsersToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix))
	if err != nil {
		return &instance, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	var newUserSecretName string
	var parsedUsers []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultUsersToCreate[nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix]), &parsedUsers)

	newUser := map[string][]byte{}

	for _, userProperties := range parsedUsers {
		newUser["username"] = []byte(userProperties["username"].(string))
		newUser["first_name"] = []byte(userProperties["first_name"].(string))
		newUser["last_name"] = []byte(userProperties["last_name"].(string))
		newUser["password"] = []byte(uniuri.New())
		newUserSecretName = fmt.Sprintf("%s-%s", instance.Name, newUser["username"])

		err = n.platformService.CreateSecret(instance, newUserSecretName, newUser)
		if err != nil {
			return &instance, errors.Wrapf(err, "failed to create %s secret", newUserSecretName)
		}

		err := n.platformService.CreateJenkinsServiceAccount(instance.Namespace, newUserSecretName)
		if err != nil {
			return &instance, errors.Wrapf(err, "failed to create Jenkins service account %s", newUserSecretName)
		}

		data, err := n.platformService.GetSecretData(instance.Namespace, newUserSecretName)
		if err != nil {
			return &instance, errors.Wrap(err, "failed to get CI user credentials")
		}

		userProperties["password"] = string(data["password"])

		_, err = n.nexusClient.RunScript("setup-user", userProperties)
		if err != nil {
			return &instance, errors.Wrapf(err, "failed to create user %v ", userProperties["username"])
		}
	}

	_ = n.client.Update(context.TODO(), &instance)

	if instance.Spec.KeycloakSpec.Enabled {
		webURL, _, _, err := n.platformService.GetExternalUrl(instance.Namespace, instance.Name)
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

		err = n.platformService.CreateKeycloakClient(&keycloakClient)
		if err != nil {
			return &instance, nil
		}
	}

	err = n.createEDPComponent(instance)

	return &instance, err
}

func (n NexusServiceImpl) createEDPComponent(nexus v1alpha1.Nexus) error {
	url, err := n.getUrl(nexus)
	if err != nil {
		return err
	}
	icon, err := n.getIcon()
	if err != nil {
		return err
	}
	return n.platformService.CreateEDPComponentIfNotExist(nexus, *url, *icon)
}

func (n NexusServiceImpl) getUrl(nexus v1alpha1.Nexus) (*string, error) {
	url, _, _, err := n.platformService.GetExternalUrl(nexus.Namespace, nexus.Name)
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (n NexusServiceImpl) getIcon() (*string, error) {
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
func (n NexusServiceImpl) Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, bool, error) {
	adminSecret := map[string][]byte{
		"user":     []byte(nexusDefaultSpec.NexusDefaultAdminUser),
		"password": []byte(nexusDefaultSpec.NexusDefaultAdminPassword),
	}

	err := n.platformService.CreateSecret(instance, instance.Name+"-admin-password", adminSecret)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to create Secret")
	}

	u, err := n.getNexusRestApiUrl(instance)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get Nexus REST API URL")
	}

	nexusPassword, err := n.getNexusAdminPassword(instance)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get Nexus admin password from secret")
	}

	err = n.nexusClient.InitNewRestClient(&instance, u, nexusDefaultSpec.NexusDefaultAdminUser, nexusPassword)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to initialize Nexus client")
	}

	if nexusApiIsReady, _, err := n.nexusClient.IsNexusRestApiReady(); err != nil {
		return &instance, false, errors.Wrap(err, "checking if Nexus REST API is ready has been failed")
	} else if !nexusApiIsReady {
		log.Info("Nexus REST API is not ready for configuration yet",
			"Namespace", instance.Namespace, "Name", instance.Name)
		return &instance, false, nil
	}

	nexusDefaultScriptsToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultScriptsConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	err = n.nexusClient.DeclareDefaultScripts(nexusDefaultScriptsToCreate)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to upload default scripts")
	}

	defaultScriptsAreDeclared, err := n.nexusClient.AreDefaultScriptsDeclared(nexusDefaultScriptsToCreate)
	if !defaultScriptsAreDeclared || err != nil {
		return &instance, false, errors.Wrap(err, "default scripts are not uploaded yet")
	}

	if nexusPassword == nexusDefaultSpec.NexusDefaultAdminPassword {
		updatePasswordParameters := map[string]interface{}{"new_password": uniuri.New()}

		nexusAdminPassword, err := n.platformService.GetSecret(instance.Namespace, instance.Name+"-admin-password")
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to get Nexus admin secret to update")
		}

		nexusAdminPassword.Data["password"] = []byte(updatePasswordParameters["new_password"].(string))
		err = n.platformService.UpdateSecret(nexusAdminPassword)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to update Nexus admin secret with new password")
		}

		_, err = n.nexusClient.RunScript("update-admin-password", updatePasswordParameters)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to update admin password")
		}

		passwordString := string(nexusAdminPassword.Data["password"])

		err = n.nexusClient.InitNewRestClient(&instance, u, nexusDefaultSpec.NexusDefaultAdminUser, passwordString)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to initialize Nexus client")
		}
	}
	nexusDefaultTasksToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	var parsedTasks []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultTasksToCreate[nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix]), &parsedTasks)
	for _, taskParameters := range parsedTasks {
		_, err = n.nexusClient.RunScript("create-task", taskParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create task %v", taskParameters["name"])
		}
	}

	var emptyParameter map[string]interface{}
	_, err = n.nexusClient.RunScript("disable-outreach-capability", emptyParameter)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to run disable-outreach-capability scripts")
	}

	nexusCapabilities, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, "default-capabilities"))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default tasks from Config Map")
	}

	var nexusParsedCapabilities []map[string]interface{}
	err = json.Unmarshal([]byte(nexusCapabilities["default-capabilities"]), &nexusParsedCapabilities)

	for _, capability := range nexusParsedCapabilities {
		_, err = n.nexusClient.RunScript("setup-capability", capability)
		if err != nil {
			return &instance, false, errors.Wrap(err, "failed to install default capabilities")
		}
	}

	enabledRealms := []map[string]interface{}{
		{"name": "NuGetApiKey"},
	}
	for _, realmName := range enabledRealms {
		_, err = n.nexusClient.RunScript("enable-realm", realmName)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to enable %v realm", enabledRealms)
		}
	}

	nexusDefaultRolesToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to get default roles from Config Map")
	}

	var parsedRoles []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultRolesToCreate[nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix]), &parsedRoles)
	for _, roleParameters := range parsedRoles {
		_, err := n.nexusClient.RunScript("setup-role", roleParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create role %v", roleParameters["name"])
		}
	}

	// Creating blob storage configuration from config map
	blobsConfig, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-blobs", instance.Name))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to get data from ConfigMap %v-blobs", instance.Name)
	}

	var parsedBlobsConfig []map[string]interface{}
	err = json.Unmarshal([]byte(blobsConfig["blobs"]), &parsedBlobsConfig)
	if err != nil {
		return &instance, false, errors.Wrap(err, "failed to unmarshal blob ConfigMap")
	}

	for _, blob := range parsedBlobsConfig {
		_, err := n.nexusClient.RunScript("create-blobstore", blob)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create blob store %v", blob["name"])
		}
	}

	// Creating repositoriesToCreate from config map
	reposToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix))
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
		_, err := n.nexusClient.RunScript(fmt.Sprintf("create-repo-%v", repositoryType), repositoryToCreate)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create repository %v", repositoryName)
		}
	}

	reposToDelete, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to get data from ConfigMap %v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix)
	}

	var parsedReposToDelete []map[string]interface{}
	err = json.Unmarshal([]byte(reposToDelete[nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix]), &parsedReposToDelete)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "failed to unmarshal %v-%v ConfigMap", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix)
	}

	for _, repositoryToDelete := range parsedReposToDelete {
		_, err := n.nexusClient.RunScript("delete-repo", repositoryToDelete)
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

		_, err = n.nexusClient.RunScript("setup-user", setupUserParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "failed to create user %v", user.Username)
		}
	}

	return &instance, true, nil
}
