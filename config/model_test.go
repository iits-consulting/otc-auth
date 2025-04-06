package config_test

import (
	"fmt"
	"strings"
	"testing"

	"otc-auth/config"
)

func TestGetClusterByName(t *testing.T) {
	clusters := config.Clusters{
		{Name: "ClusterA"},
		{Name: "ClusterB"},
		{Name: "ClusterC"},
	}

	tests := []struct {
		name          string
		clusters      config.Clusters
		searchName    string
		expectedError bool
		expectedName  string
	}{
		{
			name:          "Cluster Exists",
			clusters:      clusters,
			searchName:    "ClusterB",
			expectedError: false,
			expectedName:  "ClusterB",
		},
		{
			name:          "Cluster Does Not Exist",
			clusters:      clusters,
			searchName:    "ClusterD",
			expectedError: true,
			expectedName:  "",
		},
		{
			name:          "Case Sensitivity",
			clusters:      clusters,
			searchName:    "clustera",
			expectedError: true,
			expectedName:  "",
		},
		{
			name:          "Empty Name",
			clusters:      clusters,
			searchName:    "",
			expectedError: true,
			expectedName:  "",
		},
		{
			name:          "Empty Clusters Object",
			clusters:      config.Clusters{},
			searchName:    "ClusterA",
			expectedError: true,
			expectedName:  "",
		},
		{
			name:          "Multiple Clusters",
			clusters:      clusters,
			searchName:    "ClusterC",
			expectedError: false,
			expectedName:  "ClusterC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := tt.clusters.GetClusterByName(tt.searchName)

			if tt.expectedError {
				assertError(t, err, tt.clusters)
			} else {
				assertCluster(t, cluster, tt.expectedName, err)
			}
		})
	}
}

func assertError(t *testing.T, err error, clusters config.Clusters) {
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	expectedMessage := fmt.Sprintf("cluster not found.\nhere's a list of valid clusters:\n%s",
		strings.Join(clusters.GetClusterNames(), ",\n"))

	if err.Error() != expectedMessage {
		t.Errorf("unexpected error message: got %q, want %q", err.Error(), expectedMessage)
	}
}

func assertCluster(t *testing.T, cluster *config.Cluster, expectedName string, err error) {
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if cluster.Name != expectedName {
		t.Errorf("unexpected cluster name: got %q, want %q", cluster.Name, expectedName)
	}
}
