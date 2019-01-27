package slackfirehose

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/alex-laties/slackfirehose/oauth"
	"github.com/caarlos0/env"
	"github.com/nlopes/slack"
)

type config struct {
	clientID     string `env:"CLIENT_ID"`
	clientSecret string `env:"CLIENT_SECRET"`
	port         int    `env:"PORT"`
}

// Run loads up a local http server and prints out to stdout a URL to use
func Run() error {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalln(err)
	}

	agent, err := oauth.NewFlowAgent(cfg.clientID, cfg.clientSecret, "localhost", cfg.port)
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
