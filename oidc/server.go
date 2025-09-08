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

type IVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error)
}

type authFlow struct {
	oAuth2Config    oauth2.Config
	idTokenVerifier IVerifier
	state           string
}

func (a *authFlow) handleRoot(w http.ResponseWriter, r *http.Request) {
	rawAccessToken := r.Header.Get(headers.Authorization)
	if rawAccessToken == "" {
		http.Redirect(w, r, a.oAuth2Config.AuthCodeURL(a.state), http.StatusFound)
		return
	}
	parts := strings.Split(rawAccessToken, " ")
	if len(parts) != 2 { //nolint:mnd // Bearer tokens need to be of the format "Bearer ey..."
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := a.idTokenVerifier.Verify(backgroundCtx, parts[1])
	if err != nil {
		http.Redirect(w, r, a.oAuth2Config.AuthCodeURL(a.state), http.StatusFound)
		return
	}
}

type listenerFactory func(address string) (net.Listener, error)

func createListener(address string) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("can't listen on %s, something might already be using this port", address))
	}
	return listener, nil
}

func startAndListenHTTPServer(channel chan common.OidcCredentialsResponse, a *authFlow, createListener listenerFactory) error {
	registerHandlers(channel, a)

	listener, err := createListener(localhost)
	if err != nil {
		return errors.Wrap(err, "couldn't start http server")
	}

	server := newHTTPServer(rwTimeout, rwTimeout, idleTimeout)
	return server.Serve(listener)
}

func registerHandlers(channel chan common.OidcCredentialsResponse, a *authFlow) {
	http.HandleFunc("/", a.handleRoot)
	http.HandleFunc("/oidc/auth", func(w http.ResponseWriter, r *http.Request) {
		handleOIDCAuth(w, r, channel, a)
	})
}

func handleOIDCAuth(w http.ResponseWriter, r *http.Request, channel chan common.OidcCredentialsResponse, a *authFlow) {
	if r.URL.Query().Get(queryState) != a.state {
		http.Error(w, "state does not match", http.StatusBadRequest)
		return
	}

	oauth2Token, err := a.oAuth2Config.Exchange(backgroundCtx, r.URL.Query().Get(queryCode))
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	idToken, ok := oauth2Token.Extra(idTokenField).(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token", http.StatusInternalServerError)
		return
	}

	if len(idToken) > normalMaxIDTokenLength {
		glog.Warningf(
			"warning: id token longer than %d characters – consider removing some groups or roles",
			normalMaxIDTokenLength,
		)
	}

	rawIDToken, err := a.idTokenVerifier.Verify(backgroundCtx, idToken)
	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var creds common.OidcCredentialsResponse
	if err = rawIDToken.Claims(&creds.Claims); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err = w.Write([]byte(common.SuccessPageHTML)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if idToken != "" {
		creds.BearerToken = fmt.Sprintf("Bearer %s", idToken)
		channel <- creds
	}
}

func newHTTPServer(readTimeout, writeTimeout, idleTimeout time.Duration) *http.Server {
	return &http.Server{
		Handler:      nil,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}
func authenticateWithIdp(params common.AuthInfo) (*common.OidcCredentialsResponse, error) {
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, params.IdpURL)
	if err != nil {
		return nil, err
	}
	a := authFlow{
		oAuth2Config: oauth2.Config{
			ClientID:     params.ClientID,
			ClientSecret: params.ClientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       params.OidcScopes,
		},
		idTokenVerifier: provider.Verifier(&oidc.Config{ClientID: params.ClientID}),
		state:           uuid.New().String(),
	}
	channel := make(chan common.OidcCredentialsResponse)
	go func() error {
		return startAndListenHTTPServer(channel, &a, createListener)
	}()

	err = browser.OpenURL(fmt.Sprintf("http://%s", localhost))
	if err != nil {
		return nil, err
	}

	resp := <-channel
	return &resp, nil
}
