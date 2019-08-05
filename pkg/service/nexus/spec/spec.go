package spec

const (
	//NexusProperties - default configuration of Nexus
	NexusProperties = "/usr/local/configs/nexus-default.properties"

	//NexusScriptsPath - default scripts for uploading to Nexus
	NexusScriptsPath = "/usr/local/configs/scripts"

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
)
