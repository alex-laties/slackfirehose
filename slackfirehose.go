package slackfirehose

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/phayes/freeport"
)

const clientID = "20678827605.529118891108"
const clientSecret = "0cce6c1983650e8554882a35c3550cf7"
const slackOauthURL = "https://slack.com/oauth/authorize"

var desiredSlackPermissions = []string{
	"channels:history",
	"channels:read",
	"channels:write",
	"chat:write",
	"chat:write",
	"client",
	"commands",
	"conversations.app_home:create",
	"conversations:history",
	"conversations:read",
	"conversations:write",
	"dnd:read",
	"dnd:write",
	"emoji:read",
	"files:read",
	"files:write",
	"im:read",
	"im:write",
	"im:history",
	"links:read",
	"links:write",
	"mpim:history",
	"mpim:read",
	"mpim:write",
	"pins:write",
	"reactions:read",
	"reactions:write",
	"reminders:read",
	"reminders:write",
	"search:read",
	"stars:read",
	"stars:write",
	"team:read",
	"usergroups:read",
	"usergroups:write",
	"users.profile:read",
	"users.profile:write",
	"users:read",
	"users:read.email",
	"users:write",
}
var slackPermsURLFormatted = strings.Join(desiredSlackPermissions, " ")

func main() {
	// just try to get through oauth flow first
	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err)
	}

	localURL := fmt.Sprintf("http://localhost:%d/oauth/redirect")
	oauthOptions := fmt.Sprintf("client_id=%s&scope=%s&redirect_uri=%s", "20678827605.529118891108", url.QueryEscape(slackPermsURLFormatted), url.QueryEscape(localURL))
	oauthURL := fmt.Sprintf("%s?%s", slackOauthURL, oauthOptions)
	log.Println(fmt.Sprintf("click me: %s", oauthURL))

	http.HandleFunc("/oauth/redirect", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Printf("could not parse: %s", err)
			w.WriteHeader(http.StatusBadRequest)
		}

		code := r.FormValue("code")
		log.Printf("poop: %s", code)
		w.WriteHeader(http.StatusOK)
	})
}
