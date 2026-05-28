//nolint:testpackage // whitebox testing
package iam

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

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

// TestGetProjectsInActiveCloud_FetchUpdateOnly guards against two regressions
// at once:
//   - The original silent `projects list` (issue #177): output went to glog
//     instead of stdout. The cmd handler now calls WriteProjectNames explicitly,
//     so the writer-side regression is covered by TestWriteProjectNames above.
//   - The reverse: login flow accidentally printing the project list. The
//     seam takes no writer; this test is a structural witness that fetch+update
//     happens with no I/O. A future refactor that re-adds a stdout write here
//     would have to also break this test's signature.
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
