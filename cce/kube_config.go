package cce

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"otc-auth/config"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getKubeConfig(kubeConfigParams KubeConfigParams) string {
	log.Println("Getting kube config...")

	clusterID, err := getClusterID(kubeConfigParams.ClusterName, kubeConfigParams.ProjectName)
	if err != nil {
		log.Fatalf("fatal: error receiving cluster id: %s", err)
	}

	response, err := getClusterCertFromServiceProvider(kubeConfigParams.ProjectName, clusterID, kubeConfigParams.DaysValid)
	if err != nil {
		log.Fatal(err)
	}
	responseMarshalled, err := json.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}
	return string(responseMarshalled)
}

func mergeKubeConfig(configParams KubeConfigParams, kubeConfigData string) {
	kubeConfigContextData := addContextInformationToKubeConfig(configParams.ProjectName,
		configParams.ClusterName, kubeConfigData)
	currentConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().GetStartingConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeConfigContextData))
	if err != nil {
		log.Fatal(err)
	}
	kubeConfig, err := clientConfig.RawConfig()
	if err != nil {
		log.Fatal(err)
	}

	if configParams.Server != "" {
		kubeConfigBkp := kubeConfig
		for idx := range kubeConfigBkp.Clusters {
			kubeConfig.Clusters[idx].Server = configParams.Server
		}
	}

	filenameNewFile := "kubeConfig_new"
	filenameCurrentFile := "kubeConfig_current"

	err = clientcmd.WriteToFile(kubeConfig, filenameNewFile)
	if err != nil {
		log.Fatal(err)
	}
	err = clientcmd.WriteToFile(*currentConfig, filenameCurrentFile)
	if err != nil {
		log.Fatal(err)
	}

	loadingRules := clientcmd.ClientConfigLoadingRules{
		Precedence: []string{filenameNewFile, filenameCurrentFile},
	}

	mergedConfig, err := loadingRules.Load()
	if err != nil {
		log.Fatal(err)
	}
	err = clientcmd.WriteToFile(*mergedConfig, determineTargetLocation(configParams.TargetLocation))
	if err != nil {
		log.Fatal(err)
	}

	_ = os.RemoveAll(filenameNewFile)
	_ = os.RemoveAll(filenameCurrentFile)
}

func determineTargetLocation(targetLocation string) string {
	defaultKubeConfigLocation := path.Join(homedir.HomeDir(), ".kube", "config")
	if targetLocation != "" {
		err := os.MkdirAll(filepath.Dir(targetLocation), os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		return targetLocation
	}
	return defaultKubeConfigLocation
}

func addContextInformationToKubeConfig(projectName string, clusterName string, kubeConfigData string) string {
	cloud := config.GetActiveCloudConfig()

	kubeConfigData = strings.ReplaceAll(kubeConfigData, "internalCluster", fmt.Sprintf("%s/%s-intranet",
		projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "externalCluster", fmt.Sprintf("%s/%s", projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "internal", fmt.Sprintf("%s/%s-intranet", projectName,
		clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "external", fmt.Sprintf("%s/%s", projectName, clusterName))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, ":\"user\"",
		fmt.Sprintf(":\"%s-%s-%s\"", projectName, clusterName, cloud.Username))

	return kubeConfigData
}
