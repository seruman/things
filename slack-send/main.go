package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	if err := realMain(os.Args, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func realMain(args []string, stdin io.Reader, stdout io.Writer) error {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	flagToken := flags.String("token", os.Getenv("SLACK_TOKEN"), "Slack auth token.")
	flagChannel := flags.String("channel", os.Getenv("SLACK_CHANNEL"), "Slack channel to send message.")
	flagBlocks := flags.Bool("blocks", false, "Read message as 'blocks' JSON")
	flagUsername := flags.String("username", "", "Username to send message.")
	flagIconEmoji := flags.String("icon-emoji", "", "Icon emoji to send message.")

	err := flags.Parse(args[1:])
	if err != nil {
		return err
	}

	if *flagToken == "" {
		return fmt.Errorf("token is required")
	}

	if *flagChannel == "" {
		return fmt.Errorf("channel is required")
	}

	content, err := ioutil.ReadAll(stdin)
	if err != nil {
		return err
	}

	var msgOption slack.MsgOption = slack.MsgOptionText(string(content), false)
	if *flagBlocks {
		var blocks slack.Blocks
		err := json.Unmarshal(content, &blocks)
		if err != nil {
			return fmt.Errorf("unmarshal blocks: %w", err)
		}
		msgOption = slack.MsgOptionBlocks(blocks.BlockSet...)
	}

	options := []slack.MsgOption{
		msgOption,
	}

	if *flagUsername != "" {
		options = append(options, slack.MsgOptionUsername(*flagUsername))
	}

	if *flagIconEmoji != "" {
		options = append(options, slack.MsgOptionIconEmoji(*flagIconEmoji))
	}

	client := slack.New(*flagToken)
	_, _, err = client.PostMessage(
		*flagChannel,
		options...,
	)
	if err != nil {
		return err
	}

	return nil
}
