package config_test

import (
	"fmt"
	"reflect"
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
			args:   args{name: "CLOUD1"},
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

func TestClouds_SetActiveByName(t *testing.T) {
	type args struct {
		name string
	}
	cloud1 := config.Cloud{Domain: config.NameAndIDResource{Name: "cloud1"}, Active: false}
	cloud2 := config.Cloud{Domain: config.NameAndIDResource{Name: "cloud2"}, Active: true}
	cloud3 := config.Cloud{Domain: config.NameAndIDResource{Name: "cloud3"}, Active: false}

	tests := []struct {
		name   string
		clouds config.Clouds
		args   args
		want   []bool // Expected Active statuses in order
	}{
		{
			name:   "set existing cloud active",
			clouds: config.Clouds{cloud1, cloud2, cloud3},
			args:   args{name: "cloud2"},
			want:   []bool{false, true, false},
		},
		{
			name:   "set first cloud active",
			clouds: config.Clouds{cloud1, cloud2, cloud3},
			args:   args{name: "cloud1"},
			want:   []bool{true, false, false},
		},
		{
			name:   "set last cloud active",
			clouds: config.Clouds{cloud1, cloud2, cloud3},
			args:   args{name: "cloud3"},
			want:   []bool{false, false, true},
		},
		{
			name:   "non-existent cloud name",
			clouds: config.Clouds{cloud1, cloud2, cloud3},
			args:   args{name: "unknown"},
			want:   []bool{false, false, false},
		},
		{
			name:   "empty clouds list",
			clouds: config.Clouds{},
			args:   args{name: "any"},
			want:   []bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of the original slice for testing
			cloudsCopy := make(config.Clouds, len(tt.clouds))
			copy(cloudsCopy, tt.clouds)

			cloudsCopy.SetActiveByName(tt.args.name)

			// Verify Active statuses
			for i, cloud := range cloudsCopy {
				if cloud.Active != tt.want[i] {
					t.Errorf("Cloud %d Active = %v, want %v", i, cloud.Active, tt.want[i])
				}
			}
		})
	}
}

func TestClouds_FindActiveCloudConfigOrNil(t *testing.T) {
	// Helper function to create int pointer
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name      string
		clouds    config.Clouds
		wantCloud *config.Cloud
		wantIndex *int
		wantErr   bool
	}{
		{
			name:      "no clouds",
			clouds:    config.Clouds{},
			wantCloud: nil,
			wantIndex: nil,
			wantErr:   true,
		},
		{
			name: "single active cloud",
			clouds: config.Clouds{
				{Region: "east", Active: true},
			},
			wantCloud: &config.Cloud{Region: "east", Active: true},
			wantIndex: intPtr(0),
			wantErr:   false,
		},
		{
			name: "multiple clouds with one active",
			clouds: config.Clouds{
				{Region: "west", Active: false},
				{Region: "east", Active: true},
				{Region: "north", Active: false},
			},
			wantCloud: &config.Cloud{Region: "east", Active: true},
			wantIndex: intPtr(1),
			wantErr:   false,
		},
		{
			name: "no active cloud",
			clouds: config.Clouds{
				{Region: "west", Active: false},
				{Region: "east", Active: false},
			},
			wantCloud: nil,
			wantIndex: nil,
			wantErr:   true,
		},
		{
			name: "multiple active clouds",
			clouds: config.Clouds{
				{Region: "west", Active: true},
				{Region: "east", Active: true},
			},
			wantCloud: nil,
			wantIndex: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCloud, gotIndex, err := tt.clouds.FindActiveCloudConfigOrNil()
			if (err != nil) != tt.wantErr {
				t.Errorf("FindActiveCloudConfigOrNil() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCloud, tt.wantCloud) {
				t.Errorf("FindActiveCloudConfigOrNil() gotCloud = %v, want %v", gotCloud, tt.wantCloud)
			}
			if !reflect.DeepEqual(gotIndex, tt.wantIndex) {
				t.Errorf("FindActiveCloudConfigOrNil() gotIndex = %v, want %v", gotIndex, tt.wantIndex)
			}
		})
	}
}

func TestClouds_GetActiveCloudIndex(t *testing.T) {
	tests := []struct {
		name    string
		clouds  config.Clouds
		want    int
		wantErr bool
	}{
		{
			name: "single active cloud",
			clouds: config.Clouds{
				config.Cloud{Active: true},
			},
			want: 0,
		},
		{
			name: "multiple clouds with second active",
			clouds: config.Clouds{
				config.Cloud{Active: false},
				config.Cloud{Active: true},
				config.Cloud{Active: false},
			},
			want: 1,
		},
		{
			name: "no active cloud",
			clouds: config.Clouds{
				config.Cloud{Active: false},
				config.Cloud{Active: false},
			},
			wantErr: true,
		},
		{
			name:    "empty clouds",
			clouds:  config.Clouds{},
			wantErr: true,
		},
		{
			name: "multiple active clouds (invalid state)",
			clouds: config.Clouds{
				config.Cloud{Active: true},
				config.Cloud{Active: true},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if r := recover(); r != nil {
				if !tt.wantErr {
					t.Errorf("GetActiveCloudIndex() panicked unexpectedly: %v", r)
				}
				return
			}

			got, err := tt.clouds.GetActiveCloudIndex()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Got an error when we didn't want one (wantErr:%v): %v", tt.wantErr, err)
				}
				return
			}

			if tt.wantErr {
				t.Errorf("Wanted an error got none. wantErr: %v, err: %v", tt.wantErr, err)
			}

			if *got != tt.want {
				t.Errorf("GetActiveCloudIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClouds_NumberOfActiveCloudConfigs(t *testing.T) {
	tests := []struct {
		name   string
		clouds config.Clouds
		want   int
	}{
		{
			name:   "empty clouds",
			clouds: config.Clouds{},
			want:   0,
		},
		{
			name: "no active clouds",
			clouds: config.Clouds{
				{Active: false},
				{Active: false},
			},
			want: 0,
		},
		{
			name: "some active clouds",
			clouds: config.Clouds{
				{Active: true},
				{Active: false},
				{Active: true},
			},
			want: 2,
		},
		{
			name: "all active clouds",
			clouds: config.Clouds{
				{Active: true},
				{Active: true},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.clouds.NumberOfActiveCloudConfigs(); got != tt.want {
				t.Errorf("NumberOfActiveCloudConfigs() = %v, want %v", got, tt.want)
			}
		})
	}
}
