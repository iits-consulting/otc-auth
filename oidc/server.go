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

type iIDToken interface {
	Claims(v interface{}) error
}

type iVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (iIDToken, error)
}

type oidcVerifierWrapper struct {
	realVerifier *oidc.IDTokenVerifier
}

type authFlow struct {
	oAuth2Config    oauth2.Config
	idTokenVerifier iVerifier
	state           string
}

type listenerFactory func(address string, ctx context.Context) (net.Listener, error)

type flowController struct {
	newProvider func(ctx context.Context, issuer string) (*oidc.Provider, error)
	openURL     func(url string) error
	startServer func(ch chan common.OidcCredentialsResponse, a *authFlow, cf listenerFactory, ctx context.Context) error
	newUUID     func() string
}

func newFlowController() *flowController {
	return &flowController{
		newProvider: oidc.NewProvider,
		openURL:     browser.OpenURL,
		startServer: startAndListenHTTPServer,
		newUUID: func() string {
			return uuid.New().String()
		},
	}
}

func (w *oidcVerifierWrapper) Verify(ctx context.Context, rawIDToken string) (iIDToken, error) {
	return w.realVerifier.Verify(ctx, rawIDToken)
}

func (fc *flowController) Authenticate(params common.AuthInfo,
	ctx context.Context,
) (*common.OidcCredentialsResponse, error) {
	provider, err := fc.newProvider(ctx, params.IdpURL)
	if err != nil {
		return nil, err
	}

	realVerifier := provider.Verifier(&oidc.Config{ClientID: params.ClientID})
	a := authFlow{
		oAuth2Config: oauth2.Config{
			ClientID:     params.ClientID,
			ClientSecret: params.ClientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       params.OidcScopes,
		},
		idTokenVerifier: &oidcVerifierWrapper{realVerifier: realVerifier},
		state:           fc.newUUID(),
	}

	respChan := make(chan common.OidcCredentialsResponse)
	errChan := make(chan error, 1) // Buffer of 1 so it doesn't block if an error is sent to chan
	go func() {
		if startErr := fc.startServer(respChan, &a, createListener, ctx); startErr != nil {
			errChan <- startErr
		}
	}()

	err = fc.openURL(fmt.Sprintf("http://%s", localhost))
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-respChan:
		return &resp, nil
	case startErr := <-errChan:
		return nil, startErr
	}
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
	_, err := a.idTokenVerifier.Verify(r.Context(), parts[1])
	if err != nil {
		http.Redirect(w, r, a.oAuth2Config.AuthCodeURL(a.state), http.StatusFound)
		return
	}
}

func createListener(address string, ctx context.Context) (net.Listener, error) {
	listener, err := (*net.ListenConfig).Listen(&net.ListenConfig{}, ctx, "tcp", address)
	if err != nil {
		return nil, errors.Wrap(err,
			fmt.Sprintf("can't listen on %s, something might already be using this port", address))
	}
	return listener, nil
}

func startAndListenHTTPServer(channel chan common.OidcCredentialsResponse,
	a *authFlow, createListener listenerFactory,
	ctx context.Context,
) error {
	registerHandlers(channel, a)

	listener, err := createListener(localhost, ctx)
	if err != nil {
		return errors.Wrap(err, "couldn't start http server")
	}

	server := newHTTPServer(rwTimeout, rwTimeout, idleTimeout, ctx)
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

	oauth2Token, err := a.oAuth2Config.Exchange(r.Context(), r.URL.Query().Get(queryCode))
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

	rawIDToken, err := a.idTokenVerifier.Verify(r.Context(), idToken)
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

func newHTTPServer(readTimeout, writeTimeout, idleTimeout time.Duration, ctx context.Context) *http.Server {
	return &http.Server{
		Handler:      nil,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}
}

func authenticateWithIdp(params common.AuthInfo, ctx context.Context) (*common.OidcCredentialsResponse, error) {
	return newFlowController().Authenticate(params, ctx)
}
