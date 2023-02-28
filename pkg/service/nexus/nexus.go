package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/dchest/uniuri"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jenkinsApi "github.com/epam/edp-jenkins-operator/v2/pkg/apis/v2/v1"
	keycloakHelper "github.com/epam/edp-keycloak-operator/controllers/helper"
	nexusApi "github.com/epam/edp-nexus-operator/v2/api/v1"
	"github.com/epam/edp-nexus-operator/v2/controllers/helper"
	"github.com/epam/edp-nexus-operator/v2/pkg/client/nexus"
	nexusDefaultSpec "github.com/epam/edp-nexus-operator/v2/pkg/service/nexus/spec"
	"github.com/epam/edp-nexus-operator/v2/pkg/service/platform"
)

var (
	log = ctrl.Log.WithName("nexus_service")
)

const (
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
	IsDeploymentReady(instance *nexusApi.Nexus) (*bool, error)
	ClientForNexusChild(ctx context.Context, child Child) (*nexus.Client, error)
}

type Client interface {
	IsNexusRestApiReady() (bool, int, error)
	DeclareDefaultScripts(listOfScripts map[string]string) error
	AreDefaultScriptsDeclared(listOfScripts map[string]string) (bool, error)
	RunScript(scriptName string, parameters map[string]interface{}) ([]byte, error)
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

	if !s.runningInClusterFunc() {
		externalURL, _, _, err := s.platformService.GetExternalUrl(instance.Namespace, instance.Name)
		if err != nil {
			return "", fmt.Errorf("failed to get Route for %v/%v: %w", instance.Namespace, instance.Name, err)
		}

		URL, err := url.JoinPath(externalURL, nexusDefaultSpec.NexusRestApiUrlPath)
		if err != nil {
			return "", fmt.Errorf("failed to build Nexus API URL using external URL: %w", err)
		}

		return URL, nil
	}

	URL := fmt.Sprintf(
		"http://%s.%s:%d",
		instance.Name,
		instance.Namespace,
		nexusDefaultSpec.NexusPort,
	)

	URL, err := url.JoinPath(URL, instance.Spec.BasePath, nexusDefaultSpec.NexusRestApiUrlPath)
	if err != nil {
		return "", fmt.Errorf("failed to build Nexus API URL: %w", err)
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

// ExposeConfiguration creates new users in Nexus.
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
		return instance, fmt.Errorf("failed to update nexus instance: %w", err)
	}

	return instance, nil
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
		return instance, false, fmt.Errorf("failed to get default roles from Config Map: %w", err)
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
	instance *nexusApi.Nexus, nexusClient Client,
) error {
	newUserSecretName := fmt.Sprintf("%s-%s", instance.Name, newUser[crUsernameKey])
	if err := s.platformService.CreateSecret(instance, newUserSecretName, newUser); err != nil {
		return fmt.Errorf("failed to create secret %s: %w", newUserSecretName, err)
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
