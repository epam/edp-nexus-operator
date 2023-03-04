package spec

const (
	// NexusDefaultTasksConfigMapPrefix is a default prefix of the tasks config map.
	NexusDefaultTasksConfigMapPrefix = "default-tasks"

	// NexusDefaultScriptsConfigMapPrefix is a default prefix of the scripts config map.
	NexusDefaultScriptsConfigMapPrefix = "scripts"

	// NexusDefaultRolesConfigMapPrefix is a default prefix of the roles config map.
	NexusDefaultRolesConfigMapPrefix = "default-roles"

	// NexusDefaultReposToCreateConfigMapPrefix is a default prefix of the config map of repos staged for creation.
	NexusDefaultReposToCreateConfigMapPrefix = "repos-to-create"

	// NexusDefaultReposToDeleteConfigMapPrefix is a default prefix of the config map of repos staged for deletion.
	NexusDefaultReposToDeleteConfigMapPrefix = "repos-to-delete"

	// NexusDefaultUsersConfigMapPrefix is a default key for ConfigMap entry with default users list.
	NexusDefaultUsersConfigMapPrefix = "default-users"

	// EdpAnnotationsPrefix is a general prefix for all annotation made by EDP team.
	EdpAnnotationsPrefix = "edp.epam.com"

	// NexusPort is a default Nexus port.
	NexusPort = 8081

	// NexusDefaultAdminUser is a default admin username in Nexus.
	NexusDefaultAdminUser string = "admin"

	// NexusDefaultAdminPassword is a default admin password in Nexus.
	NexusDefaultAdminPassword string = "admin123"

	// NexusRestApiUrlPath is the Nexus relative REST API path.
	NexusRestApiUrlPath = "service/rest/v1"
)
