package cce

import (
	"errors"
	"fmt"
	"github.com/go-http-utils/headers"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	"otc-auth/common/xheaders"
	"otc-auth/config"
	"strings"
)

func GetClusterNames(projectName string) config.Clusters {
	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	var clusters config.Clusters
	for _, item := range clustersResult {
		clusters = append(clusters, config.Cluster{
			Name: item.Metadata.Name,
			Id:   item.Metadata.Id,
		})
	}

	config.UpdateClusters(clusters)
	println(fmt.Sprintf("CCE Clusters for project %s:\n%s", projectName, strings.Join(clusters.GetClusterNames(), ",\n")))
	return clusters
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

func getClusterCertFromServiceProvider(projectName string, clusterId string, duration string) (response *http.Response) {
	body := fmt.Sprintf("{\"duration\": %s}", duration)
	projectId := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName).Id

	request := common.GetRequest(http.MethodPost, endpoints.ClusterCert(projectId, clusterId), strings.NewReader(body))
	request.Header.Add(headers.ContentType, headervalues.ApplicationJson)
	request.Header.Add(headers.Accept, headervalues.ApplicationJson)
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	request.Header.Add(xheaders.XAuthToken, project.ScopedToken.Secret)

	response = common.HttpClientMakeRequest(request)

	return response
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

	var clusters config.Clusters
	for _, cluster := range clustersResult {
		clusters = append(clusters, config.Cluster{
			Name: cluster.Metadata.Name,
			Id:   cluster.Metadata.Id,
		})
	}
	println(fmt.Sprintf("Clusters for project %s:\n%s", projectName, strings.Join(clusters.GetClusterNames(), ",\n")))

	config.UpdateClusters(clusters)
	cloud = config.GetActiveCloudConfig()

	if cloud.Clusters.ContainsClusterByName(clusterName) {
		return cloud.Clusters.GetClusterByNameOrThrow(clusterName).Id, nil
	}

	errorMessage := fmt.Sprintf("cluster not found.\nhere's a list of valid clusters:\n%s", strings.Join(clusters.GetClusterNames(), ",\n"))
	return clusterId, errors.New(errorMessage)
}
