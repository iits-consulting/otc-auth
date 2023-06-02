package cce

import (
	"encoding/json"
	"fmt"
	. "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"otc-auth/common"
	"otc-auth/config"
	"path"
	"path/filepath"
	"strings"
)

func getKubeConfig(kubeConfigParams KubeConfigParams) string {
	println("Getting kube config...")

	clusterId, err := getClusterId(kubeConfigParams.ClusterName, kubeConfigParams.ProjectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error receiving cluster id: %s")
	}

	response, err := getClusterCertFromServiceProvider(kubeConfigParams.ProjectName, clusterId, kubeConfigParams.DaysValid)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	responseMarshalled, err := json.Marshal(response)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	return string(responseMarshalled)
}

func mergeKubeConfig(configParams KubeConfigParams, kubeConfigData string) {
	// TODO - merge misses some lines
	kubeConfigContextData := addContextInformationToKubeConfig(configParams.ProjectName, configParams.ClusterName, kubeConfigData)
	currentConfig, err := NewDefaultClientConfigLoadingRules().GetStartingConfig()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	clientConfig, err := NewClientConfigFromBytes([]byte(kubeConfigContextData))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	kubeConfig, err := clientConfig.RawConfig()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	filenameNewFile := "kubeConfig_new"
	filenameCurrentFile := "kubeConfig_current"

	err = WriteToFile(kubeConfig, filenameNewFile)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	err = WriteToFile(*currentConfig, filenameCurrentFile)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	loadingRules := ClientConfigLoadingRules{
		Precedence: []string{filenameNewFile, filenameCurrentFile},
	}

	mergedConfig, err := loadingRules.Load()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	err = WriteToFile(*mergedConfig, determineTargetLocation(configParams.TargetLocation))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	_ = os.RemoveAll(filenameNewFile)
	_ = os.RemoveAll(filenameCurrentFile)
}

func determineTargetLocation(targetLocation string) string {
	defaultKubeConfigLocation := path.Join(homedir.HomeDir(), ".kube", "config")
	if targetLocation != "" {
		err := os.MkdirAll(filepath.Dir(targetLocation), os.ModePerm)
		if err != nil {
			common.OutputErrorMessageToConsoleAndExit(err.Error())
		}
		return targetLocation
	}
	return defaultKubeConfigLocation
}

func addContextInformationToKubeConfig(projectName string, clusterName string, kubeConfigData string) string {
	cloud := config.GetActiveCloudConfig()

	kubeConfigData = strings.ReplaceAll(kubeConfigData, "internalCluster", fmt.Sprintf("%s/%s-intranet", projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "externalCluster", fmt.Sprintf("%s/%s", projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "internal", fmt.Sprintf("%s/%s-intranet", projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "external", fmt.Sprintf("%s/%s", projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, ":\"user\"", fmt.Sprintf(":\"%s\"", cloud.Username))

	return kubeConfigData
}
