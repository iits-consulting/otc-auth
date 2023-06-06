package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"otc-auth/common"
)

var otcConfigPath = path.Join(GetHomeFolder(), ".otc-auth-config")

func LoadCloudConfig(domainName string) {
	otcConfig := getOtcConfig()
	clouds := otcConfig.Clouds
	if !clouds.ContainsCloud(domainName) {
		clouds = registerNewCloud(domainName)
	}
	clouds.SetActiveByName(domainName)
	otcConfig.Clouds = clouds
	writeOtcConfigContentToFile(otcConfig)

	_, err := fmt.Fprintf(os.Stdout, "Cloud %s loaded successfully and set to active.\n", domainName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
}

func registerNewCloud(domainName string) Clouds {
	otcConfig := getOtcConfig()
	clouds := otcConfig.Clouds

	newCloud := Cloud{
		Domain: NameAndIdResource{
			Name: domainName,
		},
	}
	if otcConfig.Clouds.ContainsCloud(newCloud.Domain.Name) {
		common.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("warning: cloud with name %s already exists.\n\nUse the cloud-config load command.", newCloud.Domain.Name))

		return nil
	}

	return append(clouds, newCloud)
}

func IsAuthenticationValid() bool {
	cloud := GetActiveCloudConfig()

	if !cloud.UnscopedToken.IsTokenValid() {
		return false
	}

	unscopedToken := cloud.UnscopedToken

	tokenExpirationDate := common.ParseTimeOrThrow(unscopedToken.ExpiresAt)
	if tokenExpirationDate.After(time.Now()) {
		// token still valid
		println(fmt.Sprintf("info: unscoped token valid until %s", tokenExpirationDate.Format(common.PrintTimeFormat)))

		return true
	}

	// token expired
	return false
}

func RemoveCloudConfig(domainName string) {
	otcConfig := getOtcConfig()
	if !otcConfig.Clouds.ContainsCloud(domainName) {
		common.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: cloud with name %s does not exist in the config file.", domainName))
	}

	removeCloudConfig(domainName)

	_, err := fmt.Fprintf(os.Stdout, "Cloud %s deleted successfully.", domainName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
}

func UpdateClusters(clusters Clusters) {
	otcConfig := getOtcConfig()
	cloudIndex := otcConfig.Clouds.GetActiveCloudIndex()
	otcConfig.Clouds[cloudIndex].Clusters = clusters
	writeOtcConfigContentToFile(otcConfig)
}

func UpdateProjects(projects Projects) {
	otcConfig := getOtcConfig()
	cloudIndex := otcConfig.Clouds.GetActiveCloudIndex()
	otcConfig.Clouds[cloudIndex].Projects = projects
	writeOtcConfigContentToFile(otcConfig)
}

func UpdateCloudConfig(updatedCloud Cloud) {
	otcConfig := getOtcConfig()
	index := otcConfig.Clouds.GetActiveCloudIndex()
	otcConfig.Clouds[index] = updatedCloud

	writeOtcConfigContentToFile(otcConfig)
}

func GetActiveCloudConfig() Cloud {
	otcConfig := getOtcConfig()
	clouds := otcConfig.Clouds
	cloud, _, err := clouds.FindActiveCloudConfigOrNil()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: %s.\n\nPlease use the cloud-config register or the cloud-config load command to set an active cloud configuration.")
	}
	return *cloud
}

func OtcConfigFileExists() bool {
	fileInfo, err := os.Stat(otcConfigPath)
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return !fileInfo.IsDir()
}

func getOtcConfig() OtcConfigContent {
	if !OtcConfigFileExists() {
		createConfigFileWithCloudConfig(OtcConfigContent{})
		println("info: cloud config created.")
	}

	var otcConfig OtcConfigContent
	content := readFileContent()

	err := json.Unmarshal([]byte(content), &otcConfig)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error deserializing json.\ntrace: %s")
	}
	return otcConfig
}

func GetHomeFolder() (homeFolder string) {
	homeFolder, err := os.UserHomeDir()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error retrieving home directory.\ntrace: %s")
	}

	return homeFolder
}

func createConfigFileWithCloudConfig(content OtcConfigContent) {
	writeOtcConfigContentToFile(content)
}

func writeOtcConfigContentToFile(content OtcConfigContent) {
	contentAsBytes, err := json.Marshal(content)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error encoding json.\ntrace: %s")
	}

	WriteConfigFile(common.ByteSliceToIndentedJsonFormat(contentAsBytes), otcConfigPath)
}

func readFileContent() string {
	file, err := os.Open(otcConfigPath)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error opening config file.\ntrace: %s")
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			common.OutputErrorToConsoleAndExit(err, "fatal: error saving config file.\ntrace: %s")
		}
	}(file)

	fileScanner := bufio.NewScanner(file)
	var content string
	for fileScanner.Scan() {
		content += fileScanner.Text()
	}
	if err := fileScanner.Err(); err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error reading config file.\ntrace: %s")
	}

	return content
}

func WriteConfigFile(content string, configPath string) {
	file, err := os.Create(configPath)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error reading config file.\ntrace: %s")
	}

	_, err = file.WriteString(content)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error writing to config file.\ntrace: %s")
	}

	err = file.Close()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error saving config file.\ntrace: %s")
	}
}

func removeCloudConfig(name string) {
	otcConfig := getOtcConfig()

	otcConfig.Clouds.RemoveCloudByNameIfExists(name)
	writeOtcConfigContentToFile(otcConfig)
}
