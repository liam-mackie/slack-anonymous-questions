package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/slack-go/slack/socketmode"

	"github.com/slack-go/slack"
)

var HashToChannelMap = make(map[string]string)

func main() {
	appToken := os.Getenv("SLACK_APP_TOKEN")
	if appToken == "" {
		panic("SLACK_APP_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(appToken, "xapp-") {
		panic("SLACK_APP_TOKEN must have the prefix \"xapp-\".")
	}

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if botToken == "" {
		panic("SLACK_BOT_TOKEN must be set.\n")
	}

	if !strings.HasPrefix(botToken, "xoxb-") {
		panic("SLACK_BOT_TOKEN must have the prefix \"xoxb-\".")
	}

	api := slack.New(
		botToken,
		slack.OptionLog(log.New(os.Stdout, "api: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionAppLevelToken(appToken),
	)

	client := socketmode.New(
		api,
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	socketmodeHandler := socketmode.NewSocketmodeHandler(client)

	socketmodeHandler.Handle(socketmode.EventTypeConnecting, middlewareConnecting)
	socketmodeHandler.Handle(socketmode.EventTypeConnectionError, middlewareConnectionError)
	socketmodeHandler.Handle(socketmode.EventTypeConnected, middlewareConnected)
	socketmodeHandler.Handle(socketmode.EventTypeHello, middlewareHello)

	// Handle slashcommand
	socketmodeHandler.Handle(socketmode.EventTypeSlashCommand, middlewareSlashCommand)
	socketmodeHandler.HandleSlashCommand("/askanon", middlewareSlashCommand)

	// Handle interactive elements
	socketmodeHandler.Handle(socketmode.EventTypeInteractive, middlewareInteractive)

	socketmodeHandler.RunEventLoop()
}

func middlewareConnecting(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connecting to Slack with Socket Mode...")
}

func middlewareConnectionError(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connection failed. Retrying later...")
}

func middlewareConnected(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Connected to Slack with Socket Mode.")
}

func middlewareHello(evt *socketmode.Event, client *socketmode.Client) {
	fmt.Println("Slack says Hello!")
}

func middlewareSlashCommand(evt *socketmode.Event, client *socketmode.Client) {
	cmd, ok := evt.Data.(slack.SlashCommand)
	if !ok {
		return
	}
	switch cmd.Command {
	case "/askanon":
		modalRequest := generateModalRequest()
		response, _ := client.Client.OpenView(cmd.TriggerID, modalRequest)
		HashToChannelMap[response.Hash] = cmd.ChannelID
	}

	client.Ack(*evt.Request)
}

func middlewareInteractive(evt *socketmode.Event, client *socketmode.Client) {
	callback, ok := evt.Data.(slack.InteractionCallback)
	if !ok {
		return
	}

	var payload interface{}

	switch callback.Type {
	case slack.InteractionTypeViewSubmission:
		question := callback.View.State.Values["Question"]["question"].Value
		questionBlock := generateQuestionBlocks(question)
		channelId := HashToChannelMap[callback.View.Hash]
		delete(HashToChannelMap, callback.View.Hash)
		_, _, err := client.Client.PostMessage(channelId, slack.MsgOptionBlocks(questionBlock...))
		if err != nil {
			fmt.Print("could not post question")
		}
	}

	client.Ack(*evt.Request, payload)
}

func generateQuestionBlocks(question string) []slack.Block {
	blocks := []slack.Block{
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "*New Question Asked*",
			},
			nil,
			nil,
		),
		slack.NewSectionBlock(
			&slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: question,
			},
			nil,
			nil),
	}
	return blocks
}

func generateModalRequest() slack.ModalViewRequest {
	titleText := slack.NewTextBlockObject("plain_text", "Ask a question", false, false)
	closeText := slack.NewTextBlockObject("plain_text", "Close", false, false)
	submitText := slack.NewTextBlockObject("plain_text", "Submit", false, false)

	headerText := slack.NewTextBlockObject("mrkdwn", "Please enter your question", false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)

	questionText := slack.NewTextBlockObject("plain_text", "Question", false, false)
	questionHint := slack.NewTextBlockObject("plain_text", "Enter your question here", false, false)
	questionPlaceholder := slack.NewTextBlockObject("plain_text", "Enter your question", false, false)
	questionElement := slack.NewPlainTextInputBlockElement(questionPlaceholder, "question")
	// Notice that blockID is a unique identifier for a block
	question := slack.NewInputBlock("Question", questionText, questionHint, questionElement)

	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			headerSection,
			question,
		},
	}

	var modalRequest slack.ModalViewRequest
	modalRequest.Type = slack.ViewType("modal")
	modalRequest.Title = titleText
	modalRequest.Close = closeText
	modalRequest.Submit = submitText
	modalRequest.Blocks = blocks
	return modalRequest
}
