package oidc

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-http-utils/headers"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"net/http"
	"otc-auth/src/common"
	"strings"
)

var (
	scopes = []string{oidc.ScopeOpenID, "profile", "roles", "name", "groups", "email"}
	ctx    = context.Background()

	oAuth2Config    oauth2.Config
	state           string
	idTokenVerifier *oidc.IDTokenVerifier
)

const (
	localhost   = "localhost:8088"
	redirectURL = "http://localhost:8088/oidc/auth"

	queryState   = "state"
	queryCode    = "code"
	idTokenField = "id_token"
)

func startAndListenHttpServer(channel chan common.OidcCredentialsResponse) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rawAccessToken := r.Header.Get(headers.Authorization)
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
		if r.URL.Query().Get(queryState) != state {
			http.Error(w, "state does not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := oAuth2Config.Exchange(ctx, r.URL.Query().Get(queryCode))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to exchange token: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		idToken, ok := oauth2Token.Extra(idTokenField).(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}
		rawIdToken, err := idTokenVerifier.Verify(ctx, idToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to verify ID Token: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		oidcUsernameAndToken := common.OidcCredentialsResponse{}
		if err := rawIdToken.Claims(&oidcUsernameAndToken.Claims); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(common.SuccessPageHtml))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if idToken != "" {
			oidcUsernameAndToken.BearerToken = fmt.Sprintf("Bearer %s", idToken)
			channel <- oidcUsernameAndToken
		}
	})

	err := http.ListenAndServe(localhost, nil)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, fmt.Sprintf("failed to start server at %s", localhost))
	}
}

func authenticateWithIdp(params common.AuthInfo) common.OidcCredentialsResponse {
	channel := make(chan common.OidcCredentialsResponse)
	go startAndListenHttpServer(channel)
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, params.IdpUrl)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
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
		common.OutputErrorToConsoleAndExit(err)
	}

	return <-channel
}
