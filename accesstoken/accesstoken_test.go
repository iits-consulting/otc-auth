//nolint:testpackage // whitebox testing
package accesstoken

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"
)

// TestCredentialsList_UpstreamRegression is a canary for the documented
// gophertelekomcloud regression at credentials.List (see accesstoken.go).
// It calls the SDK directly with a stub client; the URL builder fails via
// reflection BEFORE any HTTP, so no network is needed.
//
// If this test starts failing because result.Err is nil OR the error message
// changed, upstream likely fixed the bug. Action: remove listCredentials in
// accesstoken.go, switch ListAccessToken back to credentials.List, and drop
// this canary.
func TestCredentialsList_UpstreamRegression(t *testing.T) {
	t.Parallel()

	client := &golangsdk.ServiceClient{
		ProviderClient: &golangsdk.ProviderClient{},
		Endpoint:       "http://example.invalid/",
	}

	result := credentials.List(client, credentials.ListOpts{UserID: "any-user-id"})

	if result.Err == nil {
		t.Fatal("upstream regression appears to be FIXED: credentials.List no longer errors " +
			"before the request. Drop listCredentials workaround in accesstoken.go and this canary.")
	}

	const wantSubstr = "options type is not a struct"
	if !strings.Contains(result.Err.Error(), wantSubstr) {
		t.Errorf("upstream error message changed; got %q, expected substring %q. "+
			"Re-check the documented behavior and update accesstoken.go accordingly.",
			result.Err.Error(), wantSubstr)
	}
}

// TestListCredentials_HitsV30Endpoint exercises the local workaround: it must
// issue GET against /v3.0/OS-CREDENTIAL/credentials with the user_id query and
// parse the {"credentials":[...]} envelope. Guards against accidental reverts
// to the broken SDK path.
func TestListCredentials_HitsV30Endpoint(t *testing.T) {
	t.Parallel()

	var gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"credentials":[
			{"user_id":"u1","access":"AK1","description":"Token by otc-auth","status":"active"},
			{"user_id":"u1","access":"AK2","description":"other","status":"active"}
		]}`))
	}))
	t.Cleanup(server.Close)

	client := &golangsdk.ServiceClient{
		ProviderClient: &golangsdk.ProviderClient{},
		Endpoint:       server.URL + "/v3/",
	}

	got, err := listCredentials(client, "u1")
	if err != nil {
		t.Fatalf("listCredentials returned error: %v", err)
	}

	if want := "/v3.0/OS-CREDENTIAL/credentials"; gotPath != want {
		t.Errorf("path = %q, want %q (must hit v3.0 endpoint)", gotPath, want)
	}
	if want := "user_id=u1"; gotQuery != want {
		t.Errorf("query = %q, want %q", gotQuery, want)
	}
	if len(got) != 2 {
		t.Fatalf("got %d credentials, want 2", len(got))
	}
	if got[0].AccessKey != "AK1" || got[0].Description != "Token by otc-auth" {
		t.Errorf("credential[0] = %+v, want AK1/'Token by otc-auth'", got[0])
	}
}
