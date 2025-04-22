package openstack

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	"github.com/golang/glog"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"gopkg.in/yaml.v3"
)

func WriteOpenStackCloudsYaml(openStackConfigFileLocation string) {
	cloudConfig := config.GetActiveCloudConfig()
	domainName := cloudConfig.Domain.Name
	clouds := make(map[string]clientconfig.Cloud)
	for _, project := range cloudConfig.Projects {
		cloudName := domainName + "_" + project.Name
		clouds[cloudName] = createOpenstackCloudConfig(project, domainName, cloudConfig.Region)
	}

	createOpenstackCloudsYAML(clientconfig.Clouds{Clouds: clouds}, openStackConfigFileLocation)
}

func createOpenstackCloudConfig(project config.Project, domainName string, regionCode string) clientconfig.Cloud {
	projectName := project.Name
	cloudName := domainName + "_" + projectName

	authInfo := clientconfig.AuthInfo{
		AuthURL:           endpoints.BaseURLIam(regionCode),
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
		common.ThrowError(fmt.Errorf("fatal: error encoding json.\ntrace: %w", err))
	}

	if openStackConfigFileLocation == "" {
		openStackConfigFileLocation = path.Join(config.GetHomeFolder(), ".config", "openstack", "clouds.yaml")
	}
	mkDirError := os.MkdirAll(filepath.Dir(openStackConfigFileLocation), os.ModePerm)
	if mkDirError != nil {
		common.ThrowError(err)
	}
	config.WriteConfigFile(string(contentAsBytes), openStackConfigFileLocation)

	glog.V(1).Info("info: openstack clouds.yaml was updated")
}
