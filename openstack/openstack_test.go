//nolint:testpackage // We use whitebox tests here to validate internal logic like createOpenstackCloudConfig
package openstack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"otc-auth/config"

	"github.com/gophercloud/utils/openstack/clientconfig"
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
			_ = os.WriteFile(filepath.Join(config.GetHomeFolder(), ".otc-auth-config"), content, 0o644)
			defer os.Remove(filepath.Join(config.GetHomeFolder(), ".otc-auth-config"))

			WriteOpenStackCloudsYaml(tt.outputFile)
			defer os.Remove(tt.outputFile)

			if _, err := os.Stat(tt.outputFile); (err == nil) != tt.expectFileExists {
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

func TestCreateOpenstackCloudsYAML(t *testing.T) {
	tests := []struct {
		name        string
		clouds      clientconfig.Clouds
		outputPath  string
		expectError bool
	}{
		{
			name: "Writes YAML with one cloud entry",
			clouds: clientconfig.Clouds{
				Clouds: map[string]clientconfig.Cloud{
					"testcloud": {
						Cloud:   "testcloud",
						Profile: "testcloud",
						AuthInfo: &clientconfig.AuthInfo{
							AuthURL:           "https://iam.eu-de.otc.t-systems.com/v3",
							Token:             "abc123",
							ProjectDomainName: "demo",
						},
						AuthType:           "token",
						Interface:          "public",
						IdentityAPIVersion: "3",
					},
				},
			},
			outputPath:  filepath.Join(os.TempDir(), "clouds-out.yaml"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Remove(tt.outputPath)
			createOpenstackCloudsYAML(tt.clouds, tt.outputPath)

			content, err := os.ReadFile(tt.outputPath)
			if tt.expectError && err == nil {
				t.Errorf("expected error, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error reading file: %v", err)
			}

			if !strings.Contains(string(content), "abc123") {
				t.Errorf("token missing from YAML output: got\n%s", string(content))
			}
		})
	}
}
