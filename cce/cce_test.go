//nolint:testpackage // whitebox testing
package cce

import (
	"encoding/base64"
	"testing"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
)

func Test_certToKubeConfig(t *testing.T) {
	t.Parallel()

	caData := "fake-ca-data"
	caDataB64 := base64.StdEncoding.EncodeToString([]byte(caData))

	tests := []struct {
		name         string
		clusters     []clusters.CertClusters
		wantInsecure map[string]bool
		wantCAData   map[string]string
	}{
		{
			name: "Jumbo setup without EIP: CA data only, no insecure flag",
			clusters: []clusters.CertClusters{
				{Name: "internalCluster", Cluster: clusters.CertCluster{
					Server:            "https://192.168.0.1:5443",
					CertAuthorityData: caDataB64,
				}},
			},
			wantInsecure: map[string]bool{"internalCluster": false},
			wantCAData:   map[string]string{"internalCluster": caData},
		},
		{
			name: "EIP bound: externalCluster has insecure-skip-tls-verify",
			clusters: []clusters.CertClusters{
				{Name: "internalCluster", Cluster: clusters.CertCluster{
					Server:            "https://192.168.0.1:5443",
					CertAuthorityData: caDataB64,
				}},
				{Name: "externalCluster", Cluster: clusters.CertCluster{
					Server:                "https://80.158.0.1:5443",
					InsecureSkipTLSVerify: true,
				}},
			},
			wantInsecure: map[string]bool{"internalCluster": false, "externalCluster": true},
			wantCAData:   map[string]string{"internalCluster": caData, "externalCluster": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cert := &clusters.Certificate{
				Kind:       "Config",
				ApiVersion: "v1",
				Clusters:   tt.clusters,
			}

			got, err := certToKubeConfig(cert)
			if err != nil {
				t.Fatalf("certToKubeConfig() error = %v", err)
			}

			if len(got.Clusters) != len(tt.clusters) {
				t.Fatalf("got %d clusters, want %d", len(got.Clusters), len(tt.clusters))
			}
			for name, wantInsecure := range tt.wantInsecure {
				cluster, ok := got.Clusters[name]
				if !ok {
					t.Fatalf("cluster %q missing from kube config", name)
				}
				if cluster.InsecureSkipTLSVerify != wantInsecure {
					t.Errorf("cluster %q InsecureSkipTLSVerify = %v, want %v",
						name, cluster.InsecureSkipTLSVerify, wantInsecure)
				}
				if string(cluster.CertificateAuthorityData) != tt.wantCAData[name] {
					t.Errorf("cluster %q CertificateAuthorityData = %q, want %q",
						name, cluster.CertificateAuthorityData, tt.wantCAData[name])
				}
			}
		})
	}
}
