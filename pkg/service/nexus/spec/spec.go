package spec

const (
	//NexusDefaultPropertiesConfigMapPrefix
	NexusDefaultPropertiesConfigMapPrefix = "default-properties"

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
	NexusDefaultUsersConfigMapPrefix string = "default-users"

	//NexusPort - default Nexus port
	NexusPort = 8081

	//NexusDockerImage - default Nexus Docker image
	NexusDockerImage = "sonatype/nexus3"

	//NexusMemoryRequest - default request value for memory request for deployment config
	NexusMemoryRequest = "500Mi"

	//NexusDefaultAdminUser - default admin username in Nexus
	NexusDefaultAdminUser = "admin"

	//NexusDefaultAdminPassword - default admin password in Nexus
	NexusDefaultAdminPassword = "admin123"

	//NexusRestApiUrlPath - Nexus relative REST API path
	NexusRestApiUrlPath = "service/rest/v1"

	//EdpAnnotationsPrefix general prefix for all annotation made by EDP team
	EdpAnnotationsPrefix string = "edp.epam.com"

	//EdpCiUserSuffix entity prefix for integration functionality
	EdpCiUserSuffix string = "ci-credentials"
)
