package config_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

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
			defer func() {
				if r := recover(); r != nil && !tt.wantErr {
					t.Errorf("GetActiveCloudIndex() panicked unexpectedly: %v", r)
				}
			}()

			got, err := tt.clouds.GetActiveCloudIndex()

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if got == nil {
				t.Error("Unexpected nil result")
			} else if *got != tt.want {
				t.Errorf("GetActiveCloudIndex() = %v, want %v", *got, tt.want)
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

func TestProjects_FindProjectByName(t *testing.T) {
	// Sample projects for testing
	p1 := config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-1"}}
	p2 := config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-2"}}
	p3 := config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-3"}}

	type args struct {
		name string
	}
	tests := []struct {
		name     string
		projects config.Projects
		args     args
		want     *config.Project
	}{
		{
			name:     "found at beginning",
			projects: config.Projects{p1, p2, p3},
			args:     args{name: "project-1"},
			want:     &p1,
		},
		{
			name:     "found in middle",
			projects: config.Projects{p1, p2, p3},
			args:     args{name: "project-2"},
			want:     &p2,
		},
		{
			name:     "found at end",
			projects: config.Projects{p1, p2, p3},
			args:     args{name: "project-3"},
			want:     &p3,
		},
		{
			name:     "not found",
			projects: config.Projects{p1, p2, p3},
			args:     args{name: "non-existent"},
			want:     nil,
		},
		{
			name:     "empty projects",
			projects: config.Projects{},
			args:     args{name: "any"},
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.projects.FindProjectByName(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindProjectByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjects_GetProjectByName(t *testing.T) {
	// Setup test projects
	existingProject := &config.Project{NameAndIDResource: config.NameAndIDResource{Name: "existing"}}
	projects := config.Projects{*existingProject}

	type args struct {
		name string
	}

	tests := []struct {
		name     string
		projects config.Projects
		args     args
		want     *config.Project
		wantErr  bool
	}{
		{
			name:     "project exists",
			projects: projects,
			args:     args{name: "existing"},
			want:     existingProject,
			wantErr:  false,
		},
		{
			name:     "project does not exist",
			projects: projects,
			args:     args{name: "nonexistent"},
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "empty projects list",
			projects: config.Projects{},
			args:     args{name: "any"},
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.projects.GetProjectByName(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProjectByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetProjectByName() got = %v, want %v", got, tt.want)
			}
			// Additional error message check for error cases
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.args.name) {
					t.Errorf("Error message should contain project name, got: %v", err.Error())
				}
				if !strings.Contains(err.Error(), "cce list-projects") {
					t.Errorf("Error message should mention 'cce list-projects', got: %v", err.Error())
				}
			}
		})
	}
}

func TestProjects_FindProjectIndexByName(t *testing.T) {
	// Helper variables for pointer returns
	zero := 0
	one := 1
	two := 2

	type args struct {
		name string
	}
	tests := []struct {
		name     string
		projects config.Projects
		args     args
		want     *int
	}{
		{
			name:     "empty projects",
			projects: config.Projects{},
			args:     args{name: "test"},
			want:     nil,
		},
		{
			name: "project found at index 0",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "test"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "other"}},
			},
			args: args{name: "test"},
			want: &zero,
		},
		{
			name: "project found at middle index",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "first"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "test"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "last"}},
			},
			args: args{name: "test"},
			want: &one,
		},
		{
			name: "project found at last index",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "first"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "second"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "test"}},
			},
			args: args{name: "test"},
			want: &two,
		},
		{
			name: "project not found",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "first"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "second"}},
			},
			args: args{name: "test"},
			want: nil,
		},
		{
			name: "case sensitive matching",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "Test"}},
			},
			args: args{name: "test"},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.projects.FindProjectIndexByName(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindProjectIndexByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjects_GetProjectNames(t *testing.T) {
	tests := []struct {
		name     string
		projects config.Projects
		want     []string
	}{
		{
			name:     "empty projects",
			projects: config.Projects{},
			want:     nil, // Reminder that []string{} != nil
		},
		{
			name: "single project",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-1"}},
			},
			want: []string{"project-1"},
		},
		{
			name: "multiple projects",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-1"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-2"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-3"}},
			},
			want: []string{"project-1", "project-2", "project-3"},
		},
		{
			name: "projects with empty names",
			projects: config.Projects{
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: ""}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: "project-2"}},
				config.Project{NameAndIDResource: config.NameAndIDResource{Name: ""}},
			},
			want: []string{"", "project-2", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.projects.GetProjectNames(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetProjectNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusters_ContainsClusterByName(t *testing.T) {
	tests := []struct {
		name        string
		clusters    config.Clusters
		clusterName string
		want        bool
	}{
		{
			name:        "empty clusters",
			clusters:    config.Clusters{},
			clusterName: "test",
			want:        false,
		},
		{
			name: "cluster exists",
			clusters: config.Clusters{
				{Name: "cluster1"},
				{Name: "cluster2"},
			},
			clusterName: "cluster1",
			want:        true,
		},
		{
			name: "cluster does not exist",
			clusters: config.Clusters{
				{Name: "cluster1"},
				{Name: "cluster2"},
			},
			clusterName: "cluster3",
			want:        false,
		},
		{
			name: "case sensitive comparison",
			clusters: config.Clusters{
				{Name: "Cluster1"},
			},
			clusterName: "cluster1",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.clusters.ContainsClusterByName(tt.clusterName); got != tt.want {
				t.Errorf("ContainsClusterByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToken_IsTokenValid(t *testing.T) {
	now := time.Now()
	type fields struct {
		Secret    string
		IssuedAt  string
		ExpiresAt string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "valid future expiration",
			fields: fields{
				ExpiresAt: now.Add(1 * time.Hour).Format(time.RFC3339),
			},
			want: true,
		},
		{
			name: "expired token",
			fields: fields{
				ExpiresAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			want: false,
		},
		{
			name: "just expired token",
			fields: fields{
				ExpiresAt: now.Add(-1 * time.Second).Format(time.RFC3339),
			},
			want: false,
		},
		{
			name: "empty expiration time",
			fields: fields{
				ExpiresAt: "",
			},
			want: false,
		},
		{
			name: "malformed expiration time",
			fields: fields{
				ExpiresAt: "invalid-time-format",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &config.Token{
				Secret:    tt.fields.Secret,
				IssuedAt:  tt.fields.IssuedAt,
				ExpiresAt: tt.fields.ExpiresAt,
			}
			if got := token.IsTokenValid(); got != tt.want {
				t.Errorf("IsTokenValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToken_UpdateToken(t *testing.T) {
	type fields struct {
		Secret    string
		IssuedAt  string
		ExpiresAt string
	}
	type args struct {
		updatedToken config.Token
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   config.Token
	}{
		{
			name: "update all fields",
			fields: fields{
				Secret:    "old_secret",
				IssuedAt:  "old_issued",
				ExpiresAt: "old_expires",
			},
			args: args{
				updatedToken: config.Token{
					Secret:    "new_secret",
					IssuedAt:  "new_issued",
					ExpiresAt: "new_expires",
				},
			},
			want: config.Token{
				Secret:    "new_secret",
				IssuedAt:  "new_issued",
				ExpiresAt: "new_expires",
			},
		},
		{
			name: "update partial fields",
			fields: fields{
				Secret:    "old_secret",
				IssuedAt:  "old_issued",
				ExpiresAt: "old_expires",
			},
			args: args{
				updatedToken: config.Token{
					Secret: "new_secret",
					// IssuedAt and ExpiresAt left as zero values
				},
			},
			want: config.Token{
				Secret:    "new_secret",
				IssuedAt:  "", // Should be updated to zero value
				ExpiresAt: "", // Should be updated to zero value
			},
		},
		{
			name:   "update empty token",
			fields: fields{}, // Original token is empty
			args: args{
				updatedToken: config.Token{
					Secret:    "new_secret",
					IssuedAt:  "new_issued",
					ExpiresAt: "new_expires",
				},
			},
			want: config.Token{
				Secret:    "new_secret",
				IssuedAt:  "new_issued",
				ExpiresAt: "new_expires",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &config.Token{
				Secret:    tt.fields.Secret,
				IssuedAt:  tt.fields.IssuedAt,
				ExpiresAt: tt.fields.ExpiresAt,
			}
			if got := token.UpdateToken(tt.args.updatedToken); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
