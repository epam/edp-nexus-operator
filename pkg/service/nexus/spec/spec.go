package spec

const (
	//NexusDefaultPropertiesConfigMapPrefix
	NexusDefaultPropertiesConfigMapPrefix = "nexus-default.properties"

	//NexusDefaultTasksConfigMapPrefix
	NexusDefaultTasksConfigMapPrefix = "default-tasks"

	//NexusDefaultTasksConfigMapPrefix
	NexusDefaultScriptsConfigMapPrefix = "scripts"

	//NexusDefaultRolesConfigMapPrefix
	NexusDefaultRolesConfigMapPrefix = "default-roles"

	//NexusDefaultReposToCreateConfigMapPrefix
	NexusDefaultReposToCreateConfigMapPrefix = "repos-to-create"

	//NexusDefaultReposToDeleteConfigMapPrefix
	NexusDefaultReposToDeleteConfigMapPrefix = "repos-to-delete"

	//NexusDefaultUsersConfigMapPrefix key for ConfigMap entry with default users list
	NexusDefaultUsersConfigMapPrefix = "default-users"

	//EdpAnnotationsPrefix general prefix for all annotation made by EDP team
	EdpAnnotationsPrefix = "edp.epam.com"

	//IdentityServiceCredentialsSecretPostfix
	IdentityServiceCredentialsSecretPostfix = "is-credentials"

	//NexusPort - default Nexus port
	NexusPort = 8081

	//NexusMemoryRequest - default request value for memory request for deployment config
	NexusMemoryRequest = "500Mi"

	//NexusDefaultAdminUser - default admin username in Nexus
	NexusDefaultAdminUser string = "admin"

	//NexusDefaultAdminPassword - default admin password in Nexus
	NexusDefaultAdminPassword string = "admin123"

	//NexusRestApiUrlPath - Nexus relative REST API path
	NexusRestApiUrlPath = "service/rest/v1"

	//EdpCiUserSuffix entity prefix for integration functionality
	EdpCiUserSuffix string = "ci-credentials"

	NexusKeycloakProxyPort int32 = 3000

	NexusKeycloakProxyImage string = "keycloak/keycloak-gatekeeper:6.0.1"
)
