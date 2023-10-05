package cce

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetClusterNames(projectName string) config.Clusters {
	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	var clustersArr config.Clusters

	for _, item := range clustersResult {
		clustersArr = append(clustersArr, config.Cluster{
			Name: item.Metadata.Name,
			ID:   item.Metadata.Id,
		})
	}

	config.UpdateClusters(clustersArr)
	log.Printf("CCE Clusters for project %s:\n%s", projectName, strings.Join(clustersArr.GetClusterNames(), ",\n"))

	return clustersArr
}

func GetKubeConfig(configParams KubeConfigParams, skipKubeTLS bool) {
	kubeConfig, err := getKubeConfig(configParams)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	if skipKubeTLS || configParams.Server != "" {
		kubeConfigBkp := kubeConfig
		for idx := range kubeConfigBkp.Clusters {
			if skipKubeTLS {
				kubeConfig.Clusters[idx].InsecureSkipTLSVerify = true
			}
			if configParams.Server != "" {
				kubeConfig.Clusters[idx].Server = configParams.Server
			}
		}
	}

	mergeKubeConfig(configParams, kubeConfig)

	log.Printf("Successfully fetched and merge kube config for cce cluster %s. \n", configParams.ClusterName)
}

func getClustersForProjectFromServiceProvider(projectName string) ([]clusters.Clusters, error) {
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	cloud := config.GetActiveCloudConfig()
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(cloud.Region) + "/v3",
		DomainID:         cloud.Domain.ID,
		TokenID:          project.ScopedToken.Secret,
		TenantID:         project.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get provider: %w", err)
	}
	client, err := openstack.NewCCE(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get clusters for project: %w", err)
	}
	return clusters.List(client, clusters.ListOpts{})
}

func getClusterCertFromServiceProvider(kubeConfigParams KubeConfigParams, clusterID string) (api.Config, error) {
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(kubeConfigParams.ProjectName)
	cloud := config.GetActiveCloudConfig()
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(cloud.Region) + "/v3",
		DomainID:         cloud.Domain.ID,
		TokenID:          project.ScopedToken.Secret,
		TenantID:         project.ID,
	})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	client, err := openstack.NewCCE(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	var expOpts clusters.ExpirationOpts
	expOpts.Duration, err = strconv.Atoi(kubeConfigParams.DaysValid)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	cert := clusters.GetCertWithExpiration(client, clusterID, expOpts).Body
	certWithContext := addContextInformationToKubeConfig(kubeConfigParams.ProjectName,
		kubeConfigParams.ClusterName, string(cert))
	extractedCert, err := clientcmd.NewClientConfigFromBytes([]byte(certWithContext))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	return extractedCert.RawConfig()
}

func getClusterID(clusterName string, projectName string) (clusterID string, err error) {
	cloud := config.GetActiveCloudConfig()

	if cloud.Clusters.ContainsClusterByName(clusterName) {
		return cloud.Clusters.GetClusterByNameOrThrow(clusterName).ID, nil
	}

	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	var clusterArr config.Clusters
	for _, cluster := range clustersResult {
		clusterArr = append(clusterArr, config.Cluster{
			Name: cluster.Metadata.Name,
			ID:   cluster.Metadata.Id,
		})
	}
	log.Printf("Clusters for project %s:\n%s", projectName, strings.Join(clusterArr.GetClusterNames(), ",\n"))

	config.UpdateClusters(clusterArr)
	cloud = config.GetActiveCloudConfig()

	if cloud.Clusters.ContainsClusterByName(clusterName) {
		return cloud.Clusters.GetClusterByNameOrThrow(clusterName).ID, nil
	}

	errorMessage := fmt.Sprintf("cluster not found.\nhere's a list of valid clusters:\n%s",
		strings.Join(clusterArr.GetClusterNames(), ",\n"))
	return clusterID, errors.New(errorMessage)
}
