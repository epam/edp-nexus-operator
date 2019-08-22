package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"
	"nexus-operator/pkg/apis/edp/v1alpha1"
	"nexus-operator/pkg/client/nexus"
	"nexus-operator/pkg/helper"
	nexusDefaultSpec "nexus-operator/pkg/service/nexus/spec"
	"nexus-operator/pkg/service/platform"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("nexus_service")

const (
	//NexusDefaultConfigurationDirectoryPath
	NexusDefaultConfigurationDirectoryPath = "/usr/local/configs/default-configuration"

	//NexusDefaultScriptsPath - default scripts for uploading to Nexus
	NexusDefaultScriptsPath = "/usr/local/configs/scripts"

	LocalConfigsRelativePath = "configs"
)

// NexusService interface for Nexus EDP component
type NexusService interface {
	Install(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, bool, error)
	ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	Integration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error)
	IsDeploymentConfigReady(instance v1alpha1.Nexus) (bool, error)
}

// NewNexusService function that returns NexusService implementation
func NewNexusService(platformService platform.PlatformService, k8sClient client.Client) NexusService {
	return NexusServiceImpl{platformService: platformService, k8sClient: k8sClient}
}

// NexusServiceImpl struct fo Nexus EDP Component
type NexusServiceImpl struct {
	platformService platform.PlatformService
	k8sClient       client.Client
	nexusClient     nexus.NexusClient
}

// IsDeploymentConfigReady check if DC for Nexus is ready
func (n NexusServiceImpl) IsDeploymentConfigReady(instance v1alpha1.Nexus) (bool, error) {
	nexusIsReady := false
	nexusDc, err := n.platformService.GetDeploymentConfig(instance)
	if err != nil {
		return nexusIsReady, helper.LogErrorAndReturn(err)
	}
	if nexusDc.Status.AvailableReplicas == 1 {
		nexusIsReady = true
	}
	return nexusIsReady, nil
}

func (n NexusServiceImpl) getNexusRestApiUrl(instance v1alpha1.Nexus) (string, error) {
	nexusApiUrl := fmt.Sprintf("http://%v.%v:%v/%v", instance.Name, instance.Namespace, nexusDefaultSpec.NexusPort, nexusDefaultSpec.NexusRestApiUrlPath)
	if _, err := k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		nexusRoute, nexusRouteScheme, err := n.platformService.GetRoute(instance.Namespace, instance.Name)
		if err != nil {
			return "", errors.Wrapf(err, "[ERROR] Failed to get Route for %v/%v", instance.Namespace, instance.Name)
		}
		nexusApiUrl = fmt.Sprintf("%v://%v/%v", nexusRouteScheme, nexusRoute.Spec.Host, nexusDefaultSpec.NexusRestApiUrlPath)
	}
	return nexusApiUrl, nil
}

func (n NexusServiceImpl) getNexusAdminPassword(instance v1alpha1.Nexus) (string, error) {
	secretName := fmt.Sprintf("%v-admin-password", instance.Name)
	nexusAdminCredentials, err := n.platformService.GetSecretData(instance.Namespace, secretName)
	if err != nil {
		return "", errors.Wrapf(err, "[ERROR] Failed to get Secret %v for %v/%v", secretName, instance.Namespace, instance.Name)
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
		keycloakSecretName := fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.IdentityServiceCredentialsSecretPostfix)

		keycloakSecretData, err := n.platformService.GetSecretData(instance.Namespace, keycloakSecretName)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to get Keycloak client data!")
		}

		err = n.platformService.AddKeycloakProxyToDeployConf(instance, keycloakSecretData)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to add Keycloak proxy!")
		}
	} else {
		log.V(1).Info("Keycloak integration not enabled.")
	}

	return &instance, nil
}

// ExposeConfiguration performs exposing Nexus configuration for other EDP components
func (n NexusServiceImpl) ExposeConfiguration(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	nexusApiUrl, err := n.getNexusRestApiUrl(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to get Nexus REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	err = n.nexusClient.InitNewRestClient(&instance, nexusApiUrl, nexusDefaultSpec.NexusDefaultAdminUser, nexusDefaultSpec.NexusDefaultAdminPassword)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to initialize Nexus client for %v/%v", instance.Namespace, instance.Name)
	}

	nexusDefaultUsersToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix))
	if err != nil {
		return &instance, errors.Wrapf(err, "Failed to get default tasks from Config Map for %v/%v", instance.Namespace, instance.Name)
	}

	var newUserSecretName string
	var parsedUsers []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultUsersToCreate[nexusDefaultSpec.NexusDefaultUsersConfigMapPrefix]), &parsedUsers)
	newUser := map[string][]byte{
		"username": []byte(""),
		"password": []byte(""),
	}

	for _, user := range parsedUsers {
		newUserSecretName = fmt.Sprintf("%v-%v", instance.Name, user["type"].(string))
		newUser["username"] = []byte(user["username"].(string))
		newUser["password"] = []byte(uniuri.New())
		ciUserAnnotationKey := helper.GenerateAnnotationKey(user["type"].(string))
		n.setAnnotation(&instance, ciUserAnnotationKey, newUserSecretName)

		err = n.platformService.CreateSecret(instance, newUserSecretName, newUser)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to create CI User credentials!")
		}

		data, err := n.platformService.GetSecretData(instance.Namespace, newUserSecretName)
		if err != nil {
			return &instance, errors.Wrap(err, "Failed to get CI user credentials!")
		}

		user["password"] = data["password"]

		_, err = n.nexusClient.RunScript("setup-user", user)
		if err != nil {
			return &instance, errors.Wrapf(err, "Failed to create user %v for %v/%v", user["username"], instance.Namespace, instance.Name)
		}
	}

	_ = n.k8sClient.Update(context.TODO(), &instance)

	identityServiceClientCredenrials := map[string][]byte{
		"client_id":     []byte(instance.Name),
		"client_secret": []byte(uniuri.New()),
	}

	identityServiceSecretName := fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.IdentityServiceCredentialsSecretPostfix)
	err = n.platformService.CreateSecret(instance, identityServiceSecretName, identityServiceClientCredenrials)
	if err != nil {
		return &instance, errors.Wrapf(err, fmt.Sprintf("Failed to create secret %v", identityServiceSecretName))
	}

	annotationKey := helper.GenerateAnnotationKey(nexusDefaultSpec.IdentityServiceCredentialsSecretPostfix)
	n.setAnnotation(&instance, annotationKey, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.IdentityServiceCredentialsSecretPostfix))
	_ = n.k8sClient.Update(context.TODO(), &instance)

	return &instance, nil
}

// Configure performs self-configuration of Nexus
func (n NexusServiceImpl) Configure(instance v1alpha1.Nexus) (*v1alpha1.Nexus, bool, error) {
	nexusApiUrl, err := n.getNexusRestApiUrl(instance)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to get Nexus REST API URL %v/%v", instance.Namespace, instance.Name)
	}

	nexusGeneratedPassword, err := n.getNexusAdminPassword(instance)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to get Nexus admin password from secret for %v/%v", instance.Namespace, instance.Name)
	}

	err = n.nexusClient.InitNewRestClient(&instance, nexusApiUrl, nexusDefaultSpec.NexusDefaultAdminUser, nexusDefaultSpec.NexusDefaultAdminPassword)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to initialize Nexus client for %v/%v", instance.Namespace, instance.Name)
	}

	if _, responseStatus, err := n.nexusClient.IsNexusRestApiReady(); err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Checking if Nexus REST API for %v/%v object is ready has been failed", instance.Namespace, instance.Name)
	} else if responseStatus == 401 {
		err = n.nexusClient.InitNewRestClient(&instance, nexusApiUrl, nexusDefaultSpec.NexusDefaultAdminUser, nexusGeneratedPassword)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed to initialize Nexus client for %v/%v", instance.Namespace, instance.Name)
		}
	}

	if nexusApiIsReady, _, err := n.nexusClient.IsNexusRestApiReady(); err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Checking if Nexus REST API for %v/%v object is ready has been failed", instance.Namespace, instance.Name)
	} else if !nexusApiIsReady {
		log.Info(fmt.Sprintf("Nexus REST API for %v/%v object is not ready for configuration yet", instance.Namespace, instance.Name))
		return &instance, false, nil
	}

	nexusDefaultScriptsToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultScriptsConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrap(err, "[ERROR] Failed to get default tasks from Config Map")
	}

	err = n.nexusClient.DeclareDefaultScripts(nexusDefaultScriptsToCreate)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to upload default scripts for %v/%v", instance.Namespace, instance.Name)
	}

	defaultScriptsAreDeclared, err := n.nexusClient.AreDefaultScriptsDeclared(nexusDefaultScriptsToCreate)
	if !defaultScriptsAreDeclared || err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Default scripts for %v/%v are not uploaded yet", instance.Namespace, instance.Name)
	}

	updatePasswordParameters := map[string]interface{}{"new_password": nexusGeneratedPassword}
	_, err = n.nexusClient.RunScript("update-admin-password", updatePasswordParameters)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed update admin password for %v/%v", instance.Namespace, instance.Name)
	}
	err = n.nexusClient.InitNewRestClient(&instance, nexusApiUrl, nexusDefaultSpec.NexusDefaultAdminUser, nexusGeneratedPassword)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to initialize Nexus client for %v/%v", instance.Namespace, instance.Name)
	}

	nexusDefaultTasksToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to get default tasks from Config Map for %v/%v", instance.Namespace, instance.Name)
	}

	var parsedTasks []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultTasksToCreate[nexusDefaultSpec.NexusDefaultTasksConfigMapPrefix]), &parsedTasks)
	for _, taskParameters := range parsedTasks {
		_, err = n.nexusClient.RunScript("create-task", taskParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed to create task %v for %v/%v", taskParameters["name"], instance.Namespace, instance.Name)
		}
	}

	var emptyParameter map[string]interface{}
	_, err = n.nexusClient.RunScript("disable-outreach-capability", emptyParameter)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to run disable-outreach-capability scripts for %v/%v", instance.Namespace, instance.Name)
	}

	enabledRealms := []map[string]interface{}{
		{"name": "NuGetApiKey"},
	}
	for _, realmName := range enabledRealms {
		_, err = n.nexusClient.RunScript("enable-realm", realmName)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed enable %v for %v/%v", enabledRealms, instance.Namespace, instance.Name)
		}
	}

	nexusDefaultRolesToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to get default roles from Config Map for %v/%v", instance.Namespace, instance.Name)
	}

	var parsedRoles []map[string]interface{}
	err = json.Unmarshal([]byte(nexusDefaultRolesToCreate[nexusDefaultSpec.NexusDefaultRolesConfigMapPrefix]), &parsedRoles)
	for _, roleParameters := range parsedRoles {
		_, err := n.nexusClient.RunScript("setup-role", roleParameters)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed to create role %v for %v/%v", roleParameters["name"], instance.Namespace, instance.Name)
		}
	}

	// Creating blob storage configuration from config map
	blobsConfig, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-blobs", instance.Name))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to get data from ConfigMap %v-blobs", instance.Name)
	}

	var parsedBlobsConfig []map[string]interface{}
	err = json.Unmarshal([]byte(blobsConfig["blobs"]), &parsedBlobsConfig)
	if err != nil {
		return &instance, false, errors.Wrap(err, "[ERROR] Failed to unmarshal blob ConfigMap")
	}

	for _, blob := range parsedBlobsConfig {
		_, err := n.nexusClient.RunScript("create-blobstore", blob)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed to create blob store %v!", blob["name"])
		}
	}

	// Creating repositoriesToCreate from config map
	reposToCreate, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR]  Failed to get data from ConfigMap %v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix)
	}

	var parsedReposToCreate []map[string]interface{}
	err = json.Unmarshal([]byte(reposToCreate[nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix]), &parsedReposToCreate)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to unmarshal %v-%v ConfigMap!", instance.Name, nexusDefaultSpec.NexusDefaultReposToCreateConfigMapPrefix)
	}

	for _, repositoryToCreate := range parsedReposToCreate {
		repositoryName := repositoryToCreate["name"].(string)
		repositoryType := repositoryToCreate["repositoryType"].(string)
		_, err := n.nexusClient.RunScript(fmt.Sprintf("create-repo-%v", repositoryType), repositoryToCreate)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed to create repository %v!", repositoryName)
		}
	}

	reposToDelete, err := n.platformService.GetConfigMapData(instance.Namespace, fmt.Sprintf("%v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix))
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR]  Failed to get data from ConfigMap %v-%v", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix)
	}

	var parsedReposToDelete []map[string]interface{}
	err = json.Unmarshal([]byte(reposToDelete[nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix]), &parsedReposToDelete)
	if err != nil {
		return &instance, false, errors.Wrapf(err, "[ERROR] Failed to unmarshal %v-%v ConfigMap!", instance.Name, nexusDefaultSpec.NexusDefaultReposToDeleteConfigMapPrefix)
	}

	for _, repositoryToDelete := range parsedReposToDelete {
		_, err := n.nexusClient.RunScript("delete-repo", repositoryToDelete)
		if err != nil {
			return &instance, false, errors.Wrapf(err, "[ERROR] Failed to delete repository %v", repositoryToDelete)
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
			return &instance, false, errors.Wrapf(err, "Failed to create user %v", user.Username, instance.Namespace, instance.Name)
		}
	}

	return &instance, true, nil
}

// Install performs installation of Nexus
func (n NexusServiceImpl) Install(instance v1alpha1.Nexus) (*v1alpha1.Nexus, error) {
	adminSecret := map[string][]byte{
		"user":     []byte(nexusDefaultSpec.NexusDefaultAdminUser),
		"password": []byte(uniuri.New()),
	}

	err := n.platformService.CreateSecret(instance, instance.Name+"-admin-password", adminSecret)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to Secret for %v/%v", instance.Namespace, instance.Name)
	}

	err = n.platformService.CreateVolume(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create Volume for %v/%v", instance.Namespace, instance.Name)
	}

	_, err = n.platformService.CreateServiceAccount(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create Service Account for %v/%v", instance.Namespace, instance.Name)
	}

	err = n.platformService.CreateService(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create Service for %v/%v", instance.Namespace, instance.Name)
	}

	executableFilePath := helper.GetExecutableFilePath()
	NexusConfigurationDirectoryPath := NexusDefaultConfigurationDirectoryPath
	if _, err = k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		NexusConfigurationDirectoryPath = fmt.Sprintf("%v/../%v/default-configuration", executableFilePath, LocalConfigsRelativePath)
	}
	err = n.platformService.CreateConfigMapsFromDirectory(instance, NexusConfigurationDirectoryPath, true)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create default Config Maps for configuration %v/%v", instance.Namespace, instance.Name)
	}

	NexusScriptsPath := NexusDefaultScriptsPath
	if _, err = k8sutil.GetOperatorNamespace(); err != nil && err == k8sutil.ErrNoNamespace {
		NexusScriptsPath = fmt.Sprintf("%v/../%v/scripts", executableFilePath, LocalConfigsRelativePath)
	}
	err = n.platformService.CreateConfigMapsFromDirectory(instance, NexusScriptsPath, false)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create default Config Maps for scripts for %v/%v", instance.Namespace, instance.Name)
	}

	err = n.platformService.CreateDeployConf(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create Deployment Config for %v/%v", instance.Namespace, instance.Name)
	}

	err = n.platformService.CreateExternalEndpoint(instance)
	if err != nil {
		return &instance, errors.Wrapf(err, "[ERROR] Failed to create External Route for %v/%v", instance.Namespace, instance.Name)
	}

	return &instance, nil
}
