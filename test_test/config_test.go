package test_test

import (
	"testing"

	"otc-auth/config"
)

const firstDomain = "firstDomain"

func TestLoadCloudConfig_init(t *testing.T) {
	err := config.LoadCloudConfig(firstDomain)
	if err != nil {
		t.Errorf("could not load cloud config: %v", err)
	}

	result := config.GetActiveCloudConfig().Domain
	if result.Name != firstDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", firstDomain, result.Name)
	}
}

func TestLoadCloudConfig_two_domains(t *testing.T) {
	secondDomain := "second"

	err := config.LoadCloudConfig(firstDomain)
	if err != nil {
		t.Errorf("Error loading first cloud: %s", err)
	}
	err = config.LoadCloudConfig(secondDomain)
	if err != nil {
		t.Errorf("Error loading second cloud: %s", err)
	}

	result := config.GetActiveCloudConfig().Domain
	if result.Name != secondDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", secondDomain, result.Name)
	}
}

func TestLoadCloudConfig_make_domain_twice_active(t *testing.T) {
	err := config.LoadCloudConfig(firstDomain)
	if err != nil {
		t.Errorf("Error loading first cloud: %s", err)
	}
	err = config.LoadCloudConfig(firstDomain)
	if err != nil {
		t.Errorf("Error loading second cloud: %s", err)
	}

	result := config.GetActiveCloudConfig().Domain
	if result.Name != firstDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", firstDomain, result.Name)
	}
}
