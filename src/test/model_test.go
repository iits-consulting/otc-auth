package test

import (
	"otc-auth/src/config"
	"reflect"
	"strings"
	"testing"
)

func TestCloudsSlice_RemoveCloudByNameIfExists(t *testing.T) {
	testCases := []struct {
		desc     string
		actual   config.Clouds
		expected config.Clouds
		input    string
	}{
		{
			desc: "cloud to be removed exists",
			actual: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}},
			},
			expected: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}},
			},
			input: "cloud-2",
		},
		{
			desc: "cloud to be removed does not exist",
			actual: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}},
			},
			expected: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}},
			},
			input: "cloud-4",
		},
		{
			desc:     "clouds is empty",
			actual:   config.Clouds{},
			expected: config.Clouds{},
			input:    "cloud-1",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.actual.RemoveCloudByNameIfExists(tc.input)

			var actualCloudNames []string
			for _, cloud := range tc.actual {
				actualCloudNames = append(actualCloudNames, cloud.Domain.Name)
			}
			var expectedCloudNames []string
			for _, cloud := range tc.actual {
				expectedCloudNames = append(expectedCloudNames, cloud.Domain.Name)
			}

			if !reflect.DeepEqual(actualCloudNames, expectedCloudNames) {
				t.Errorf("(%s): actual %s, expected %s", tc.desc, strings.Join(actualCloudNames, ", "), strings.Join(expectedCloudNames, ", "))
			}
		})
	}
}

func TestCloudsSlice_SetActiveByName(t *testing.T) {
	testCases := []struct {
		desc     string
		expected config.Clouds
		actual   config.Clouds
		input    string
	}{
		{
			desc: "set active makes all others inactive",
			expected: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}, Active: false},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}, Active: true},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}, Active: false},
			},
			actual: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}, Active: true},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}, Active: true},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}, Active: true},
			},
			input: "cloud-2",
		},
		{
			desc: "set active with unknown name sets all inactive",
			expected: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}, Active: false},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}, Active: false},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}, Active: false},
			},
			actual: config.Clouds{
				{Domain: config.NameAndIdResource{Name: "cloud-1"}, Active: true},
				{Domain: config.NameAndIdResource{Name: "cloud-2"}, Active: true},
				{Domain: config.NameAndIdResource{Name: "cloud-3"}, Active: true},
			},
			input: "cloud-4",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.actual.SetActiveByName(tc.input)
			if tc.actual.NumberOfActiveCloudConfigs() != tc.expected.NumberOfActiveCloudConfigs() {
				t.Errorf("(%s): expected %d, actual %d", tc.desc, tc.expected.NumberOfActiveCloudConfigs(), tc.actual.NumberOfActiveCloudConfigs())
			}
		})
	}
}

func TestTokens_UpdateToken(t *testing.T) {
	testCases := []struct {
		desc     string
		expected config.Tokens
		actual   config.Tokens
		input    config.Token
	}{
		{
			desc: "update scoped token successfully",
			expected: config.Tokens{
				{
					Type:      config.Unscoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
				{
					Type:      config.Scoped,
					Secret:    "updated-secret",
					IssuedAt:  "updated-date",
					ExpiresAt: "updated-date",
				},
			},
			actual: config.Tokens{
				{
					Type:      config.Unscoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
				{
					Type:      config.Scoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
			},
			input: config.Token{
				Type:      config.Scoped,
				Secret:    "updated-secret",
				IssuedAt:  "updated-date",
				ExpiresAt: "updated-date",
			},
		},
		{
			desc: "scoped token created successfully",
			expected: config.Tokens{
				{
					Type:      config.Scoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
			},
			actual: config.Tokens{},
			input: config.Token{
				Type:      config.Scoped,
				Secret:    "secret",
				IssuedAt:  "date",
				ExpiresAt: "date",
			},
		},
		{
			desc: "update unscoped token successfully",
			expected: config.Tokens{
				{
					Type:      config.Unscoped,
					Secret:    "updated-secret",
					IssuedAt:  "updated-date",
					ExpiresAt: "updated-date",
				},
				{
					Type:      config.Scoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
			},
			actual: config.Tokens{
				{
					Type:      config.Unscoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
				{
					Type:      config.Scoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
			},
			input: config.Token{
				Type:      config.Unscoped,
				Secret:    "updated-secret",
				IssuedAt:  "updated-date",
				ExpiresAt: "updated-date",
			},
		},
		{
			desc: "unscoped token created successfully",
			expected: config.Tokens{
				{
					Type:      config.Unscoped,
					Secret:    "secret",
					IssuedAt:  "date",
					ExpiresAt: "date",
				},
			},
			actual: config.Tokens{},
			input: config.Token{
				Type:      config.Unscoped,
				Secret:    "secret",
				IssuedAt:  "date",
				ExpiresAt: "date",
			},
		},
		{
			desc:     "token neither created nor updated, ok false",
			expected: config.Tokens{},
			actual:   config.Tokens{},
			input: config.Token{
				Type:      "invalid type",
				Secret:    "secret",
				IssuedAt:  "date",
				ExpiresAt: "date",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ok := tc.actual.UpdateToken(tc.input)
			if !reflect.DeepEqual(tc.actual, tc.expected) {
				t.Errorf("(%s): expected %s, actual %s", tc.desc, tc.expected, tc.actual)
			}
			if tc.desc == "token neither created nor updated, ok false" {
				if ok {
					t.Errorf("(%s): expected %t, actual %t", tc.desc, false, ok)
				}
			}
		})
	}
}
