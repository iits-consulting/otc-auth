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
			if err := writeProjectNames(&buf, tc.projects); err != nil {
				t.Fatalf("writeProjectNames returned error: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestGetProjectsAndPrint_WritesNamesToWriter would have caught the original
// silent-output regression on `projects list` (issue #177): the previous
// implementation sent the project list to glog instead of stdout. Asserting
// the buffer is non-empty via the io.Writer seam catches that class directly.
func TestGetProjectsAndPrint_WritesNamesToWriter(t *testing.T) {
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

	var buf bytes.Buffer
	got := getProjectsAndPrint(&buf, fakeFetch, fakeUpdate)

	if buf.Len() == 0 {
		t.Fatal("no output written — regression to glog-only behavior?")
	}
	out := buf.String()
	for _, want := range []string{"eu-de", "eu-de_MyProject"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got %q", want, out)
		}
	}
	if len(updated) != 2 {
		t.Errorf("updater received %d projects, want 2", len(updated))
	}
	if len(got) != 2 {
		t.Errorf("returned %d projects, want 2", len(got))
	}
}
