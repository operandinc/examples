package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PullRequestInc/go-gpt3"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// MessageDirection is an enumeration over the possible directions of a message.
type MessageDirection int

// Supported message directions.
const (
	MessageDirectionInbound MessageDirection = iota
	MessageDirectionOutbound
)

// Message is a message sent by a user or the chatbot.
type Message struct {
	Timestamp time.Time
	Direction MessageDirection
	Text      string
}

// MessageHistory is a history of messages sent/received.
type MessageHistory []Message

// LogFromUser logs a message from the user.
func (mh *MessageHistory) LogFromUser(text string) {
	*mh = append(*mh, Message{
		Timestamp: time.Now(),
		Direction: MessageDirectionInbound,
		Text:      text,
	})
}

// LogFromChatbot logs a message from the chatbot.
func (mh *MessageHistory) LogFromChatbot(text string) {
	*mh = append(*mh, Message{
		Timestamp: time.Now(),
		Direction: MessageDirectionOutbound,
		Text:      text,
	})
}

// LastN returns up to the last N messages in the history.
func (mh *MessageHistory) LastN(n int) []Message {
	if len(*mh) < n {
		return *mh
	}
	return (*mh)[len(*mh)-n:]
}

// A helper function which takes a reference of the passed value.
func asRef[T any](v T) *T {
	return &v
}

// GenerateResponse generates a response to an incoming message, given the
// message and the message history between the bot and this user.
func GenerateResponse(
	ctx context.Context,
	client gpt3.Client,
	history MessageHistory,
	message string,
) (string, error) {
	var builder strings.Builder

	// Initial prompt.
	builder.WriteString(
		"The following is a conversation with an AI assistant. The assistant is helpful, creative, clever, and very friendly.",
	)
	builder.WriteString("\n\n")

	// Fetch the last few messages from the conversation. This gives the bot some immediate
	// context to use when generating a response. Experimentally, usually keeping this to a moderate
	// size is best (we also don't want the prompts getting too big).
	previous := history.LastN(5)
	for _, m := range previous {
		if m.Direction == MessageDirectionInbound {
			builder.WriteString("Human: ")
		} else {
			builder.WriteString("AI: ")
		}
		builder.WriteString(m.Text)
		builder.WriteString("\n")
	}

	// Finally, add the user's message.
	builder.WriteString("Human: ")
	builder.WriteString(message)
	builder.WriteString("\nAI:")

	// Fire the request to OpenAI.
	resp, err := client.CompletionWithEngine(ctx, "text-davinci-002", gpt3.CompletionRequest{
		Prompt:           []string{builder.String()},
		Temperature:      asRef(float32(0.9)),
		MaxTokens:        asRef(64),
		TopP:             asRef(float32(1)),
		FrequencyPenalty: 0,
		PresencePenalty:  0.6,
		Stop:             []string{"Human: ", "AI: ", "\n"},
	})
	if err != nil {
		return "", err
	}

	// Return the trimmed response.
	if len(resp.Choices) == 0 {
		return "", errors.New("openai returned zero choices")
	}
	return strings.TrimSpace(resp.Choices[0].Text), nil
}

func run() error {
	// Initalize the GPT-3 client.
	key, ok := os.LookupEnv("OPENAI_KEY")
	if !ok {
		return errors.New("OPENAI_KEY not set")
	}
	client := gpt3.NewClient(key)

	// Create a new message history object.
	mh := MessageHistory{}

	// Add a bit of starting context to get the conversation started.
	mh.LogFromUser("Hello, who are you?")
	mh.LogFromChatbot("I am an AI created by OpenAI. How can I help you today?")

	// Start the loop, i.e. the conversation between the user and the AI chatbot.
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Prompt the user for a message.
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		// Get the user's message.
		message := strings.TrimSpace(scanner.Text())
		if message == "quit" {
			break
		} else if message == "" {
			continue
		}

		// Generate a message and respond with it.
		response, err := GenerateResponse(context.Background(), client, mh, message)
		if err != nil {
			return err
		}
		fmt.Println("Bot: " + response)

		// At this point, we need to log the messages (both directions).
		mh.LogFromUser(message)
		mh.LogFromChatbot(response)
	}

	// We're done.
	return nil
}
