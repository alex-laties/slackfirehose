package slackfirehose

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/alex-laties/slackfirehose/oauth"
	"github.com/nlopes/slack"
)

const clientID = "20678827605.529118891108"
const clientSecret = "0cce6c1983650e8554882a35c3550cf7"
const port = 49953

// Run loads up a local http server and prints out to stdout a URL to use
func Run() error {
	agent, err := oauth.NewFlowAgent(clientID, clientSecret, "localhost", port)
	if err != nil {
		return err
	}
	doneChan, errChan := agent.Start()
	log.Printf("click me: %s", agent.UserOAuthURL())
	select {
	case done := <-doneChan:
		if !done {
			return errors.New("something went wrong with the token exchange")
		}
	case err := <-errChan:
		return err
	}

	log.Printf("token exchange went fine. starting up firehose")
	firehose(agent.AccessToken())
	return nil
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
