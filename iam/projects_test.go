//nolint:testpackage // whitebox testing
package iam

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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

	expiredToken := config.Token{Secret: "stale", ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339)}
	freshToken := config.Token{Secret: "fresh", ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339)}
	freshGopherToken, convErr := configTokenToGopherToken(&freshToken)
	if convErr != nil {
		t.Fatalf("couldn't convert config token to gopher token: %v", convErr)
	}
	cloudWith := func(names ...string) *config.Cloud {
		var ps config.Projects
		for i, n := range names {
			ps = append(ps, config.Project{
				NameAndIDResource: config.NameAndIDResource{Name: n, ID: fmt.Sprintf("id%d", i)},
				ScopedToken:       expiredToken,
			})
		}
		return &config.Cloud{Projects: ps, Region: "eu-de"}
	}

	tests := []struct {
		name          string
		projectNames  []string
		store         *mockConfigStore
		wantErr       bool
		wantGetCalls  int
		wantSaveCalls int
	}{
		{
			name:          "all projects refreshed and saved once",
			projectNames:  []string{"p1", "p2"},
			store:         &mockConfigStore{cloudToReturn: cloudWith("p1", "p2")},
			wantGetCalls:  2,
			wantSaveCalls: 1,
		},
		{
			name:         "empty project list does not save",
			projectNames: nil,
			store:        &mockConfigStore{cloudToReturn: cloudWith()},
			wantGetCalls: 1,
		},
		{
			name:         "stops at first failure without saving",
			projectNames: []string{"missing", "p1"},
			store:        &mockConfigStore{cloudToReturn: cloudWith("p1")},
			wantErr:      true,
			wantGetCalls: 1,
		},
		{
			name:         "config read fails",
			projectNames: []string{"p1"},
			store:        &mockConfigStore{getError: errors.New("disk error")},
			wantErr:      true,
			wantGetCalls: 1,
		},
		{
			name:          "config save fails",
			projectNames:  []string{"p1"},
			store:         &mockConfigStore{cloudToReturn: cloudWith("p1"), saveError: errors.New("disk full")},
			wantErr:       true,
			wantGetCalls:  2,
			wantSaveCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tc := &mockTokenCreator{tokenToReturn: freshGopherToken}
			err := createScopedTokenForEveryProject(tt.store, tc, tt.projectNames)
			if (err != nil) != tt.wantErr {
				t.Fatalf("createScopedTokenForEveryProject() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.store.GetCallCount != tt.wantGetCalls {
				t.Errorf("GetActiveCloud calls = %d, want %d", tt.store.GetCallCount, tt.wantGetCalls)
			}
			if tt.store.SaveCallCount != tt.wantSaveCalls {
				t.Errorf("SaveActiveCloud calls = %d, want %d", tt.store.SaveCallCount, tt.wantSaveCalls)
			}
			if tt.store.SavedCloudState != nil {
				assertAllScopedTokens(t, tt.store.SavedCloudState, freshToken)
			}
		})
	}
}

func assertAllScopedTokens(t *testing.T, cloud *config.Cloud, want config.Token) {
	t.Helper()
	for _, project := range cloud.Projects {
		if project.ScopedToken != want {
			t.Errorf("saved token for %s = %v, want %v", project.Name, project.ScopedToken, want)
		}
	}
}

func Test_projectsFromResponse(t *testing.T) {
	t.Parallel()

	var resp common.ProjectsResponse
	if err := json.Unmarshal([]byte(fakeProjectsBody), &resp); err != nil {
		t.Fatalf("seed payload unmarshal: %v", err)
	}

	got := projectsFromResponse(resp)

	want := config.Projects{
		{NameAndIDResource: config.NameAndIDResource{Name: "eu-de", ID: "p1"}},
		{NameAndIDResource: config.NameAndIDResource{Name: "eu-de_MyProject", ID: "p2"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("projectsFromResponse() = %v, want %v", got, want)
	}
	if projectsFromResponse(common.ProjectsResponse{}) != nil {
		t.Error("empty response should yield nil projects")
	}
}

func TestFetchProjects_DoesNotPersist(t *testing.T) {
	useFakeIAM(t, http.StatusCreated, fakeTokenBody, fakeCloud())

	got := FetchProjects()

	want := []string{"eu-de", "eu-de_MyProject"}
	if !reflect.DeepEqual(got.GetProjectNames(), want) {
		t.Errorf("returned projects = %v, want %v", got.GetProjectNames(), want)
	}
	persisted, err := config.GetActiveCloudConfig()
	if err != nil {
		t.Fatalf("GetActiveCloudConfig() error = %v", err)
	}
	if len(persisted.Projects) != 0 {
		t.Errorf("FetchProjects persisted %d projects, want none", len(persisted.Projects))
	}
}

func TestCreateScopedTokenForEveryProject_PersistsTokens(t *testing.T) {
	useFakeIAM(t, http.StatusCreated, fakeTokenBody, fakeCloud("eu-de", "eu-nl"))

	if err := CreateScopedTokenForEveryProject([]string{"eu-de", "eu-nl"}); err != nil {
		t.Fatalf("CreateScopedTokenForEveryProject() error = %v", err)
	}

	persisted, err := config.GetActiveCloudConfig()
	if err != nil {
		t.Fatalf("GetActiveCloudConfig() error = %v", err)
	}
	for _, project := range persisted.Projects {
		if project.ScopedToken.Secret != fakeTokenID {
			t.Errorf("project %s scoped token = %q, want %q", project.Name, project.ScopedToken.Secret, fakeTokenID)
		}
	}
}
