package test_test

import (
	"testing"

	"otc-auth/config"
)

const firstDomain = "firstDomain"

func TestLoadCloudConfig_init(t *testing.T) {
	config.LoadCloudConfig(firstDomain)

	result := config.GetActiveCloudConfig().Domain
	if result.Name != firstDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", firstDomain, result.Name)
	}
}

func TestLoadCloudConfig_two_domains(t *testing.T) {
	secondDomain := "second"

	config.LoadCloudConfig(firstDomain)
	config.LoadCloudConfig(secondDomain)

	result := config.GetActiveCloudConfig().Domain
	if result.Name != secondDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", secondDomain, result.Name)
	}
}

func TestLoadCloudConfig_make_domain_twice_active(t *testing.T) {
	config.LoadCloudConfig(firstDomain)
	config.LoadCloudConfig(firstDomain)

	result := config.GetActiveCloudConfig().Domain
	if result.Name != firstDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", firstDomain, result.Name)
	}
}
