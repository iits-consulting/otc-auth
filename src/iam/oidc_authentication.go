package iam

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"net/http"
	"otc-auth/src/util"
	"strings"
)

var (
	scopes = []string{oidc.ScopeOpenID, "profile", "roles", "name", "groups", "email"}
	ctx    = context.Background()

	oAuth2Config    oauth2.Config
	state           string
	idTokenVerifier *oidc.IDTokenVerifier
)

const htmlFile = `
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

const localhost = "localhost:8088"
const redirectURL = "http://localhost:8088/oidc/auth"

func startAndListenHttpServer(channel chan OIDCUsernameAndToken) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rawAccessToken := r.Header.Get("Authorization")
		if rawAccessToken == "" {
			http.Redirect(w, r, oAuth2Config.AuthCodeURL(state), http.StatusFound)
			return
		}

		parts := strings.Split(rawAccessToken, " ")
		if len(parts) != 2 {
			w.WriteHeader(400)
			return
		}

		_, err := idTokenVerifier.Verify(ctx, parts[1])
		if err != nil {
			http.Redirect(w, r, oAuth2Config.AuthCodeURL(state), http.StatusFound)
			return
		}
	})

	http.HandleFunc("/oidc/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state does not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := oAuth2Config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		idToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}
		rawIdToken, err := idTokenVerifier.Verify(ctx, idToken)
		if err != nil {
			http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		oidcUsernameAndToken := OIDCUsernameAndToken{}
		if err := rawIdToken.Claims(&oidcUsernameAndToken.Claims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(htmlFile))

		if idToken != "" {
			oidcUsernameAndToken.BearerToken = fmt.Sprintf("Bearer %s", idToken)
			channel <- oidcUsernameAndToken
		}
	})

	err := http.ListenAndServe(localhost, nil)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err, fmt.Sprintf("failed to start server at %s", localhost))
	}
}

func AuthenticateWithIdp(params LoginParams) OIDCUsernameAndToken {
	channel := make(chan OIDCUsernameAndToken)
	go startAndListenHttpServer(channel)
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, params.IdentityProviderUrl)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	oAuth2Config = oauth2.Config{
		ClientID:     params.ClientId,
		ClientSecret: params.ClientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	idTokenVerifier = provider.Verifier(&oidc.Config{ClientID: params.ClientId})
	state = uuid.New().String()

	err = browser.OpenURL(fmt.Sprintf("http://%s", localhost))
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	return <-channel
}
