package test

import (
	"otc-auth/config"
	"testing"
)

func TestLoadCloudConfig_init(t *testing.T) {
	domainName := "first"

	config.LoadCloudConfig(domainName)

	result := config.GetActiveCloudConfig().Domain
	if result.Name != domainName {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", domainName, result.Name)
	}

}

func TestLoadCloudConfig_two_domains(t *testing.T) {
	firstDomain := "first"
	secondDomain := "second"

	config.LoadCloudConfig(firstDomain)
	config.LoadCloudConfig(secondDomain)

	result := config.GetActiveCloudConfig().Domain
	if result.Name != secondDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", secondDomain, result.Name)
	}
}

func TestLoadCloudConfig_make_domain_twice_active(t *testing.T) {
	firstDomain := "first"

	config.LoadCloudConfig(firstDomain)
	config.LoadCloudConfig(firstDomain)

	result := config.GetActiveCloudConfig().Domain
	if result.Name != firstDomain {
		t.Errorf("Expected result to contain cloud: %s, but result contains: %s ", firstDomain, result.Name)
	}
}
