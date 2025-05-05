//nolint:testpackage // whitebox testing
package cce

import (
	"reflect"
	"testing"

	"k8s.io/client-go/tools/clientcmd/api"
)

func Test_merge(t *testing.T) {
	t.Parallel()
	knownConfigA := api.Config{
		Clusters:       map[string]*api.Cluster{"cluster-a": {Server: "server-a"}},
		AuthInfos:      map[string]*api.AuthInfo{"user-a": {Token: "token-a"}},
		Contexts:       map[string]*api.Context{"context-a": {Cluster: "cluster-a", AuthInfo: "user-a"}},
		CurrentContext: "context-a",
	}

	knownConfigB := api.Config{
		Clusters:  map[string]*api.Cluster{"cluster-b": {Server: "server-b"}},
		AuthInfos: map[string]*api.AuthInfo{"user-b": {Token: "token-b"}},
		Contexts: map[string]*api.Context{
			"context-b": {Cluster: "cluster-b", AuthInfo: "user-b"},
			"context-a": {Cluster: "cluster-b", AuthInfo: "user-b", Namespace: "ns-b"},
		},
		CurrentContext: "context-b",
	}

	// Expected result of merging B into A (Known C)
	expectedMergedConfigC := api.Config{
		Clusters: map[string]*api.Cluster{
			"cluster-a": {Server: "server-a"}, // From A
			"cluster-b": {Server: "server-b"}, // From B
		},
		AuthInfos: map[string]*api.AuthInfo{
			"user-a": {Token: "token-a"}, // From A
			"user-b": {Token: "token-b"}, // From B
		},
		Contexts: map[string]*api.Context{
			// context-a from B overrides context-a from A because of WithOverride
			"context-a": {Cluster: "cluster-b", AuthInfo: "user-b", Namespace: "ns-b"},
			"context-b": {Cluster: "cluster-b", AuthInfo: "user-b"}, // From B
		},
		CurrentContext: "context-b", // From B (overrides A's)
	}

	tests := []struct {
		name           string
		currentConfig  *api.Config // Input - will be modified by Merge()
		kubeConfig     api.Config  // Input
		expectedConfig api.Config  // Expected state of currentConfig after Merge
		wantErr        bool
	}{
		{
			name:           "Empty config with Known config",
			currentConfig:  &api.Config{},
			kubeConfig:     knownConfigA,
			expectedConfig: knownConfigA,
			wantErr:        false,
		},
		{
			name:           "Empty config with Empty config",
			currentConfig:  &api.Config{},
			kubeConfig:     api.Config{},
			expectedConfig: api.Config{},
			wantErr:        false,
		},
		{
			name: "Known Config A with Known Config B (Override)",
			// IMPORTANT: Create a *copy* of knownConfigA for the input,
			// otherwise the previous test might have modified the shared pointer.
			// Here we create a new literal struct that is identical to knownConfigA.
			currentConfig:  knownConfigA.DeepCopy(),
			kubeConfig:     knownConfigB,
			expectedConfig: expectedMergedConfigC,
			wantErr:        false,
		},
		{
			name: "Nil currentConfig should error",
			// Merge requires a non-nil currentConfig pointer because it merges data *into* the
			// existing object it points to. A nil pointer references no object, making the Merge impossible.
			// mergo.Merge(dst, ...) will return an error if dst is nil.
			currentConfig:  nil,
			kubeConfig:     knownConfigA,
			expectedConfig: api.Config{},
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := merge(tt.currentConfig, tt.kubeConfig)

			if (err != nil) != tt.wantErr {
				t.Errorf("Merge() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(*tt.currentConfig, tt.expectedConfig) {
				t.Errorf("Merge() resulting state mismatch:\ngot = %+v\nwant = %+v", *tt.currentConfig, tt.expectedConfig)
			}
		})
	}
}
