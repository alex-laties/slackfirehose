package slackfirehose

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

const clientID = "20678827605.529118891108"
const clientSecret = "0cce6c1983650e8554882a35c3550cf7"
const slackOauthURL = "https://slack.com/oauth/authorize"
const slackTokenURL = "https://slack.com/api/oauth.access"
const port = 49953

var desiredSlackPermissions = []string{
	"client",
}
var slackPermsURLFormatted = strings.Join(desiredSlackPermissions, " ")

type SlackOAuthResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TeamName    string `json:"team_name"`
	TeamID      string `json:"team_id"`
}

// Run loads up a local http server and prints out to stdout a URL to use
func Run() error {
	// just try to get through oauth flow first
	localURL := fmt.Sprintf("http://localhost:%d/oauth/redirect", port)
	oauthOptions := fmt.Sprintf("client_id=%s&scope=%s&redirect_uri=%s", "20678827605.529118891108", url.QueryEscape(slackPermsURLFormatted), url.QueryEscape(localURL))
	oauthURL := fmt.Sprintf("%s?%s", slackOauthURL, oauthOptions)
	log.Println(fmt.Sprintf("click me: %s", oauthURL))

	http.HandleFunc("/oauth/redirect", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Printf("could not parse: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")
		resp, err := http.PostForm(
			slackTokenURL,
			url.Values{
				"client_id":     {clientID},
				"client_secret": {clientSecret},
				"code":          {code},
				"redirect_uri":  {localURL},
			},
		)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		if resp.StatusCode != http.StatusOK {
			w.WriteHeader(resp.StatusCode)
		}

		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		var tokenresp SlackOAuthResponse
		err = json.Unmarshal(contents, &tokenresp)
		if err != nil {
			panic(err)
		}

		token := tokenresp.AccessToken
		log.Println(token)

		go firehose(token)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok, firehose running"))
	})
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func firehose(token string) {
	api := slack.New(
		token,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		fmt.Printf("%v\n", msg)
	}
}
