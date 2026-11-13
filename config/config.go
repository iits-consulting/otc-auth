package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"otc-auth/common"

	"github.com/golang/glog"
)

const configFileName = ".otc-auth-config"

var configFilePath string //nolint:gochecknoglobals // TODO - Consider DI?

func SetCustomConfigFilePath(path string) {
	configFilePath = path
}

func effectiveConfigPath() (string, error) {
	configPath := configFilePath

	if configFilePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error retrieving home directory: %w", err)
		}

		configPath = homeDir
	}

	return path.Join(configPath, configFileName), nil
}

func LoadCloudConfig(domainName string) error {
	otcConfig, err := getOtcConfig()
	if err != nil {
		return err
	}
	clouds := otcConfig.Clouds
	if !clouds.ContainsCloud(domainName) {
		clouds = registerNewCloud(domainName)
	}
	clouds.SetActiveByName(domainName)
	otcConfig.Clouds = clouds
	err = writeOtcConfigContentToFile(*otcConfig)
	if err != nil {
		return err
	}

	glog.V(common.InfoLogLevel).Infof("info: cloud %s loaded successfully and set to active.\n", domainName)
	return nil
}

func registerNewCloud(domainName string) Clouds {
	otcConfig, err := getOtcConfig()
	if err != nil {
		common.ThrowError(err)
	}
	clouds := otcConfig.Clouds

	newCloud := Cloud{
		Domain: NameAndIDResource{
			Name: domainName,
		},
	}
	if otcConfig.Clouds.ContainsCloud(newCloud.Domain.Name) {
		common.ThrowError(fmt.Errorf(
			"warning: cloud with name %s already exists.\n\nUse the cloud-config load command",
			newCloud.Domain.Name))

		return nil
	}

	return append(clouds, newCloud)
}

func IsAuthenticationValid() bool {
	cloud, err := GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}

	if !cloud.UnscopedToken.IsValid() {
		return false
	}

	unscopedToken := cloud.UnscopedToken

	tokenExpirationDate, err := common.ParseTime(unscopedToken.ExpiresAt)
	if err != nil {
		common.ThrowError(err)
	}
	if tokenExpirationDate.After(time.Now()) {
		// token still valid
		glog.V(common.InfoLogLevel).Infof("info: unscoped token valid until %s",
			tokenExpirationDate.Format(common.PrintTimeFormat))

		return true
	}

	// token expired
	return false
}

func RemoveCloudConfig(domainName string) {
	otcConfig, err := getOtcConfig()
	if err != nil {
		common.ThrowError(err)
	}
	if !otcConfig.Clouds.ContainsCloud(domainName) {
		glog.Warning("warning: cloud with name %s doesn't exist.\n", domainName)
		return
	}

	removeCloudConfig(domainName)

	_, err = fmt.Fprintf(os.Stdout, "Cloud %s deleted successfully", domainName)
	if err != nil {
		common.ThrowError(err)
	}
}

func UpdateClusters(clusters Clusters) {
	otcConfig, err := getOtcConfig()
	if err != nil {
		common.ThrowError(err)
	}
	cloudIndex, err := otcConfig.Clouds.GetActiveCloudIndex()
	if err != nil {
		common.ThrowError(err)
	}
	otcConfig.Clouds[*cloudIndex].Clusters = clusters
	err = writeOtcConfigContentToFile(*otcConfig)
	if err != nil {
		common.ThrowError(err)
	}
}

func UpdateProjects(projects Projects) {
	otcConfig, err := getOtcConfig()
	if err != nil {
		common.ThrowError(err)
	}
	cloudIndex, err := otcConfig.Clouds.GetActiveCloudIndex()
	if err != nil {
		common.ThrowError(err)
	}
	otcConfig.Clouds[*cloudIndex].Projects = projects
	err = writeOtcConfigContentToFile(*otcConfig)
	if err != nil {
		common.ThrowError(err)
	}
}

func UpdateCloudConfig(updatedCloud Cloud) error {
	otcConfig, err := getOtcConfig()
	if err != nil {
		return fmt.Errorf("couldn't get otc config: %w", err)
	}
	index, err := otcConfig.Clouds.GetActiveCloudIndex()
	if err != nil {
		return fmt.Errorf("couldn't get active cloud idx: %w", err)
	}
	otcConfig.Clouds[*index] = updatedCloud

	err = writeOtcConfigContentToFile(*otcConfig)
	if err != nil {
		return fmt.Errorf("couldn't write config to file: %w", err)
	}

	return nil
}

func GetActiveCloudConfig() (*Cloud, error) {
	otcConfig, err := getOtcConfig()
	if err != nil {
		return nil, err
	}
	clouds := otcConfig.Clouds
	cloud, _, err := clouds.FindActiveCloudConfigOrNil()
	if err != nil {
		return nil,
			fmt.Errorf(
				"fatal: %w.\n\nPlease use the cloud-config register or the cloud-config load command "+
					"to set an active cloud configuration", err)
	}
	return cloud, nil
}

func OtcConfigFileExists() (bool, error) {
	path, err := effectiveConfigPath()
	if err != nil {
		return false, err
	}
	fileInfo, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}

	return !fileInfo.IsDir(), nil
}

func getOtcConfig() (*OtcConfigContent, error) {
	exists, err := OtcConfigFileExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		err = createConfigFileWithCloudConfig(OtcConfigContent{})
		if err != nil {
			return nil, err
		}
		glog.V(common.InfoLogLevel).Info("info: cloud config created")
	}

	var otcConfig OtcConfigContent
	content, err := readFileContent()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(*content), &otcConfig)
	if err != nil {
		return nil, fmt.Errorf("fatal: error deserializing json.\ntrace: %w", err)
	}
	return &otcConfig, nil
}

func createConfigFileWithCloudConfig(content OtcConfigContent) error {
	err := writeOtcConfigContentToFile(content)
	if err != nil {
		return err
	}
	return nil
}

func writeOtcConfigContentToFile(content OtcConfigContent) error {
	contentAsBytes, err := json.Marshal(content)
	if err != nil {
		err = errors.Join(err, errors.New("fatal: error encoding json"))
		return err
	}

	path, err := effectiveConfigPath()
	indentedContent, indErr := common.ByteSliceToIndentedJSONFormat(contentAsBytes)
	writeErr := WriteConfigFile(indentedContent, path)
	return errors.Join(err, indErr, writeErr)
}

func readFileContent() (*string, error) {
	path, err := effectiveConfigPath()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fatal: error opening config file.\ntrace: %w", err)
	}
	var errClose error
	defer func(file *os.File) {
		errClose = file.Close()
		if errClose != nil {
			errClose = fmt.Errorf("fatal: error saving config file.\ntrace: %w", errClose)
		}
	}(file)

	fileScanner := bufio.NewScanner(file)
	var content string
	for fileScanner.Scan() {
		content += fileScanner.Text()
	}
	if errScanner := fileScanner.Err(); errScanner != nil {
		common.ThrowError(fmt.Errorf("fatal: error reading config file.\ntrace: %w", errScanner))
	}

	return &content, errClose
}

func WriteConfigFile(content string, configPath string) error {
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("fatal: error reading config file.\ntrace: %w", err)
	}

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("fatal: error writing to config file.\ntrace: %w", err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("fatal: error saving config file.\ntrace: %w", err)
	}
	return nil
}

func removeCloudConfig(name string) {
	otcConfig, err := getOtcConfig()
	if err != nil {
		common.ThrowError(err)
	}

	otcConfig.Clouds.RemoveCloudByNameIfExists(name)
	err = writeOtcConfigContentToFile(*otcConfig)
	if err != nil {
		common.ThrowError(err)
	}
}
