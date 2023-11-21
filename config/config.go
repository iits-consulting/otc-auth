package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"otc-auth/common"
)

func LoadCloudConfig(domainName string) {
	otcConfig := getOtcConfig()
	clouds := otcConfig.Clouds
	if !clouds.ContainsCloud(domainName) {
		clouds = registerNewCloud(domainName)
	}
	clouds.SetActiveByName(domainName)
	otcConfig.Clouds = clouds
	writeOtcConfigContentToFile(otcConfig)

	log.Printf("info: cloud %s loaded successfully and set to active.\n", domainName)
}

func registerNewCloud(domainName string) Clouds {
	otcConfig := getOtcConfig()
	clouds := otcConfig.Clouds

	newCloud := Cloud{
		Domain: NameAndIDResource{
			Name: domainName,
		},
	}
	if otcConfig.Clouds.ContainsCloud(newCloud.Domain.Name) {
		log.Fatalf(
			"warning: cloud with name %s already exists.\n\nUse the cloud-config load command",
			newCloud.Domain.Name)

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
		log.Printf("info: unscoped token valid until %s", tokenExpirationDate.Format(common.PrintTimeFormat))

		return true
	}

	// token expired
	return false
}

func RemoveCloudConfig(domainName string) {
	otcConfig := getOtcConfig()
	if !otcConfig.Clouds.ContainsCloud(domainName) {
		log.Fatalf(
			"fatal: cloud with name %s does not exist in the config file", domainName)
	}

	removeCloudConfig(domainName)

	_, err := fmt.Fprintf(os.Stdout, "Cloud %s deleted successfully", domainName)
	if err != nil {
		log.Fatal(err)
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
		log.Fatalf(
			"fatal: %s.\n\nPlease use the cloud-config register or the cloud-config load command "+
				"to set an active cloud configuration", err)
	}
	return *cloud
}

func OtcConfigFileExists() bool {
	fileInfo, err := os.Stat(path.Join(GetHomeFolder(), ".otc-auth-config"))
	if err != nil && os.IsNotExist(err) {
		return false
	}

	return !fileInfo.IsDir()
}

func getOtcConfig() OtcConfigContent {
	if !OtcConfigFileExists() {
		createConfigFileWithCloudConfig(OtcConfigContent{})
		log.Println("info: cloud config created")
	}

	var otcConfig OtcConfigContent
	content := readFileContent()

	err := json.Unmarshal([]byte(content), &otcConfig)
	if err != nil {
		log.Fatalf("fatal: error deserializing json.\ntrace: %s", err)
	}
	return otcConfig
}

func GetHomeFolder() (homeFolder string) {
	homeFolder, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("fatal: error retrieving home directory.\ntrace: %s", err)
	}

	return homeFolder
}

func createConfigFileWithCloudConfig(content OtcConfigContent) {
	writeOtcConfigContentToFile(content)
}

func writeOtcConfigContentToFile(content OtcConfigContent) {
	contentAsBytes, err := json.Marshal(content)
	if err != nil {
		log.Fatalf("fatal: error encoding json.\ntrace: %s", err)
	}

	WriteConfigFile(common.ByteSliceToIndentedJSONFormat(contentAsBytes), path.Join(GetHomeFolder(), ".otc-auth-config"))
}

func readFileContent() string {
	file, err := os.Open(path.Join(GetHomeFolder(), ".otc-auth-config"))
	if err != nil {
		log.Fatalf("fatal: error opening config file.\ntrace: %s", err)
	}
	defer func(file *os.File) {
		errClose := file.Close()
		if errClose != nil {
			log.Fatalf("fatal: error saving config file.\ntrace: %s", errClose)
		}
	}(file)

	fileScanner := bufio.NewScanner(file)
	var content string
	for fileScanner.Scan() {
		content += fileScanner.Text()
	}
	if errScanner := fileScanner.Err(); errScanner != nil {
		log.Printf("fatal: error reading config file.\ntrace: %s", errScanner)
	}

	return content
}

func WriteConfigFile(content string, configPath string) {
	file, err := os.Create(configPath)
	if err != nil {
		log.Fatalf("fatal: error reading config file.\ntrace: %s", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		log.Fatalf("fatal: error writing to config file.\ntrace: %s", err)
	}

	err = file.Close()
	if err != nil {
		log.Fatalf("fatal: error saving config file.\ntrace: %s", err)
	}
}

func removeCloudConfig(name string) {
	otcConfig := getOtcConfig()

	otcConfig.Clouds.RemoveCloudByNameIfExists(name)
	writeOtcConfigContentToFile(otcConfig)
}
