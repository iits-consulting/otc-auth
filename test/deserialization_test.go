package test

import (
	"otc-auth/common"
	"otc-auth/config"
	"reflect"
	"testing"
)

func TestDeserializeJsonForType__TokenResponse(t *testing.T) {
	desc := "token response gets deserialized"
	expected := getExpectedTokenResponse()
	var actual *common.TokenResponse
	input := []byte(`{"token":{"expires_at":"2022-11-30T14:01:54.956000Z","methods":["password"],"catalog":[{"endpoints":[{"id":"endpoint-id","interface":"endpoint-interface","region":"endpoint-region","region_id":"endpoint-region-id","url":"endpoint-url"}],"id":"catalog-id","name":"catalog-name","type":"catalog-type"}],"domain":{"id":"domain-id","name":"domain-name","xdomain_id":"x-domain-id","xdomain_type":"x-domain-type"},"roles":[{"id":"role-id","name":"role-name"}],"issued_at":"2022-11-29T14:01:54.956000Z","user":{"domain":{"id":"domain-id","name":"domain-name","xdomain_id":"x-domain-id","xdomain_type":"x-domain-type"},"id":"user-id","name":"user-name","password_expires_at":"2023-02-26T13:59:21.000000Z"}}}`)

	actual = common.DeserializeJsonForType[common.TokenResponse](input)
	if !reflect.DeepEqual(expected, *actual) {
		t.Errorf("(%s): expected %s, actual %s", desc, expected, *actual)
	}
}

func TestDeserializeJsonForType__Cluster(t *testing.T) {
	var actual *config.Cluster
	desc := "cluster gets deserialized"
	expected := config.Cluster{Name: "cluster-name", Id: "cluster-id"}
	input := []byte(`{"name":"cluster-name","id":"cluster-id"}`)

	actual = common.DeserializeJsonForType[config.Cluster](input)

	if !reflect.DeepEqual(expected, *actual) {
		t.Errorf("(%s): expected %s, actual %s", desc, expected, *actual)
	}
}

func getExpectedTokenResponse() common.TokenResponse {
	expected := common.TokenResponse{}
	expected.Token.ExpiresAt = "2022-11-30T14:01:54.956000Z"
	expected.Token.IssuedAt = "2022-11-29T14:01:54.956000Z"
	expected.Token.User.Domain.Id = "domain-id"
	expected.Token.User.Domain.Name = "domain-name"
	expected.Token.User.Name = "user-name"
	return expected
}
