package cce

import (
	"crypto/x509"
	"encoding/base64"
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
	glog.V(common.InfoLogLevel).Infof(
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

	CheckAndWarnCertsValidity(*kubeConfig)

	if printKubeConfig {
		// Create a configuration file in kubectl-compatible format
		configBytes, errMarshal := clientcmd.Write(*kubeConfig)
		if errMarshal != nil {
			common.ThrowError(errMarshal)
		}
		// Output the YAML data to STDOUT, since STDERR already contains log messages
		_, err = os.Stdout.Write(configBytes)
		if err != nil {
			common.ThrowError(errors.New("error writing YAML to STDOUT"))
		}
		glog.V(common.InfoLogLevel).Info("info: successfully fetched kube config for cce cluster %s. \n",
			configParams.ClusterName)
	} else {
		mergeKubeConfig(configParams, *kubeConfig)
		glog.V(common.InfoLogLevel).Infof("info: successfully fetched and Merge kube config for cce cluster %s. \n",
			configParams.ClusterName)
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
			glog.V(common.DebugLogLevel).Infof("certificate and current time match. certificate: %s",
				sprintCertInfo(cert))
		}
	}

	if issueFound {
		glog.V(common.InfoLogLevel).Info(
			"issue found with kube config, please refresh it with `otc-auth cce get-kube-config`")
	}
}

func getAuthInfoCerts(kubeConfig api.Config, issueFound bool, certs []*x509.Certificate) (bool, []*x509.Certificate) {
	for name, authInfo := range kubeConfig.AuthInfos {
		if len(authInfo.ClientCertificateData) == 0 {
			glog.V(common.DebugLogLevel).Infof(
				"Skipping cluster '%s' during expiry check: no certificate authority data present.", name)
			continue
		}
		p, rest := pem.Decode(authInfo.ClientCertificateData)
		if p == nil {
			glog.Warningf("can't decode authInfo certificate during expiry check. authInfo: %+v, rest: %+v",
				authInfo, rest)
			issueFound = true
			continue
		}
		nCerts, certErr := x509.ParseCertificates(p.Bytes)
		if certErr != nil {
			common.ThrowError(certErr)
		}
		glog.V(common.DebugLogLevel).Infof("found certs in authInfo. cert: %+v", nCerts)
		certs = append(certs, nCerts...)
	}
	return issueFound, certs
}

func getClusterCerts(kubeConfig api.Config, issueFound bool, certs []*x509.Certificate) (bool, []*x509.Certificate) {
	for name, cluster := range kubeConfig.Clusters {
		if len(cluster.CertificateAuthorityData) == 0 {
			glog.V(common.DebugLogLevel).Infof(
				"Skipping cluster '%s' during expiry check: no certificate authority data present.", name)
			continue
		}
		p, rest := pem.Decode(cluster.CertificateAuthorityData)
		if p == nil {
			if cluster.InsecureSkipTLSVerify {
				continue
			}
			glog.Warningf(
				"can't decode cluster authority certificate during expiry check. cluster: %+v, rest: %+v",
				cluster, rest)
			issueFound = true
			continue
		}
		nCerts, certErr := x509.ParseCertificates(p.Bytes)
		if certErr != nil {
			common.ThrowError(certErr)
		}
		glog.V(common.DebugLogLevel).Infof("found certs in cluster. cert: %+v", nCerts)
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

func getKubeConfFromServiceProvider(kubeConfigParams KubeConfigParams,
	clusterID string, alias string,
) (*api.Config, error) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't get active cloud: %w", err)
	}
	project, err := activeCloud.Projects.GetProjectByName(kubeConfigParams.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get project %s: %w", kubeConfigParams.ProjectName, err)
	}
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(activeCloud.Region),
		DomainID:         activeCloud.Domain.ID,
		TokenID:          project.ScopedToken.Secret,
		TenantID:         project.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get new openstack client: %w", err)
	}
	client, err := openstack.NewCCE(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get new cce client: %w", err)
	}

	var expOpts clusters.ExpirationOpts
	expOpts.Duration, err = strconv.Atoi(kubeConfigParams.DaysValid)
	if err != nil {
		return nil, fmt.Errorf("couldn't convert string to int: %w", err)
	}
	cert, err := clusters.GetCertWithExpiration(client, clusterID, expOpts)
	if err != nil {
		return nil, fmt.Errorf("couldn't get cert: %w", err)
	}

	rawConfig := api.Config{
		Kind:           cert.Kind,
		APIVersion:     cert.ApiVersion,
		Preferences:    api.Preferences{},
		Clusters:       make(map[string]*api.Cluster),
		AuthInfos:      make(map[string]*api.AuthInfo),
		Contexts:       make(map[string]*api.Context),
		CurrentContext: cert.CurrentContext,
	}

	for _, c := range cert.Clusters {
		decodedCA, errDecode := base64.StdEncoding.DecodeString(c.Cluster.CertAuthorityData)
		if errDecode != nil {
			return nil, fmt.Errorf("failed to decode cluster cert auth data for cluster '%s': %w",
				c.Name, err)
		}
		rawConfig.Clusters[c.Name] = &api.Cluster{
			Server:                   c.Cluster.Server,
			CertificateAuthorityData: decodedCA,
		}
	}

	for _, u := range cert.Users {
		decodedCert, errDecode := base64.StdEncoding.DecodeString(u.User.ClientCertData)
		if errDecode != nil {
			return nil, fmt.Errorf("failed to decode client certificate data for user '%s': %w", u.Name, err)
		}
		decodedKey, errDecode := base64.StdEncoding.DecodeString(u.User.ClientKeyData)
		if errDecode != nil {
			return nil, fmt.Errorf("failed to decode client key data for user '%s': %w", u.Name, err)
		}
		rawConfig.AuthInfos[u.Name] = &api.AuthInfo{
			ClientCertificateData: decodedCert,
			ClientKeyData:         decodedKey,
		}
	}

	for _, ctx := range cert.Contexts {
		rawConfig.Contexts[ctx.Name] = &api.Context{
			Cluster:  ctx.Context.Cluster,
			AuthInfo: ctx.Context.User,
		}
	}

	err = renameKubeconfigEntries(&rawConfig, kubeConfigParams.ProjectName, kubeConfigParams.ClusterName, alias)
	if err != nil {
		return nil, fmt.Errorf("couldn't rename entries: %w", err)
	}
	return &rawConfig, nil
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
	glog.V(common.InfoLogLevel).Info("info: clusters for project %s:\n%s",
		projectName, strings.Join(clusterArr.GetClusterNames(), ",\n"))
	return clusterArr
}
