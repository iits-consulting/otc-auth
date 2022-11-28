package cce

import (
	"encoding/json"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/go-http-utils/headers"
	"io"
	"log"
	"net/http"
	"otc-auth/src/common"
	"otc-auth/src/common/endpoints"
	"otc-auth/src/common/headervalues"
	"otc-auth/src/common/xheaders"
	"otc-auth/src/iam"
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
			client := common.GetHttpClient()

			projectId := common.FindProjectID(projectName)
			req, err := http.NewRequest(http.MethodGet, endpoints.Clusters(projectId), nil)
			if err != nil {
				return err
			}

			req.Header.Add(headers.ContentType, headervalues.ApplicationJson)
			scopedToken := getScopedToken(projectName)
			req.Header.Add(xheaders.XAuthToken, scopedToken)

			resp, err := client.Do(req)
			if err != nil {
				return err
			}

			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if resp.StatusCode != 200 {
				return fmt.Errorf("error: status %s, body:\n%s", resp.Status, common.ErrorMessageToIndentedJsonFormat(responseBody))
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
		retry.Delay(time.Second*2),
	)

	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	return clustersResult
}

func postClusterCert(projectName string, clusterId string, duration string) (resp *http.Response, err error) {

	body := fmt.Sprintf("{\"duration\": %s}", duration)

	projectId := common.FindProjectID(projectName)
	req, err := http.NewRequest(http.MethodPost, endpoints.ClusterCert(projectId, clusterId), strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add(headers.ContentType, headervalues.ApplicationJson)
	req.Header.Add(headers.Accept, headervalues.ApplicationJson)
	req.Header.Add(xheaders.XAuthToken, getScopedToken(projectName))

	client := common.GetHttpClient()
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

func getScopedToken(projectName string) string {
	scopedTokenFromOTCInfoFile := common.GetScopedTokenFromOTCInfo(projectName)
	if scopedTokenFromOTCInfoFile == "" {
		iam.GetScopedToken(projectName)
		return common.GetScopedTokenFromOTCInfo(projectName)
	}
	return scopedTokenFromOTCInfoFile
}
