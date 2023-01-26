package nexus

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dchest/uniuri"
	coreV1Api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	platformHelper "github.com/epam/edp-jenkins-operator/v2/pkg/service/platform/helper"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/controllers/helper"
	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	"github.com/epam/edp-nexus-operator/v2/controllers/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
)

var (
	log                          = ctrl.Log.WithName("nexus_service")
	errCantGetOwnerKeycloakRealm = errors.New("failed to get owner keycloak realm")
)

const (
	imgFolder                                 = "img"
	nexusIcon                                 = "nexus.svg"
	crHTTPFormatString                        = "http://%v.%v:%v%v/%v"
	crUpstreamURLFormatString                 = "--upstream-url=http://127.0.0.1:%v"
	crRedirectionURLFormatString              = "--redirection-url=%v"
	crClientIDFormatString                    = "--client-id=%v"
	crDiscoveryURLFormatString                = "--discovery-url=%s/auth/realms/%s"
	crUsernameKey                             = "username"
	crFirstNameKey                            = "first_name"
	crLastNameKey                             = "last_name"
	crEmailKey                                = "email"
	crPasswordKey                             = "password"
	crRolesKey                                = "roles"
	crNameKey                                 = "name"
	crFailedToGetDefaultTasksFromConfigMapLog = "failed to get default tasks from Config Map: %w"
	crStringStringFormat                      = "%s-%s"
)

// Service interface for Nexus EDP component.
type Service interface {
	Configure(instance *nexusApi.Nexus) (*nexusApi.Nexus, bool, error)
	ExposeConfiguration(ctx context.Context, instance *nexusApi.Nexus) (*nexusApi.Nexus, error)
	Integration(instance *nexusApi.Nexus) (*nexusApi.Nexus, error)
	IsDeploymentReady(instance *nexusApi.Nexus) (*bool, error)
	ClientForNexusChild(ctx context.Context, child Child) (*nexus.Client, error)
}

type Client interface {
	IsNexusRestApiReady() (bool, int, error)
	DeclareDefaultScripts(listOfScripts map[string]string) error
	AreDefaultScriptsDeclared(listOfScripts map[string]string) (bool, error)
	RunScript(scriptName string, parameters map[string]interface{}) ([]byte, error)
}

type keycloakData struct {
	keycloakClient *keycloakApi.KeycloakClient
	keycloakRealm  *keycloakApi.KeycloakRealm
	keycloak       *keycloakApi.Keycloak
}

// NewService function that returns NexusService implementation.
func NewService(platformService platform.PlatformService, c client.Client, scheme *runtime.Scheme) Service {
	return ServiceImpl{
		platformService:      platformService,
		client:               c,
		keycloakHelper:       keycloakHelper.MakeHelper(c, scheme, ctrl.Log.WithName("nexus_service")),
		runningInClusterFunc: helper.RunningInCluster,
		clientBuilder: func(url string, user string, password string) Client {
			return nexus.Init(url, user, password)
		},
	}
}

// ServiceImpl struct fo Nexus EDP Component.
type ServiceImpl struct {
	platformService      platform.PlatformService
	client               client.Client
	keycloakHelper       *keycloakHelper.Helper
	runningInClusterFunc func() bool
	clientBuilder        func(url string, user string, password string) Client
}

// IsDeploymentReady check if deployment for Nexus is ready.
func (s ServiceImpl) IsDeploymentReady(instance *nexusApi.Nexus) (*bool, error) {
	ready, err := s.platformService.IsDeploymentReady(instance)
	if err != nil {
		return getBoolP(false), fmt.Errorf("failed to check if deployment is ready: %w", err)
	}

	return ready, nil
}

func (s ServiceImpl) getNexusRestApiUrl(instance *nexusApi.Nexus) (string, error) {
	if s.runningInClusterFunc == nil {
		return "", fmt.Errorf("missing runningInClusterFunc")
	}

	basePath := ""
	if len(instance.Spec.BasePath) > 0 {
		basePath = fmt.Sprintf("/%v", instance.Spec.BasePath)
	}

	URL := fmt.Sprintf(
		crHTTPFormatString,
		instance.Name,
		instance.Namespace,
		nexusDefaultSpec.NexusPort,
		basePath,
		nexusDefaultSpec.NexusRestApiUrlPath,
	)

	if !s.runningInClusterFunc() {
		eu, _, _, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return "", fmt.Errorf("failed to get Route for %v/%v: %w", instance.Namespace, instance.Name, err)
		}

		URL = fmt.Sprintf("%v/%v", eu, nexusDefaultSpec.NexusRestApiUrlPath)
	}

	return URL, nil
}

func (s ServiceImpl) getNexusAdminPassword(instance *nexusApi.Nexus) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)

	nexusAdminCredentials, err := s.platformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s for %s/%s: %w", secretName, instance.Namespace, instance.Name, err)
	}

	return string(nexusAdminCredentials[crPasswordKey]), nil
}

// Integration performs integration Nexus with other EDP components.
func (s ServiceImpl) Integration(instance *nexusApi.Nexus) (*nexusApi.Nexus, error) {
	if !instance.Spec.KeycloakSpec.Enabled {
		log.V(1).Info("Keycloak integration not enabled.")

		return instance, nil
	}

	keycloakDataInstance, err := s.buildKeycloak(instance)
	if err != nil {
		if errors.Is(err, errCantGetOwnerKeycloakRealm) {
			return instance, nil
		}

		return instance, err
	}

	_, host, scheme, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
	if err != nil {
		return instance, fmt.Errorf("failed to get route: %w", err)
	}

	var proxyConfig []string

	upstreamUrl := fmt.Sprintf(crUpstreamURLFormatString, nexusDefaultSpec.NexusPort)

	if instance.Spec.BasePath == "" {
		upstreamUrl = fmt.Sprintf("%v/%v", upstreamUrl, instance.Spec.BasePath)
		proxyConfig = append(proxyConfig, fmt.Sprintf("--base-uri=/%v", instance.Spec.BasePath))
	}

	listen := fmt.Sprintf("--listen=0.0.0.0:%d", nexusDefaultSpec.NexusKeycloakProxyPort)

	proxyConfig = append(
		proxyConfig,
		"--skip-openid-provider-tls-verify=true",
		fmt.Sprintf(crDiscoveryURLFormatString, keycloakDataInstance.keycloak.Spec.Url, keycloakDataInstance.keycloakRealm.Spec.RealmName),
		fmt.Sprintf(crClientIDFormatString, keycloakDataInstance.keycloakClient.Spec.ClientId),
		"--client-secret=42",
		listen,
		fmt.Sprintf(crRedirectionURLFormatString, fmt.Sprintf("%v://%v", scheme, host)),
		upstreamUrl,
	)

	if len(instance.Spec.KeycloakSpec.Roles) > 0 {
		proxyConfig = append(proxyConfig,
			fmt.Sprintf("--resources=uri=/*|roles=%s|require-any-role=true",
				strings.Join(instance.Spec.KeycloakSpec.Roles, ",")))
	}

	if err = s.platformService.AddKeycloakProxyToDeployConf(instance, proxyConfig); err != nil {
		return instance, fmt.Errorf("failed to add Keycloak proxy: %w", err)
	}

	keyCloakProxyPort := coreV1Api.ServicePort{
		Name:       "keycloak-proxy",
		Port:       nexusDefaultSpec.NexusKeycloakProxyPort,
		Protocol:   coreV1Api.ProtocolTCP,
		TargetPort: intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort},
	}
	if err = s.platformService.AddPortToService(instance, &keyCloakProxyPort); err != nil {
		return instance, fmt.Errorf("failed to add keycloak proxy port to service: %w", err)
	}

	if err = s.platformService.UpdateExternalTargetPath(instance, intstr.IntOrString{IntVal: nexusDefaultSpec.NexusKeycloakProxyPort}); err != nil {
		return instance, fmt.Errorf("failed to update target port in Route: %w", err)
	}

	return instance, nil
}

// ExposeConfiguration performs exposing Nexus configuration for other EDP components.
func (s ServiceImpl) ExposeConfiguration(ctx context.Context, instance *nexusApi.Nexus) (*nexusApi.Nexus, error) {
	u, err := s.getNexusRestApiUrl(instance)
	if err != nil {
		return instance, fmt.Errorf("failed to get Nexus REST API URL: %w", err)
	}

	nexusPassword, err := s.getNexusAdminPassword(instance)
	if err != nil {
		return instance, fmt.Errorf("failed to get Nexus admin password from secret: %w", err)
	}

	nexusClient := nexus.Init(u, nexusDefaultSpec.NexusDefaultAdminUser, nexusPassword)

	nexusDefaultUsersToCreate, err := s.platformService.GetConfigMapData(
		instance.Namespace,
		fmt.Sprintf(crStringStringFormat, instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix),
	)
	if err != nil {
		return instance, fmt.Errorf(crFailedToGetDefaultTasksFromConfigMapLog, err)
	}

	var parsedUsers []map[string]interface{}

	if err = json.Unmarshal(
		[]byte(nexusDefaultUsersToCreate[nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix]),
		&parsedUsers,
	); err != nil {
		return instance, fmt.Errorf(
			"cant unmarshal %v: %w",
			[]byte(nexusDefaultUsersToCreate[nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix]),
			err,
		)
	}

	newUser := map[string][]byte{}

	for _, userProperties := range parsedUsers {
		if propsErr := s.buildUserProps(newUser, userProperties); propsErr != nil {
			return instance, propsErr
		}

		if runErr := s.runScript(ctx, newUser, userProperties, instance, nexusClient); runErr != nil {
			return instance, runErr
		}
	}

	if updErr := s.client.Update(context.TODO(), instance); updErr != nil {
		log.Error(err, "failed to update nexus instance: %w")
	}

	if instance.Spec.KeycloakSpec.Enabled {
		if err = s.keycloakClient(instance); err != nil {
			return instance, fmt.Errorf("failed to create Keycloak client: %w", err)
		}

		return instance, nil
	}

	return instance, s.createEDPComponent(instance)
}

func (s ServiceImpl) createEDPComponent(n *nexusApi.Nexus) error {
	url, err := s.getUrl(n)
	if err != nil {
		return err
	}

	icon, err := getIcon()
	if err != nil {
		return err
	}

	if err = s.platformService.CreateEDPComponentIfNotExist(n, *url, *icon); err != nil {
		return fmt.Errorf("failed to create EDP component: %w", err)
	}

	return nil
}

func (s ServiceImpl) getUrl(n *nexusApi.Nexus) (*string, error) {
	url, _, _, err := s.platformService.GetExternalUrl(n.Namespace, n.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get external URL: %w", err)
	}

	return &url, nil
}

func getIcon() (*string, error) {
	p, err := platformHelper.CreatePathToTemplateDirectory(imgFolder)
	if err != nil {
		return nil, fmt.Errorf("failed to create image folder: %w", err)
	}

	fp := fmt.Sprintf("%v/%v", p, nexusIcon)

	f, err := os.Open(fp)
	if err != nil {
		return nil, fmt.Errorf("failed to open icon file: %w", err)
	}

	reader := bufio.NewReader(f)

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(content)

	return &encoded, nil
}

// Configure performs self-configuration of Nexus.
func (s ServiceImpl) Configure(instance *nexusApi.Nexus) (*nexusApi.Nexus, bool, error) {
	adminSecret := map[string][]byte{
		"user":        []byte(nexusDefaultSpec.NexusDefaultAdminUser),
		crPasswordKey: []byte(nexusDefaultSpec.NexusDefaultAdminPassword),
	}

	err := s.platformService.CreateSecret(instance, instance.Name+"-admin-password", adminSecret)
	if err != nil {
		return instance, false, fmt.Errorf("failed to create secret: %w", err)
	}

	u, err := s.getNexusRestApiUrl(instance)
	if err != nil {
		return instance, false, fmt.Errorf("failed to get Nexus REST API URL: %w", err)
	}

	nexusPassword, err := s.getNexusAdminPassword(instance)
	if err != nil {
		return instance, false, fmt.Errorf("failed to get Nexus admin password from secret: %w", err)
	}

	nexusClient := s.clientBuilder(u, nexusDefaultSpec.NexusDefaultAdminUser, nexusPassword)

	nexusApiIsReady, _, isReadyErr := nexusClient.IsNexusRestApiReady()
	if isReadyErr != nil {
		return instance, false, fmt.Errorf("checking if Nexus REST API is ready has been failed: %w", isReadyErr)
	}

	if !nexusApiIsReady {
		log.Info("Nexus REST API is not ready for configuration yet",
			"Namespace", instance.Namespace, "Name", instance.Name)
		return instance, false, nil
	}

	nexusDefaultScriptsToCreate, err := s.platformService.GetConfigMapData(
		instance.Namespace,
		fmt.Sprintf(crStringStringFormat, instance.Name, nexusDefaultSpec.NexusDefaultScriptsConfigMapPrefix))
	if err != nil {
		return instance, false, fmt.Errorf(crFailedToGetDefaultTasksFromConfigMapLog, err)
	}

	if declareErr := nexusClient.DeclareDefaultScripts(nexusDefaultScriptsToCreate); declareErr != nil {
		return instance, false, fmt.Errorf("failed to upload default scripts: %w", declareErr)
	}

	defaultScriptsAreDeclared, err := nexusClient.AreDefaultScriptsDeclared(nexusDefaultScriptsToCreate)
	if !defaultScriptsAreDeclared || err != nil {
		return instance, false, fmt.Errorf("default scripts declared - %v, error: %w", defaultScriptsAreDeclared, err)
	}

	if nexusPassword == nexusDefaultSpec.NexusDefaultAdminPassword {
		updatePasswordParameters := map[string]interface{}{"new_password": uniuri.New()}

		nexusAdminPassword, secretErr := s.platformService.GetSecret(instance.Namespace, instance.Name+"-admin-password")
		if secretErr != nil {
			return instance, false, fmt.Errorf("failed to get Nexus admin secret to update: %w", secretErr)
		}

		np, ok := updatePasswordParameters["new_password"].(string)
		if !ok {
			return instance, false, fmt.Errorf("invalid update password parameter type: %w", secretErr)
		}

		nexusAdminPassword.Data[crPasswordKey] = []byte(np)
		if updErr := s.platformService.UpdateSecret(nexusAdminPassword); updErr != nil {
			return instance, false, fmt.Errorf("failed to update Nexus admin secret with new password: %w", updErr)
		}

		if _, runErr := nexusClient.RunScript("update-admin-password", updatePasswordParameters); runErr != nil {
			return instance, false, fmt.Errorf("failed to update admin password: %w", runErr)
		}

		passwordString := string(nexusAdminPassword.Data[crPasswordKey])
		nexusClient = s.clientBuilder(u, nexusDefaultSpec.NexusDefaultAdminUser, passwordString)
	}

	nexusDefaultTasksToCreate, err := s.platformService.GetConfigMapData(
		instance.Namespace,
		fmt.Sprintf(crStringStringFormat, instance.Name, nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix))

	if err != nil {
		return instance, false, fmt.Errorf(crFailedToGetDefaultTasksFromConfigMapLog, err)
	}

	var parsedTasks []map[string]interface{}
	if err = json.Unmarshal(
		[]byte(nexusDefaultTasksToCreate[nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix]),
		&parsedTasks,
	); err != nil {
		return instance, false, fmt.Errorf(
			"failed to unmarshal %v: %w",
			[]byte(nexusDefaultTasksToCreate[nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix]),
			err,
		)
	}

	for _, taskParameters := range parsedTasks {
		_, err = nexusClient.RunScript("create-task", taskParameters)
		if err != nil {
			return instance, false, fmt.Errorf("failed to create task %v: %w", taskParameters[crNameKey], err)
		}
	}

	var emptyParameter map[string]interface{}

	_, err = nexusClient.RunScript("disable-outreach-capability", emptyParameter)
	if err != nil {
		return instance, false, fmt.Errorf("failed to run disable-outreach-capability scripts: %w", err)
	}

	nexusCapabilities, err := s.platformService.GetConfigMapData(
		instance.Namespace,
		fmt.Sprintf(crStringStringFormat, instance.Name, "default-capabilities"),
	)

	if err != nil {
		return instance, false, fmt.Errorf(crFailedToGetDefaultTasksFromConfigMapLog, err)
	}

	var nexusParsedCapabilities []map[string]interface{}
	if err = json.Unmarshal([]byte(nexusCapabilities["default-capabilities"]), &nexusParsedCapabilities); err != nil {
		return instance, false, fmt.Errorf(
			"failed to unmarshal %v: %w",
			[]byte(nexusCapabilities["default-capabilities"]),
			err,
		)
	}

	for _, capability := range nexusParsedCapabilities {
		if _, err = nexusClient.RunScript("setup-capability", capability); err != nil {
			return instance, false, fmt.Errorf("failed to install default capabilities: %w", err)
		}
	}

	enabledRealms := []map[string]interface{}{{crNameKey: "NuGetApiKey"}}
	for _, realmName := range enabledRealms {
		if _, err = nexusClient.RunScript("enable-realm", realmName); err != nil {
			return instance, false, fmt.Errorf("failed to enable %v realm: %w", enabledRealms, err)
		}
	}

	nexusDefaultRolesToCreate, err := s.platformService.GetConfigMapData(
		instance.Namespace,
		fmt.Sprintf(crStringStringFormat, instance.Name, nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix))
	if err != nil {
		return instance, false, fmt.Errorf("failed to get default roles from Config Mapr: %w", err)
	}

	var parsedRoles []map[string]interface{}
	if err = json.Unmarshal(
		[]byte(nexusDefaultRolesToCreate[nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix]),
		&parsedRoles,
	); err != nil {
		return instance, false, fmt.Errorf(
			"failed to unmarshal %v: %w",
			[]byte(nexusDefaultRolesToCreate[nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix]),
			err,
		)
	}

	for _, roleParameters := range parsedRoles {
		if _, err = nexusClient.RunScript("setup-role", roleParameters); err != nil {
			return instance, false, fmt.Errorf("failed to create role %v: %w", roleParameters[crNameKey], err)
		}
	}

	// Creating blob storage configuration from config map
	blobsConfig, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-blobs", instance.Name))
	if err != nil {
		return instance, false, fmt.Errorf("failed to get data from ConfigMap %v-blobs: %w", instance.Name, err)
	}

	var parsedBlobsConfig []map[string]interface{}
	if err = json.Unmarshal([]byte(blobsConfig["blobs"]), &parsedBlobsConfig); err != nil {
		return instance, false, fmt.Errorf("failed to unmarshal blob ConfigMap: %w", err)
	}

	for _, blob := range parsedBlobsConfig {
		if _, err = nexusClient.RunScript("create-blobstore", blob); err != nil {
			return instance, false, fmt.Errorf("failed to create blob store %v: %w", blob[crNameKey], err)
		}
	}

	// Creating repositoriesToCreate from config map
	reposToCreate, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf(crStringStringFormat, instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix))
	if err != nil {
		return instance, false, fmt.Errorf(
			"failed to get data from ConfigMap %v-%v: %w",
			instance.Name,
			nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix,
			err,
		)
	}

	var parsedReposToCreate []map[string]interface{}
	if err = json.Unmarshal(
		[]byte(reposToCreate[nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix]),
		&parsedReposToCreate,
	); err != nil {
		return instance, false, fmt.Errorf(
			"failed to unmarshal %v-%v ConfigMap: %w",
			instance.Name,
			nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix,
			err,
		)
	}

	for _, repositoryToCreate := range parsedReposToCreate {
		repositoryName, ok := repositoryToCreate[crNameKey].(string)
		if !ok {
			return instance, false, fmt.Errorf("invalid repository \"name\" variable type")
		}

		repositoryType, ok := repositoryToCreate["repositoryType"].(string)
		if !ok {
			return instance, false, fmt.Errorf("invalid \"repositoryType\" variable type")
		}

		if _, err = nexusClient.RunScript(
			fmt.Sprintf("create-repo-%v", repositoryType),
			repositoryToCreate,
		); err != nil {
			return instance, false, fmt.Errorf("failed to create repository %v: %w", repositoryName, err)
		}
	}

	reposToDelete, err := s.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf(crStringStringFormat, instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix))
	if err != nil {
		return instance, false, fmt.Errorf("failed to get data from ConfigMap %v-%v: %w", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix, err)
	}

	var parsedReposToDelete []map[string]interface{}
	if err = json.Unmarshal(
		[]byte(reposToDelete[nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix]),
		&parsedReposToDelete,
	); err != nil {
		return instance, false, fmt.Errorf(
			"failed to unmarshal %v-%v ConfigMap: %w",
			instance.Name,
			nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix,
			err,
		)
	}

	for _, repositoryToDelete := range parsedReposToDelete {
		if _, err = nexusClient.RunScript("delete-repo", repositoryToDelete); err != nil {
			return instance, false, fmt.Errorf("failed to delete repository %v: %w", repositoryToDelete, err)
		}
	}

	for _, user := range instance.Spec.Users {
		setupUserParameters := map[string]interface{}{
			crUsernameKey:  user.Username,
			crFirstNameKey: user.FirstName,
			crLastNameKey:  user.LastName,
			crEmailKey:     user.Email,
			crPasswordKey:  uniuri.New(),
			crRolesKey:     user.Roles,
		}

		if _, err = nexusClient.RunScript("setup-user", setupUserParameters); err != nil {
			return instance, false, fmt.Errorf("failed to create user %v: %w", user.Username, err)
		}
	}

	if _, err = nexusClient.RunScript("setup-anonymous-access", map[string]interface{}{
		"anonymous_access": false,
	}); err != nil {
		return nil, false, fmt.Errorf("failed to setup anon access for nexus: %w", err)
	}

	return instance, true, nil
}

func (s ServiceImpl) jenkinsEnabled(ctx context.Context, namespace string) bool {
	jenkinsList := &jenkinsApi.JenkinsList{}
	if err := s.client.List(ctx, jenkinsList, &client.ListOptions{Namespace: namespace}); err != nil {
		return false
	}

	return len(jenkinsList.Items) != 0
}

func getBoolP(val bool) *bool {
	return &val
}

func (s ServiceImpl) buildKeycloak(instance *nexusApi.Nexus) (*keycloakData, error) {
	keycloakClient, err := s.platformService.GetKeycloakClient(instance.Name, instance.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get keycloak client data: %w", err)
	}

	keycloakRealm, err := s.keycloakHelper.GetOwnerKeycloakRealm(&keycloakClient.ObjectMeta)
	if err != nil {
		return nil, errCantGetOwnerKeycloakRealm
	}

	if keycloakRealm == nil {
		return nil, errors.New("keycloak realm CR in not created yet")
	}

	keycloak, err := s.keycloakHelper.GetOwnerKeycloak(&keycloakRealm.ObjectMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner for %s/%s: %w", keycloakClient.Namespace, keycloakClient.Name, err)
	}

	if keycloak == nil {
		return nil, errors.New("keycloak CR is not created yet")
	}

	return &keycloakData{
		keycloakClient: &keycloakClient,
		keycloakRealm:  keycloakRealm,
		keycloak:       keycloak,
	}, nil
}

func (s ServiceImpl) keycloakClient(instance *nexusApi.Nexus) error {
	webURL, _, _, getErr := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
	if getErr != nil {
		return fmt.Errorf("failed to get route from cluster: %w", getErr)
	}

	keycloakClient := keycloakApi.KeycloakClient{}
	keycloakClient.Name = instance.Name
	keycloakClient.Namespace = instance.Namespace
	keycloakClient.Spec.ClientId = instance.Name
	keycloakClient.Spec.Public = true
	keycloakClient.Spec.WebUrl = webURL
	keycloakClient.Spec.DefaultClientScopes = []string{"edp"}

	if instance.Spec.KeycloakSpec.Realm != "" {
		keycloakClient.Spec.TargetRealm = instance.Spec.KeycloakSpec.Realm
	}

	if err := s.platformService.CreateKeycloakClient(&keycloakClient); err != nil {
		return nil
	}

	return nil
}

func (ServiceImpl) buildUserProps(newUser map[string][]byte, userProperties map[string]interface{}) error {
	uName, ok := userProperties[crUsernameKey].(string)
	if !ok {
		return fmt.Errorf("failed to get username from properties")
	}

	fName, ok := userProperties[crFirstNameKey].(string)
	if !ok {
		return fmt.Errorf("failed to get first name from properties")
	}

	lName, ok := userProperties[crLastNameKey].(string)
	if !ok {
		return fmt.Errorf("failed to get last name from properties")
	}

	newUser[crUsernameKey] = []byte(uName)
	newUser[crFirstNameKey] = []byte(fName)
	newUser[crLastNameKey] = []byte(lName)
	newUser[crPasswordKey] = []byte(uniuri.New())

	return nil
}

func (s ServiceImpl) runScript(
	ctx context.Context, newUser map[string][]byte, userProperties map[string]interface{},
	instance *nexusApi.Nexus, nexusClient Client) error {
	newUserSecretName := fmt.Sprintf("%s-%s", instance.Name, newUser[crUsernameKey])
	if err := s.platformService.CreateSecret(instance, newUserSecretName, newUser); err != nil {
		return fmt.Errorf("failed to create secret -%s: %w", newUserSecretName, err)
	}

	if s.jenkinsEnabled(ctx, instance.Namespace) {
		if err := s.platformService.CreateJenkinsServiceAccount(instance.Namespace, newUserSecretName); err != nil {
			return fmt.Errorf("failed to create Jenkins service account - %s: %w", newUserSecretName, err)
		}
	}

	data, getErr := s.platformService.GetSecretData(instance.Namespace, newUserSecretName)
	if getErr != nil {
		return fmt.Errorf("failed to get CI user credentials: %w", getErr)
	}

	userProperties[crPasswordKey] = string(data[crPasswordKey])

	if _, err := nexusClient.RunScript("setup-user", userProperties); err != nil {
		return fmt.Errorf("failed to create user - %s: %w", userProperties[crUsernameKey], err)
	}

	return nil
}
