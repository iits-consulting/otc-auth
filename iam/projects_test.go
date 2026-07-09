//nolint:testpackage // whitebox testing
package iam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"otc-auth/common"
	"otc-auth/config"
)

func TestWriteProjectNames(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		projects config.Projects
		want     string
	}{
		{
			name:     "empty list prints blank line",
			projects: config.Projects{},
			want:     "\n",
		},
		{
			name: "single project on one line",
			projects: config.Projects{
				{NameAndIDResource: config.NameAndIDResource{Name: "eu-de", ID: "1"}},
			},
			want: "eu-de\n",
		},
		{
			name: "multiple projects newline separated",
			projects: config.Projects{
				{NameAndIDResource: config.NameAndIDResource{Name: "eu-de", ID: "1"}},
				{NameAndIDResource: config.NameAndIDResource{Name: "eu-de_MyProject", ID: "2"}},
				{NameAndIDResource: config.NameAndIDResource{Name: "eu-nl", ID: "3"}},
			},
			want: "eu-de\neu-de_MyProject\neu-nl\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			if err := WriteProjectNames(&buf, tc.projects); err != nil {
				t.Fatalf("WriteProjectNames returned error: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestCreateScopedTokenForEveryProject(t *testing.T) {
	t.Parallel()

	validToken := config.Token{
		Secret:    "s",
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	cloudWith := func(names ...string) *config.Cloud {
		var ps config.Projects
		for i, n := range names {
			ps = append(ps, config.Project{
				NameAndIDResource: config.NameAndIDResource{Name: n, ID: fmt.Sprintf("id%d", i)},
				ScopedToken:       validToken,
			})
		}
		return &config.Cloud{Projects: ps, Region: "eu-de"}
	}

	tests := []struct {
		name         string
		projectNames []string
		cloud        *config.Cloud
		wantErr      bool
		wantGetCalls int
	}{
		{
			name:         "all projects succeed",
			projectNames: []string{"p1", "p2"},
			cloud:        cloudWith("p1", "p2"),
			wantErr:      false,
			wantGetCalls: 2,
		},
		{
			name:         "empty project list is a no-op",
			projectNames: nil,
			cloud:        cloudWith(),
			wantErr:      false,
			wantGetCalls: 0,
		},
		{
			name:         "stops at first failure (fail-fast)",
			projectNames: []string{"missing", "p1"},
			cloud:        cloudWith("p1"),
			wantErr:      true,
			wantGetCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &mockConfigStore{cloudToReturn: tt.cloud}
			err := createScopedTokenForEveryProject(store, &mockTokenCreator{}, tt.projectNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("createScopedTokenForEveryProject() error = %v, wantErr %v", err, tt.wantErr)
			}
			if store.GetCallCount != tt.wantGetCalls {
				t.Errorf("GetActiveCloud calls = %d, want %d", store.GetCallCount, tt.wantGetCalls)
			}
		})
	}
}

func TestGetProjectsInActiveCloud_FetchUpdateOnly(t *testing.T) {
	t.Parallel()

	fakeFetch := func() common.ProjectsResponse {
		var resp common.ProjectsResponse
		const payload = `{"projects":[{"name":"eu-de","id":"p1"},{"name":"eu-de_MyProject","id":"p2"}]}`
		if err := json.Unmarshal([]byte(payload), &resp); err != nil {
			t.Fatalf("seed payload unmarshal: %v", err)
		}
		return resp
	}
	var updated config.Projects
	fakeUpdate := func(p config.Projects) { updated = p }

	got := getProjectsInActiveCloud(fakeFetch, fakeUpdate)

	if len(updated) != 2 {
		t.Errorf("updater received %d projects, want 2", len(updated))
	}
	if len(got) != 2 {
		t.Fatalf("returned %d projects, want 2", len(got))
	}
	for _, want := range []string{"eu-de", "eu-de_MyProject"} {
		found := false
		for _, p := range got {
			if strings.Contains(p.Name, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing project %q", want)
		}
	}
}
