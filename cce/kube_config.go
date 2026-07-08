package cce

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"otc-auth/common"
	"otc-auth/config"

	"github.com/golang/glog"
	"github.com/imdario/mergo"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

const (
	internalClusterName = "internalCluster"
	externalClusterName = "externalCluster"
)

func getKubeConfig(kubeConfigParams KubeConfigParams, alias string) (*api.Config, error) {
	glog.V(common.InfoLogLevel).Infof("info: getting kube config...")

	clusterID, err := getClusterID(kubeConfigParams.ClusterName, kubeConfigParams.ProjectName)
	if err != nil {
		common.ThrowError(fmt.Errorf("fatal: error receiving cluster id: %w", err))
	}

	return getKubeConfFromServiceProvider(kubeConfigParams, clusterID, alias)
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
	if err != nil {
		return err
	}
	// mergo deep-merges colliding entries and never overrides with empty
	// values, so a re-fetched entry could keep stale CA data next to a
	// freshly-set insecure-skip-tls-verify flag (a combination client-go
	// rejects) and a once-set flag could never be cleared. Freshly fetched
	// entries are authoritative: replace them wholesale.
	for name, cluster := range kubeConfig.Clusters {
		currentConfig.Clusters[name] = cluster
	}
	for name, authInfo := range kubeConfig.AuthInfos {
		currentConfig.AuthInfos[name] = authInfo
	}
	for name, context := range kubeConfig.Contexts {
		currentConfig.Contexts[name] = context
	}
	return nil
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

func renameKubeconfigEntries(rawConfig *api.Config, projectName, clusterName, alias string) error {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		return err
	}

	if alias == "" {
		alias = fmt.Sprintf("%s/%s", projectName, clusterName)
	}

	clusterRenames := map[string]string{
		internalClusterName: fmt.Sprintf("%s-intranet", alias),
		externalClusterName: alias,
	}
	userRenames := map[string]string{
		"user": fmt.Sprintf("%s-%s-%s", projectName, clusterName, activeCloud.Username),
	}
	contextRenames := map[string]string{
		"internal": fmt.Sprintf("%s-intranet", alias),
		"external": alias,
	}

	for oldName, newName := range clusterRenames {
		if val, exists := rawConfig.Clusters[oldName]; exists {
			rawConfig.Clusters[newName] = val
			delete(rawConfig.Clusters, oldName)
		}
	}
	for oldName, newName := range userRenames {
		if val, exists := rawConfig.AuthInfos[oldName]; exists {
			rawConfig.AuthInfos[newName] = val
			delete(rawConfig.AuthInfos, oldName)
		}
	}

	for _, context := range rawConfig.Contexts {
		if newName, ok := clusterRenames[context.Cluster]; ok {
			context.Cluster = newName
		}
		if newName, ok := userRenames[context.AuthInfo]; ok {
			context.AuthInfo = newName
		}
	}

	for oldName, newName := range contextRenames {
		if val, exists := rawConfig.Contexts[oldName]; exists {
			rawConfig.Contexts[newName] = val
			delete(rawConfig.Contexts, oldName)
		}
	}
	if newName, ok := contextRenames[rawConfig.CurrentContext]; ok {
		rawConfig.CurrentContext = newName
	}

	return nil
}
