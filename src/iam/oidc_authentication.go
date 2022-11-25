package iam

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"html/template"
	"net/http"
	"os"
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

		projectDir, err := os.Getwd()
		if err != nil {
			util.OutputErrorToConsoleAndExit(err)
		}
		if !strings.HasSuffix(projectDir, "/src") {
			projectDir += "/src"
		}
		page, err := template.ParseFiles(fmt.Sprintf("%s/static/authorized.html", projectDir))
		if err != nil {
			util.OutputErrorToConsoleAndExit(err)
		}

		if err := page.ExecuteTemplate(w, "authorized.html", nil); err != nil {
			util.OutputErrorToConsoleAndExit(err)
		}

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
