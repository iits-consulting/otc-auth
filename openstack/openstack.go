package openstack

import (
	"github.com/gophercloud/utils/openstack/clientconfig"
	"gopkg.in/yaml.v2"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
	"otc-auth/iam"
)

func WriteOpenStackCloudsYaml(openStackConfigFileLocation string) {
	cloudConfig := config.GetActiveCloudConfig()
	domainName := cloudConfig.Domain.Name
	println("info: will get a scoped token for every project from domain=" + domainName)
	createScopedTokenForEveryProject(cloudConfig.Projects.GetProjectNames())

	updatedCloudConfig := config.GetActiveCloudConfig()
	clouds := make(map[string]clientconfig.Cloud)
	for _, project := range updatedCloudConfig.Projects {
		cloudName := domainName + "_" + project.Name
		clouds[cloudName] = createOpenstackCloudConfig(project, domainName)
	}
	createOpenstackCloudsYAML(clientconfig.Clouds{Clouds: clouds}, openStackConfigFileLocation)
}

func createScopedTokenForEveryProject(projectNames []string) {
	for _, projectName := range projectNames {
		iam.GetScopedToken(projectName)
	}
}

func createOpenstackCloudConfig(project config.Project, domainName string) clientconfig.Cloud {
	projectName := project.Name
	cloudName := domainName + "_" + projectName

	authInfo := clientconfig.AuthInfo{
		AuthURL:           endpoints.BaseUrlIam + "/v3",
		Token:             project.ScopedToken.Secret,
		ProjectDomainName: projectName,
	}

	openstackCloudConfig := clientconfig.Cloud{
		Cloud:              cloudName,
		Profile:            cloudName,
		AuthInfo:           &authInfo,
		AuthType:           "token",
		Interface:          "public",
		IdentityAPIVersion: "3",
	}
	return openstackCloudConfig
}

func createOpenstackCloudsYAML(clouds clientconfig.Clouds, openStackConfigFileLocation string) {
	contentAsBytes, err := yaml.Marshal(clouds)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error encoding json.\ntrace: %s")
	}

	if openStackConfigFileLocation == "" {
		config.WriteConfigFile(string(contentAsBytes), config.GetHomeFolder()+"/.config/openstack/clouds.yaml")
	} else {
		config.WriteConfigFile(string(contentAsBytes), openStackConfigFileLocation)
	}

	println("info: openstack clouds.yaml was updated")
}
