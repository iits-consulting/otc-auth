package cce

import (
	"io"
	. "k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"otc-auth/src/common"
	"strings"
)

const CceUrl = "https://cce.eu-de.otc.t-systems.com:443"

func getKubeConfig(kubeConfigParams KubeConfigParams) string {
	println("Getting kube config...\n")

	clusterId, err := getClusterId(kubeConfigParams.ClusterName, kubeConfigParams.ProjectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error receiving cluster ID: %s")
	}

	kubeConfigResponse, err := postClusterCert(kubeConfigParams.ProjectName, clusterId, kubeConfigParams.DaysValid)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error receiving cluster certificate: %s")
	}

	kubeConfigContextData, err := io.ReadAll(kubeConfigResponse.Body)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	return string(kubeConfigContextData)
}

func mergeKubeConfig(projectName string, clusterName string, newKubeConfigData string) {
	newKubeConfigContextData := addContextInformationToKubeConfig(projectName, clusterName, newKubeConfigData)
	currentConfig, err := NewDefaultClientConfigLoadingRules().GetStartingConfig()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	newClientConfig, err := NewClientConfigFromBytes([]byte(newKubeConfigContextData))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	newKubeConfig, err := newClientConfig.RawConfig()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	newKubeConfigFileName := "newKubeContext"
	currentKubeConfigFileName := "currentConfig"

	err = WriteToFile(newKubeConfig, newKubeConfigFileName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	err = WriteToFile(*currentConfig, currentKubeConfigFileName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	loadingRules := ClientConfigLoadingRules{
		Precedence: []string{newKubeConfigFileName, currentKubeConfigFileName},
	}

	mergedConfig, err := loadingRules.Load()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	err = WriteToFile(*mergedConfig, homedir.HomeDir()+"/.kube/config")
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	os.RemoveAll(newKubeConfigFileName)
	os.RemoveAll(currentKubeConfigFileName)
}

func addContextInformationToKubeConfig(projectName string, clusterName string, newKubeConfigData string) string {
	otcInfo := common.ReadOrCreateOTCAuthCredentialsFile()
	newKubeConfigData = strings.ReplaceAll(newKubeConfigData, "internalCluster", projectName+"/"+clusterName+"-intranet")
	newKubeConfigData = strings.ReplaceAll(newKubeConfigData, "externalCluster", projectName+"/"+clusterName)
	newKubeConfigData = strings.ReplaceAll(newKubeConfigData, "internal", projectName+"/"+clusterName+"-intranet")
	newKubeConfigData = strings.ReplaceAll(newKubeConfigData, "external", projectName+"/"+clusterName)
	newKubeConfigData = strings.ReplaceAll(newKubeConfigData, ":\"user\"", ":\""+otcInfo.Username+"\"")
	return newKubeConfigData
}
