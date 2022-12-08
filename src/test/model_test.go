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
