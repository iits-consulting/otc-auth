package oidc

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"otc-auth/common"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-http-utils/headers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

//nolint:gochecknoglobals // Works for now but needs a rewrite
var (
	backgroundCtx = context.Background()

	oAuth2Config    oauth2.Config
	state           string
	idTokenVerifier *oidc.IDTokenVerifier
)

const (
	localhost   = "localhost:8088"
	redirectURL = "http://localhost:8088/oidc/auth"

	queryState             = "state"
	queryCode              = "code"
	idTokenField           = "id_token"
	normalMaxIDTokenLength = 2300

	rwTimeout   = 1 * time.Minute
	idleTimeout = 2 * time.Minute
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	rawAccessToken := r.Header.Get(headers.Authorization)
	if rawAccessToken == "" {
		http.Redirect(w, r, oAuth2Config.AuthCodeURL(state), http.StatusFound)
		return
	}
	parts := strings.Split(rawAccessToken, " ")
	if len(parts) != 2 { //nolint:mnd // Bearer tokens need to be of the format "Bearer ey..."
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := idTokenVerifier.Verify(backgroundCtx, parts[1])
	if err != nil {
		http.Redirect(w, r, oAuth2Config.AuthCodeURL(state), http.StatusFound)
		return
	}
}

func startAndListenHTTPServer(channel chan common.OidcCredentialsResponse) {
	http.HandleFunc("/", handleRoot)

	http.HandleFunc("/oidc/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get(queryState) != state {
			http.Error(w, "state does not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := oAuth2Config.Exchange(backgroundCtx, r.URL.Query().Get(queryCode))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to exchange token: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		idToken, ok := oauth2Token.Extra(idTokenField).(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
			return
		}
		if len(idToken) > normalMaxIDTokenLength {
			glog.Warningf("warning: id token longer than %d characters"+
				" - consider removing some groups or roles", normalMaxIDTokenLength)
		}
		rawIDToken, err := idTokenVerifier.Verify(backgroundCtx, idToken)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to verify ID Token: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		oidcUsernameAndToken := common.OidcCredentialsResponse{}
		err = rawIDToken.Claims(&oidcUsernameAndToken.Claims)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(common.SuccessPageHTML))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if idToken != "" {
			oidcUsernameAndToken.BearerToken = fmt.Sprintf("Bearer %s", idToken)
			channel <- oidcUsernameAndToken
		}
	})

	lc := net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "tcp", localhost)
	if err != nil {
		common.ThrowError(
			errors.Wrap(err,
				fmt.Sprintf("can't listen on %s, something might already be using this port", localhost)))
	}

	server := &http.Server{
		Handler:      nil,
		ReadTimeout:  rwTimeout,
		WriteTimeout: rwTimeout,
		IdleTimeout:  idleTimeout,
	}

	err = server.Serve(listener)
	if err != nil {
		common.ThrowError(
			fmt.Errorf("failed to start server at %s: %w", localhost, err))
	}
}

func authenticateWithIdp(params common.AuthInfo) common.OidcCredentialsResponse {
	channel := make(chan common.OidcCredentialsResponse)
	go startAndListenHTTPServer(channel)
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, params.IdpURL)
	if err != nil {
		common.ThrowError(err)
	}

	oAuth2Config = oauth2.Config{
		ClientID:     params.ClientID,
		ClientSecret: params.ClientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       params.OidcScopes,
	}

	idTokenVerifier = provider.Verifier(&oidc.Config{ClientID: params.ClientID})
	state = uuid.New().String()

	err = browser.OpenURL(fmt.Sprintf("http://%s", localhost))
	if err != nil {
		common.ThrowError(err)
	}

	return <-channel
}
