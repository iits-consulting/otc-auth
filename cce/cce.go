package cce

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	"github.com/golang/glog"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func GetClusterNames(projectName string) config.Clusters {
	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.ThrowError(err)
	}

	var clustersArr config.Clusters

	for _, item := range clustersResult {
		clustersArr = append(clustersArr, config.Cluster{
			Name: item.Metadata.Name,
			ID:   item.Metadata.Id,
		})
	}

	config.UpdateClusters(clustersArr)
	glog.V(1).Infof(
		"info: CCE clusters for project %s:\n%s",
		projectName, strings.Join(clustersArr.GetClusterNames(), ",\n"))

	return clustersArr
}

func GetKubeConfig(configParams KubeConfigParams, skipKubeTLS bool, printKubeConfig bool, alias string) {
	kubeConfig, err := getKubeConfig(configParams, alias)
	if err != nil {
		common.ThrowError(err)
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

	CheckAndWarnCertsValidity(kubeConfig)

	if printKubeConfig {
		// Create a configuration file in kubectl-compatible format
		configBytes, errMarshal := clientcmd.Write(kubeConfig)
		if errMarshal != nil {
			common.ThrowError(errMarshal)
		}
		// Output the YAML data to STDOUT, since STDERR already contains log messages
		_, err = os.Stdout.Write(configBytes)
		if err != nil {
			common.ThrowError(errors.New("error writing YAML to STDOUT"))
		}
		glog.V(1).Info("info: successfully fetched kube config for cce cluster %s. \n", configParams.ClusterName)
	} else {
		mergeKubeConfig(configParams, kubeConfig)
		glog.V(1).Infof("info: successfully fetched and Merge kube config for cce cluster %s. \n", configParams.ClusterName)
	}
}

func CheckAndWarnCertsValidity(kubeConfig api.Config) {
	var certs []*x509.Certificate
	issueFound := false

	issueFound, certs = getAuthInfoCerts(kubeConfig, issueFound, certs)

	issueFound, certs = getClusterCerts(kubeConfig, issueFound, certs)

	for _, cert := range certs {
		if cert == nil {
			log.Println("failed to parse certificate")
			continue
		}

		now := time.Now()
		switch {
		case now.Before(cert.NotBefore):
			glog.Warningf("certificate is not valid yet. certificate: %s", sprintCertInfo(cert))
			issueFound = true
		case now.After(cert.NotAfter):
			glog.Warningf("certificate expired. certificate: %s", sprintCertInfo(cert))
			issueFound = true
		default:
			//nolint:mnd // V2 since this is debug info
			glog.V(2).Infof("certificate and current time match. certificate: %s", sprintCertInfo(cert))
		}
	}

	if issueFound {
		glog.V(1).Info("issue found with kube config, please refresh it with `otc-auth cce get-kube-config`")
	}
}

func getAuthInfoCerts(kubeConfig api.Config, issueFound bool, certs []*x509.Certificate) (bool, []*x509.Certificate) {
	for _, authInfo := range kubeConfig.AuthInfos {
		p, _ := pem.Decode(authInfo.ClientCertificateData)
		if p == nil {
			glog.Warningf("can't decode authInfo certificate during expiry check. authInfo: %+v", authInfo)
			issueFound = true
			continue
		}
		nCerts, certErr := x509.ParseCertificates(p.Bytes)
		if certErr != nil {
			common.ThrowError(certErr)
		}
		//nolint:mnd // V2 since this is debug info
		glog.V(2).Infof("found certs in authInfo. cert: %+v", nCerts)
		certs = append(certs, nCerts...)
	}
	return issueFound, certs
}

func getClusterCerts(kubeConfig api.Config, issueFound bool, certs []*x509.Certificate) (bool, []*x509.Certificate) {
	for _, cluster := range kubeConfig.Clusters {
		p, _ := pem.Decode(cluster.CertificateAuthorityData)
		if p == nil {
			if cluster.InsecureSkipTLSVerify {
				continue
			}
			glog.Warningf("can't decode cluster authority certificate during expiry check. cluster: %+v", cluster)
			issueFound = true
			continue
		}
		nCerts, certErr := x509.ParseCertificates(p.Bytes)
		if certErr != nil {
			common.ThrowError(certErr)
		}
		//nolint:mnd // V2 since this is debug info
		glog.V(2).Infof("found certs in cluster. cert: %+v", nCerts)
		certs = append(certs, nCerts...)
	}
	return issueFound, certs
}

func sprintCertInfo(cert *x509.Certificate) string {
	return fmt.Sprintf(
		"&Cert{SignatureAlgorithm:%s, PublicKeyAlgorithm:%s, PublicKey:%d,"+
			" Version:%d, SerialNumber:%s, Issuer:%s, Subject:%s}",
		cert.SignatureAlgorithm, cert.PublicKeyAlgorithm, cert.PublicKey,
		cert.Version, cert.SerialNumber, cert.Issuer, cert.Subject)
}

func getClustersForProjectFromServiceProvider(projectName string) ([]clusters.Clusters, error) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	project, err := activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		common.ThrowError(err)
	}
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(activeCloud.Region),
		DomainID:         activeCloud.Domain.ID,
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

func getClusterCertFromServiceProvider(kubeConfigParams KubeConfigParams,
	clusterID string, alias string,
) (api.Config, error) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	project, err := activeCloud.Projects.GetProjectByName(kubeConfigParams.ProjectName)
	if err != nil {
		common.ThrowError(err)
	}
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(activeCloud.Region),
		DomainID:         activeCloud.Domain.ID,
		TokenID:          project.ScopedToken.Secret,
		TenantID:         project.ID,
	})
	if err != nil {
		common.ThrowError(err)
	}
	client, err := openstack.NewCCE(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.ThrowError(err)
	}

	var expOpts clusters.ExpirationOpts
	expOpts.Duration, err = strconv.Atoi(kubeConfigParams.DaysValid)
	if err != nil {
		common.ThrowError(err)
	}
	cert, err := clusters.GetCertWithExpiration(client, clusterID, expOpts)
	if err != nil {
		common.ThrowError(err)
	}
	certBytes, err := yaml.Marshal(cert)
	if err != nil {
		common.ThrowError(err)
	}
	certWithContext := addContextInformationToKubeConfig(kubeConfigParams.ProjectName,
		kubeConfigParams.ClusterName, string(certBytes), alias)
	extractedCert, err := clientcmd.NewClientConfigFromBytes([]byte(certWithContext))
	if err != nil {
		common.ThrowError(err)
	}
	return extractedCert.RawConfig()
}

func getClusterID(clusterName string, projectName string) (clusterID string, err error) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}

	clusterArr := getRefreshedClusterArr(projectName)

	if !activeCloud.Clusters.ContainsClusterByName(clusterName) {
		config.UpdateClusters(clusterArr)
		activeCloud, err = config.GetActiveCloudConfig()
		if err != nil {
			common.ThrowError(err)
		}
	}

	cluster, err := activeCloud.Clusters.GetClusterByName(clusterName)
	if err != nil {
		return "", err
	}

	return cluster.ID, nil
}

func getRefreshedClusterArr(projectName string) config.Clusters {
	clustersResult, err := getClustersForProjectFromServiceProvider(projectName)
	if err != nil {
		common.ThrowError(err)
	}

	var clusterArr config.Clusters
	for _, cluster := range clustersResult {
		clusterArr = append(clusterArr, config.Cluster{
			Name: cluster.Metadata.Name,
			ID:   cluster.Metadata.Id,
		})
	}
	glog.V(1).Info("info: clusters for project %s:\n%s", projectName, strings.Join(clusterArr.GetClusterNames(), ",\n"))
	return clusterArr
}
