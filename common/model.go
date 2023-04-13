package common

import (
	"encoding/xml"
)

type AuthInfo struct {
	AuthType         string
	IdpName          string
	IdpUrl           string
	Username         string
	Password         string
	AuthProtocol     string
	DomainName       string
	Otp              string
	UserDomainId     string
	ClientId         string
	ClientSecret     string
	OverwriteFile    bool
	IsServiceAccount bool
	OidcScopes       []string
}
type SamlAssertionResponse struct {
	Name   xml.Name
	Header struct {
		Response struct {
			AssertionConsumerServiceURL string `xml:"AssertionConsumerServiceURL,attr"`
		} `xml:"Response"`
	} `xml:"Header"`
}

type OidcCredentialsResponse struct {
	BearerToken string
	Claims      struct {
		PreferredUsername string `json:"preferred_username"`
	}
}

type TokenResponse struct {
	Token struct {
		Secret    string
		ExpiresAt string `json:"expires_at"`
		IssuedAt  string `json:"issued_at"`
		User      struct {
			Domain struct {
				Id   string `json:"id"`
				Name string `json:"name"`
			} `json:"domain"`
			Name string `json:"name"`
		} `json:"user"`
	} `json:"token"`
}

type ProjectsResponse struct {
	Projects []struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	} `json:"projects"`
}

type ClustersResponse struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
			UID  string `json:"uid"`
		} `json:"metadata"`
	} `json:"items"`
}

const SuccessPageHtml = `
<!DOCTYPE html>
<html lang="en">
<head>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65" crossorigin="anonymous">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.10.2/font/bootstrap-icons.css">
    <meta name="viewport" content="width=device-width, initial-scale=1" charset="UTF-8">
    <title>Success</title>
</head>
<body style="height: 100%">
<div class="d-flex flex-column min-vh-100 justify-content-center align-items-center">
    <div class="col"></div>
    <div class="col-4">
        <h1 class="text-center">Success!</h1><br/>
        <div class="text-center" style="background-color: rgba(148, 240, 169, 0.2); padding: 1.25rem 1.25rem .25rem;border: 0.075rem solid #94F0A9;">
            <i class="bi bi-check-circle-fill text-success"></i> <strong class="text-success">Signed in via your OIDC
            provider</strong>
            <p style="margin-top: .75rem">You can now close this window.</p>
        </div>
        <div class="text-center">
            <img src="https://github.com/iits-consulting/otc-auth/blob/main/src/static/images/iits-logo-2021-red-square-xl.png?raw=true" width="250" style="padding: 2rem"/>
        </div>
    </div>
    <div class="col"></div>
</div>
</body>
<footer style="width:100%; bottom: 0px; position: fixed; border-top: solid .1em; border-top-color: #DDE0E3; background-color: #F4F5F6; padding: 2em;">
    <div class="row text-center">
        <div class="col">
            <p>Built with ❤️ by <a href="https://iits-consulting.de" target="_self">iits consulting</a></p>
        </div>
        <div class="col">
            <p><a href="https://github.com/iits-consulting/otc-auth" target="_self"><i class="bi bi-github"></i>Github</a></p>
        </div>
    </div>
</footer>
</html>
`
