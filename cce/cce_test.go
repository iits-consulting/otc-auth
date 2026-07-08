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
		Server:                "https://80.158.0.1:5443",
		InsecureSkipTLSVerify: true,
	}
	// CA data and the insecure flag are mutually exclusive in the output:
	// client-go rejects entries carrying both
	bothSet := clusters.CertClusters{Name: externalClusterName, Cluster: clusters.CertCluster{
		Server:                "https://80.158.0.1:5443",
		CertAuthorityData:     caDataB64,
		InsecureSkipTLSVerify: true,
	}}
	wantBothSet := &api.Cluster{
		Server:                   "https://80.158.0.1:5443",
		CertificateAuthorityData: []byte(caData),
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
		{
			name:     "CA data wins over insecure flag when the API sets both",
			clusters: []clusters.CertClusters{bothSet},
			want:     map[string]*api.Cluster{externalClusterName: wantBothSet},
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

// certToKubeConfig maps the SDK cert structs field-by-field; a field added to
// the SDK would otherwise be dropped silently (how insecure-skip-tls-verify
// went missing in the first place, see PR #182). This pins the field sets so
// the mapping must be revisited when the SDK grows.
func Test_certToKubeConfig_coversAllSDKFields(t *testing.T) {
	t.Parallel()

	handled := map[reflect.Type][]string{
		reflect.TypeOf(clusters.CertCluster{}): {"Server", "CertAuthorityData", "InsecureSkipTLSVerify"},
		reflect.TypeOf(clusters.CertUser{}):    {"ClientCertData", "ClientKeyData"},
		reflect.TypeOf(clusters.CertContext{}): {"Cluster", "User"},
	}

	for typ, fields := range handled {
		known := make(map[string]bool, len(fields))
		for _, f := range fields {
			known[f] = true
		}
		for i := range typ.NumField() {
			if name := typ.Field(i).Name; !known[name] {
				t.Errorf("%s.%s is not mapped by certToKubeConfig — update the mapping and this list", typ.Name(), name)
			}
		}
		if typ.NumField() < len(fields) {
			t.Errorf("%s lost fields; update the handled list", typ.Name())
		}
	}
}
