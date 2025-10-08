package test_test

import (
	"reflect"
	"testing"

	"otc-auth/common"
	"otc-auth/config"
)

func TestDeserializeJsonForType__TokenResponse(t *testing.T) {
	desc := "token response gets deserialized"
	expected := getExpectedTokenResponse()
	var actual *common.TokenResponse
	//nolint:lll
	input := []byte(`{"token":{"expires_at":"2022-11-30T14:01:54.956000Z","methods":["password"],"catalog":[{"endpoints":[{"id":"endpoint-id","interface":"endpoint-interface","region":"endpoint-region","region_id":"endpoint-region-id","url":"endpoint-url"}],"id":"catalog-id","name":"catalog-name","type":"catalog-type"}],"domain":{"id":"domain-id","name":"domain-name","xdomain_id":"x-domain-id","xdomain_type":"x-domain-type"},"roles":[{"id":"role-id","name":"role-name"}],"issued_at":"2022-11-29T14:01:54.956000Z","user":{"domain":{"id":"domain-id","name":"domain-name","xdomain_id":"x-domain-id","xdomain_type":"x-domain-type"},"id":"user-id","name":"user-name","password_expires_at":"2023-02-26T13:59:21.000000Z"}}}`)

	actual, err := common.DeserializeJSONForType[common.TokenResponse](input)
	if err != nil {
		t.Fatalf("couldn't deserialize json: %v", err)
	}
	if !reflect.DeepEqual(expected, *actual) {
		t.Errorf("(%s): expected %s, actual %s", desc, expected, *actual)
	}
}

func TestDeserializeJsonForType__Cluster(t *testing.T) {
	var actual *config.Cluster
	desc := "cluster gets deserialized"
	expected := config.Cluster{Name: "cluster-name", ID: "cluster-id"}
	input := []byte(`{"name":"cluster-name","id":"cluster-id"}`)

	actual, err := common.DeserializeJSONForType[config.Cluster](input)
	if err != nil {
		t.Fatalf("couldn't deserialize json: %v", err)
	}

	if !reflect.DeepEqual(expected, *actual) {
		t.Errorf("(%s): expected %s, actual %s", desc, expected, *actual)
	}
}

func getExpectedTokenResponse() common.TokenResponse {
	expected := common.TokenResponse{}
	expected.Token.ExpiresAt = "2022-11-30T14:01:54.956000Z"
	expected.Token.IssuedAt = "2022-11-29T14:01:54.956000Z"
	expected.Token.User.Domain.ID = "domain-id"
	expected.Token.User.Domain.Name = "domain-name"
	expected.Token.User.Name = "user-name"
	return expected
}
