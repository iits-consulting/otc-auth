//nolint:testpackage // whitebox testing
package cce

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/cce/v3/clusters"
	"k8s.io/client-go/tools/clientcmd/api"
)

func Test_certToKubeConfig(t *testing.T) {
	t.Parallel()

	caData := "fake-ca-data"
	caDataB64 := base64.StdEncoding.EncodeToString([]byte(caData))
	internal := clusters.CertClusters{Name: internalClusterName, Cluster: clusters.CertCluster{
		Server:            "https://192.168.0.1:5443",
		CertAuthorityData: caDataB64,
	}}
	external := clusters.CertClusters{Name: externalClusterName, Cluster: clusters.CertCluster{
		Server:                "https://80.158.0.1:5443",
		InsecureSkipTLSVerify: true,
	}}
	wantInternal := &api.Cluster{
		Server:                   "https://192.168.0.1:5443",
		CertificateAuthorityData: []byte(caData),
	}
	wantExternal := &api.Cluster{
		Server: "https://80.158.0.1:5443",
		// decoding the empty CA string yields an empty, non-nil slice
		CertificateAuthorityData: []byte{},
		InsecureSkipTLSVerify:    true,
	}

	tests := []struct {
		name     string
		clusters []clusters.CertClusters
		want     map[string]*api.Cluster
	}{
		{
			name:     "Jumbo setup without EIP: CA data only, no insecure flag",
			clusters: []clusters.CertClusters{internal},
			want:     map[string]*api.Cluster{internalClusterName: wantInternal},
		},
		{
			name:     "EIP bound: externalCluster has insecure-skip-tls-verify",
			clusters: []clusters.CertClusters{internal, external},
			want: map[string]*api.Cluster{
				internalClusterName: wantInternal,
				externalClusterName: wantExternal,
			},
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

			if !reflect.DeepEqual(got.Clusters, tt.want) {
				t.Errorf("certToKubeConfig() clusters mismatch:\ngot = %+v\nwant = %+v", got.Clusters, tt.want)
			}
		})
	}
}
