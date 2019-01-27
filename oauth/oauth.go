package oauth

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/nlopes/slack"
)

const slackUserOauthURL = "https://slack.com/oauth/authorize"

var desiredSlackPermissions = []string{
	"client",
}
var slackPermsURLFormatted = strings.Join(desiredSlackPermissions, " ")

// FlowAgent is an agent that will handle generating a link to Slack's user auth endpoint,
// hosting a server to handle the redirect,
// and fetching the access/bearer token using the code received from the redirect
type FlowAgent struct {
	clientID     string
	clientSecret string
	redirectHost string
	redirectPort int
	token        string
	errChan      chan error
	doneChan     chan bool
	tokenChan    chan string
	sync.Mutex
}

// NewFlowAgent ...
func NewFlowAgent(clientID, clientSecret, redirectHost string, redirectPort int) (toReturn *FlowAgent, err error) {
	if clientID == "" || clientSecret == "" || redirectHost == "" || redirectPort == 0 {
		err = fmt.Errorf("invalid config:\n clientID '%s'\n clientSecret '%s'\n redirectHost '%s'\n redirectPort '%d'", clientID, clientSecret, redirectHost, redirectPort)
		return
	}

	toReturn = &FlowAgent{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectHost: redirectHost,
		redirectPort: redirectPort,
		errChan:      make(chan error, 1),
		doneChan:     make(chan bool, 1),
		tokenChan:    make(chan string, 1),
	}
	return
}

// UserOAuthURL generates and returns the URL the user should use to start the OAuth flow
func (fa *FlowAgent) UserOAuthURL() string {
	oauthOptions := fmt.Sprintf("client_id=%s&scope=%s&redirect_uri=%s", fa.clientID, url.QueryEscape(slackPermsURLFormatted), url.QueryEscape(fa.RedirectURI()))
	return fmt.Sprintf("%s?%s", slackUserOauthURL, oauthOptions)
}

// RedirectURI generates and returns the URI to use for redirection
func (fa *FlowAgent) RedirectURI() string {
	return fmt.Sprintf("http://%s:%d/", fa.redirectHost, fa.redirectPort)
}

// redirectHandler implements http.HandlerFunc for use in responding to the OAuth redirect
func (fa *FlowAgent) redirectHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fa.errChan <- err
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		fa.errChan <- fmt.Errorf("no code given in redirect")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	accessToken, _, err := slack.GetOAuthToken(http.DefaultClient, fa.clientID, fa.clientSecret, code, fa.RedirectURI())
	if err != nil {
		fa.errChan <- err
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fa.tokenChan <- accessToken
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok, token received. it's safe to close me"))
	fa.doneChan <- true
}

// Start begins the OAuth flow by spinning up an http server on the configured port to handle the OAuth redirect.
// Start returns a completion channel and an error channel
// the completion channel will return one time with a true when the token has been received then close. in all other conditions it will close.
// the error channel will be closed when the token has been received. the completion channel will also close an error occurs.
func (fa *FlowAgent) Start() (<-chan bool, <-chan error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", fa.redirectHandler)
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", fa.redirectPort),
		Handler: mux,
	}

	doneChan := make(chan bool)
	errChan := make(chan error)
	serverErrChan := make(chan error)

	go func() {
		err := s.ListenAndServe()
		if err != http.ErrServerClosed {
			serverErrChan <- err
		}
	}()

	go func() {
		defer close(doneChan)
		defer close(errChan)
		defer close(serverErrChan)
		defer close(fa.doneChan)
		defer close(fa.errChan)
		for {
			select {
			case done := <-fa.doneChan:
				if done {
					doneChan <- true
					s.Close()
				}
				return
			case err := <-serverErrChan:
				if err != nil {
					errChan <- s.Close()
					return
				}
			case handlerError := <-fa.errChan:
				if handlerError != nil {
					errChan <- handlerError
					return
				}
			}
		}
	}()
	return doneChan, errChan
}

// AccessToken Blocks until an access token is available. Once available, will always return the same token
func (fa *FlowAgent) AccessToken() string {
	fa.Lock()
	defer fa.Unlock()
	if fa.token == "" {
		fa.token = <-fa.tokenChan
	}

	return fa.token
}
