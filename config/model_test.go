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

func TestClouds_ContainsCloud(t *testing.T) {
	type args struct {
		name string
	}
	testClouds := config.Clouds{
		{Domain: config.NameAndIDResource{Name: "cloud1"}},
		{Domain: config.NameAndIDResource{Name: "cloud2"}},
	}
	tests := []struct {
		name   string
		clouds config.Clouds
		args   args
		want   bool
	}{
		{
			name:   "existing cloud",
			clouds: testClouds,
			args:   args{name: "cloud1"},
			want:   true,
		},
		{
			name:   "another existing cloud",
			clouds: testClouds,
			args:   args{name: "cloud2"},
			want:   true,
		},
		{
			name:   "non-existent cloud",
			clouds: testClouds,
			args:   args{name: "cloud3"},
			want:   false,
		},
		{
			name:   "empty clouds list",
			clouds: config.Clouds{},
			args:   args{name: "any"},
			want:   false,
		},
		{
			name:   "case sensitive match",
			clouds: testClouds,
			args:   args{name: "CLOUD1"}, // assuming case-sensitive comparison
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.clouds.ContainsCloud(tt.args.name); got != tt.want {
				t.Errorf("ContainsCloud() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClouds_RemoveCloudByNameIfExists(t *testing.T) {
	// Test helper function to create a cloud with a name
	makeCloud := func(name string) config.Cloud {
		return config.Cloud{Domain: config.NameAndIDResource{Name: name}}
	}

	type args struct {
		name string
	}
	tests := []struct {
		name     string
		clouds   config.Clouds
		args     args
		expected config.Clouds
	}{
		{
			name:     "empty clouds",
			clouds:   config.Clouds{},
			args:     args{name: "test"},
			expected: config.Clouds{},
		},
		{
			name:     "cloud not found",
			clouds:   config.Clouds{makeCloud("cloud1"), makeCloud("cloud2")},
			args:     args{name: "nonexistent"},
			expected: config.Clouds{makeCloud("cloud1"), makeCloud("cloud2")},
		},
		{
			name:     "remove single cloud",
			clouds:   config.Clouds{makeCloud("cloud1"), makeCloud("cloud2")},
			args:     args{name: "cloud1"},
			expected: config.Clouds{makeCloud("cloud2")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original slice
			original := make(config.Clouds, len(tt.clouds))
			copy(original, tt.clouds)

			// Execute the function
			original.RemoveCloudByNameIfExists(tt.args.name)

			// Verify the result
			if len(original) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(original))
			}

			for i, cloud := range original {
				if cloud.Domain.Name != tt.expected[i].Domain.Name {
					t.Errorf("at index %d: expected %s, got %s",
						i, tt.expected[i].Domain.Name, cloud.Domain.Name)
				}
			}
		})
	}
}
