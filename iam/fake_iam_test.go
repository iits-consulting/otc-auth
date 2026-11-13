//nolint:testpackage // whitebox testing
package iam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"otc-auth/common"
	"otc-auth/config"
)

const (
	fakeTokenID    = "fake-token-id"
	fakeDomainName = "fake-domain"
	fakeTokenBody  = `{"token":{"expires_at":"2099-01-01T00:00:00.000000Z","issued_at":"2023-01-01T00:00:00.000000Z",` +
		`"user":{"id":"user-id","name":"fake-user","domain":{"id":"domain-id","name":"fake-domain"}},"catalog":[]}}`
	fakeProjectsBody = `{"projects":[{"name":"eu-de","id":"p1"},{"name":"eu-de_MyProject","id":"p2"}],"links":{}}`
)

// useFakeIAM points the package at a fake IAM and at a temp config holding
// cloud. Both are package globals, so callers must not use t.Parallel.
func useFakeIAM(t *testing.T, tokenStatus int, tokenBody string, cloud config.Cloud) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		status := tokenStatus
		if status == http.StatusCreated && r.Method == http.MethodGet {
			status = http.StatusOK
		}
		w.Header().Set("X-Subject-Token", fakeTokenID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(tokenBody))
	})
	mux.HandleFunc("/v3/projects", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakeProjectsBody))
	})
	server := httptest.NewServer(mux)

	content, err := json.Marshal(config.OtcConfigContent{Clouds: config.Clouds{cloud}})
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	dir := t.TempDir()
	if err = os.WriteFile(filepath.Join(dir, ".otc-auth-config"), content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	originalBaseURL := iamBaseURL
	iamBaseURL = func(string) string { return server.URL + "/v3" }
	config.SetCustomConfigFilePath(dir)
	t.Cleanup(func() {
		server.Close()
		iamBaseURL = originalBaseURL
		config.SetCustomConfigFilePath("")
	})
}

func fakeCloud(projectNames ...string) config.Cloud {
	cloud := config.Cloud{
		Region: "eu-de",
		Domain: config.NameAndIDResource{Name: fakeDomainName, ID: "domain-id"},
		UnscopedToken: config.Token{
			Secret:    "unscoped",
			ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
		},
		Active: true,
	}
	for _, name := range projectNames {
		cloud.Projects = append(cloud.Projects, config.Project{
			NameAndIDResource: config.NameAndIDResource{Name: name, ID: name + "-id"},
		})
	}
	return cloud
}

func fakeAuthInfo() common.AuthInfo {
	return common.AuthInfo{
		DomainName: fakeDomainName,
		Username:   "fake-user",
		Password:   "fake-password",
		Region:     "eu-de",
	}
}
