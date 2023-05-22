package cce

import (
	"errors"
	"fmt"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
	"strconv"
	"strings"
)

func GetClusterNames(projectName string) config.Clusters {
	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	var c config.Clusters
	for _, item := range clustersResult {
		c = append(c, config.Cluster{
			Name: item.Metadata.Name,
			Id:   item.Metadata.Id,
		})
	}

	config.UpdateClusters(c)
	println(fmt.Sprintf("CCE Clusters for project %s:\n%s", projectName, strings.Join(c.GetClusterNames(), ",\n")))
	return c
}

func GetKubeConfig(configParams KubeConfigParams) {
	kubeConfig := getKubeConfig(configParams)

	mergeKubeConfig(configParams, kubeConfig)

	println(fmt.Sprintf("Successfully fetched and merge kube config for cce cluster %s.", configParams.ClusterName))
}

func getClustersForProjectFromServiceProvider(projectName string) ([]clusters.Clusters, error) {
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.Id,
		TokenID:          project.ScopedToken.Secret,
		TenantID:         project.Id,
	})
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewCCE(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	return clusters.List(client, clusters.ListOpts{})
}

func getClusterCertFromServiceProvider(projectName string, clusterId string, duration string) (*clusters.Certificate, error) {
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.Id,
		TokenID:          project.ScopedToken.Secret,
		TenantID:         project.Id,
	})
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewCCE(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	var expOpts clusters.ExpirationOpts
	expOpts.Duration, err = strconv.Atoi(duration)
	if err != nil {
		return nil, err
	}
	cert := clusters.GetCertWithExpiration(client, clusterId, expOpts)
	return cert.Extract()
}

func getClusterId(clusterName string, projectName string) (clusterId string, err error) {
	cloud := config.GetActiveCloudConfig()

	if cloud.Clusters.ContainsClusterByName(clusterName) {
		return cloud.Clusters.GetClusterByNameOrThrow(clusterName).Id, nil
	}

	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	var c config.Clusters
	for _, cluster := range clustersResult {
		c = append(c, config.Cluster{
			Name: cluster.Metadata.Name,
			Id:   cluster.Metadata.Id,
		})
	}
	println(fmt.Sprintf("Clusters for project %s:\n%s", projectName, strings.Join(c.GetClusterNames(), ",\n")))

	config.UpdateClusters(c)
	cloud = config.GetActiveCloudConfig()

	if cloud.Clusters.ContainsClusterByName(clusterName) {
		return cloud.Clusters.GetClusterByNameOrThrow(clusterName).Id, nil
	}

	errorMessage := fmt.Sprintf("cluster not found.\nhere's a list of valid clusters:\n%s", strings.Join(c.GetClusterNames(), ",\n"))
	return clusterId, errors.New(errorMessage)
}
