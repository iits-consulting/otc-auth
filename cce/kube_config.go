package cce

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"otc-auth/common"
	"otc-auth/config"

	"github.com/golang/glog"
	"github.com/imdario/mergo"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

func getKubeConfig(kubeConfigParams KubeConfigParams, alias string) (api.Config, error) {
	glog.V(1).Infof("info: getting kube config...")

	clusterID, err := getClusterID(kubeConfigParams.ClusterName, kubeConfigParams.ProjectName)
	if err != nil {
		common.ThrowError(fmt.Errorf("fatal: error receiving cluster id: %w", err))
	}

	return getClusterCertFromServiceProvider(kubeConfigParams, clusterID, alias)
}

func mergeKubeConfig(configParams KubeConfigParams, kubeConfig api.Config) {
	currentConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().GetStartingConfig()
	if err != nil {
		common.ThrowError(err)
	}
	err = merge(currentConfig, kubeConfig)
	if err != nil {
		common.ThrowError(err)
	}
	err = clientcmd.WriteToFile(*currentConfig, determineTargetLocation(configParams.TargetLocation))
	if err != nil {
		common.ThrowError(err)
	}
}

func merge(currentConfig *api.Config, kubeConfig api.Config) error {
	err := mergo.Merge(currentConfig, kubeConfig, mergo.WithOverride)
	return err
}

func determineTargetLocation(targetLocation string) string {
	defaultKubeConfigLocation := path.Join(homedir.HomeDir(), ".kube", "config")
	if targetLocation != "" {
		err := os.MkdirAll(filepath.Dir(targetLocation), os.ModePerm)
		if err != nil {
			common.ThrowError(err)
		}
		return targetLocation
	}
	return defaultKubeConfigLocation
}

func addContextInformationToKubeConfig(projectName string, clusterName string,
	kubeConfigData string, alias string,
) string {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}

	if alias == "" {
		alias = fmt.Sprintf("%s/%s", projectName, clusterName)
	}

	kubeConfigData = strings.ReplaceAll(kubeConfigData, "internalCluster", fmt.Sprintf("%s-intranet",
		alias))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "externalCluster", alias)
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "internal", fmt.Sprintf("%s-intranet", alias))
	kubeConfigData = strings.ReplaceAll(kubeConfigData, "external", alias)
	kubeConfigData = strings.ReplaceAll(kubeConfigData, ":\"user\"",
		fmt.Sprintf(":\"%s-%s-%s\"", projectName, clusterName, activeCloud.Username))

	return kubeConfigData
}
