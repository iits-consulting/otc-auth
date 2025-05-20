//nolint:testpackage // We use whitebox tests here to validate internal logic like createOpenstackCloudConfig
package openstack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"otc-auth/config"
)

func TestWriteOpenStackCloudsYaml(t *testing.T) {
	tests := []struct {
		name             string
		config           config.OtcConfigContent
		outputFile       string
		expectFileExists bool
	}{
		{
			name: "Writes valid clouds.yaml",
			config: config.OtcConfigContent{
				Clouds: config.Clouds{
					{
						Domain:   config.NameAndIDResource{Name: "demo"},
						Region:   "eu-de",
						Active:   true,
						Username: "user",
						Projects: config.Projects{
							{
								NameAndIDResource: config.NameAndIDResource{Name: "projectA"},
								ScopedToken:       config.Token{Secret: "token123"},
							},
						},
					},
				},
			},
			outputFile:       filepath.Join(os.TempDir(), "test-clouds.yaml"),
			expectFileExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := json.Marshal(tt.config)
			tempdir, err := os.MkdirTemp("", "otc-auth_test")
			if err != nil {
				t.Error(err)
			}
			config.SetCustomConfigFilePath(tempdir)
			_ = os.WriteFile(filepath.Join(tempdir, ".otc-auth-config"), content, 0o644)
			defer os.Remove(filepath.Join(tempdir, ".otc-auth-config"))

			WriteOpenStackCloudsYaml(tt.outputFile)
			defer os.Remove(tt.outputFile)

			if _, err = os.Stat(tt.outputFile); (err == nil) != tt.expectFileExists {
				t.Errorf("expected file existence: %v, got error: %v", tt.expectFileExists, err)
			}
		})
	}
}

func TestCreateOpenstackCloudConfig(t *testing.T) {
	tests := []struct {
		name         string
		project      config.Project
		domain       string
		region       string
		expectedName string
		expectedURL  string
	}{
		{
			name: "Valid project config",
			project: config.Project{
				NameAndIDResource: config.NameAndIDResource{
					Name: "projectA",
				},
				ScopedToken: config.Token{
					Secret: "token123",
				},
			},
			domain:       "testdomain",
			region:       "eu-de",
			expectedName: "testdomain_projectA",
			expectedURL:  "https://iam.eu-de.otc.t-systems.com:443/v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createOpenstackCloudConfig(tt.project, tt.domain, tt.region)

			if result.Cloud != tt.expectedName {
				t.Errorf("unexpected Cloud name: got %q, want %q", result.Cloud, tt.expectedName)
			}

			if result.AuthInfo == nil || result.AuthInfo.Token != tt.project.ScopedToken.Secret {
				t.Errorf("unexpected Auth token: got %v", result.AuthInfo)
			}

			if result.AuthInfo.AuthURL != tt.expectedURL {
				t.Errorf("unexpected AuthURL: got %q, want %q", result.AuthInfo.AuthURL, tt.expectedURL)
			}
		})
	}
}
