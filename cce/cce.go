package cce

import (
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/go-http-utils/headers"
	"log"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	"otc-auth/common/xheaders"
	"otc-auth/config"
	"otc-auth/iam"
	"strings"
	"time"
)

func GetClusterNames(projectName string) config.Clusters {
	clustersResult := getClustersForProjectFromServiceProvider(projectName)
	var clusters config.Clusters
	for _, item := range clustersResult.Items {
		clusters = append(clusters, config.Cluster{
			Name: item.Metadata.Name,
			Id:   item.Metadata.UID,
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

func GetProjectsInActiveCloud() {
	projectsResponse := getProjectsFromServiceProvider()
	var projects config.Projects
	for _, project := range projectsResponse.Projects {
		projects = append(projects, config.Project{
			NameAndIdResource: config.NameAndIdResource{Name: project.Name, Id: project.Id},
		})
	}

	config.UpdateProjects(projects)
	println(fmt.Sprintf("Projects for active cloud:\n%s", strings.Join(projects.GetProjectNames(), ",\n")))
}

func getProjectsFromServiceProvider() (projectsResponse common.ProjectsResponse) {
	cloud := config.GetActiveCloudConfig()
	println(fmt.Sprintf("info: fetching projects for cloud %s", cloud.Domain.Name))

	request := common.GetRequest(http.MethodGet, endpoints.IamProjects, nil)
	request.Header.Add(headers.ContentType, headervalues.ApplicationJson)
	request.Header.Add(xheaders.XAuthToken, cloud.UnscopedToken.Secret)

	response := common.HttpClientMakeRequest(request)
	bodyBytes := common.GetBodyBytesFromResponse(response)
	projectsResponse = *common.DeserializeJsonForType[common.ProjectsResponse](bodyBytes)

	return projectsResponse
}

func getClustersForProjectFromServiceProvider(projectName string) common.ClustersResponse {
	clustersResponse := common.ClustersResponse{}
	cloud := config.GetActiveCloudConfig()
	project := cloud.Projects.FindProjectByName(projectName)
	if project == nil {
		GetProjectsInActiveCloud()
		cloud = config.GetActiveCloudConfig()
		verifiedProject := cloud.Projects.GetProjectByNameOrThrow(projectName)
		project = &verifiedProject
	}

	err := retry.Do(
		func() error {
			infoMessage := fmt.Sprintf("info: fetching clusters for project %s", projectName)
			println(infoMessage)
			request := common.GetRequest(http.MethodGet, endpoints.Clusters(project.Id), nil)
			request.Header.Add(headers.ContentType, headervalues.ApplicationJson)
			scopedToken := iam.GetScopedToken(projectName)
			request.Header.Add(xheaders.XAuthToken, scopedToken.Secret)

			response := common.HttpClientMakeRequest(request)
			bodyBytes := common.GetBodyBytesFromResponse(response)

			clustersResponse = *common.DeserializeJsonForType[common.ClustersResponse](bodyBytes)
			return nil
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("#%d: %s\n", n, err)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(time.Second*2),
	)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	return clustersResponse
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

	clustersResult := getClustersForProjectFromServiceProvider(projectName)

	var clusters config.Clusters
	for _, cluster := range clustersResult.Items {
		clusters = append(clusters, config.Cluster{
			Name: cluster.Metadata.Name,
			Id:   cluster.Metadata.UID,
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
