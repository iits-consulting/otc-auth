package cce

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"io"
	"log"
	"net/http"
	"otc-auth/src/iam"
	"otc-auth/src/util"
	"strconv"
	"strings"
	"time"
)

func GetClusterNames(projectName string) []string {
	clustersResult := getClusters(projectName)
	var clusterNames []string
	for i := range clustersResult.Items {
		clusterNames = append(clusterNames, clustersResult.Items[i].Metadata.Name)
	}
	return clusterNames
}

func GetKubeConfig(kubeConfigParams KubeConfigParams) string {
	return getKubeConfig(kubeConfigParams)
}

func MergeKubeConfig(projectName string, clusterName string, newKubeConfigData string) {
	mergeKubeConfig(projectName, clusterName, newKubeConfigData)
}

func getClusters(projectName string) GetClustersResult {
	clustersResult := GetClustersResult{}
	err := retry.Do(
		func() error {
			client := iam.GetHttpClient()

			projectId := iam.GetProjectId(projectName)
			req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v3/projects/%s/clusters", CceUrl, projectId), nil)
			if err != nil {
				return err
			}

			req.Header.Add("Content-Type", util.JsonContentType)
			scopedToken := iam.GetScopedToken(projectName)
			req.Header.Add("X-Auth-Token", scopedToken)

			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return errors.New("Statuscode=" + strconv.Itoa(resp.StatusCode) + "," + string(responseBody))
			}

			err = json.Unmarshal(responseBody, &clustersResult)
			if err != nil {
				return err
			}
			return nil
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("#%d: %s\n", n, err)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(2*time.Second),
	)

	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	return clustersResult
}

func postClusterCert(projectName string, clusterId string, duration string) (resp *http.Response, err error) {

	body := fmt.Sprintf("{\"duration\": %s}", duration)

	projectId := util.FindProjectID(projectName)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v3/projects/%s/clusters/%s/clustercert", CceUrl, projectId, clusterId), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", util.JsonContentType)
	req.Header.Add("Accept", util.JsonContentType)
	req.Header.Add("X-Auth-Token", iam.GetScopedToken(projectName))

	client := iam.GetHttpClient()
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func getClusterId(clusterName string, projectName string) (clusterId string, err error) {
	clustersResult := getClusters(projectName)

	for i := range clustersResult.Items {
		cluster := clustersResult.Items[i]
		if cluster.Metadata.Name == clusterName {
			clusterId = cluster.Metadata.UID
			break
		}
	}
	return clusterId, err
}
