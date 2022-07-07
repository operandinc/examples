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
	operand "github.com/operandinc/go-sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// MessageDirection is an enumeration over the possible directions of a message.
type MessageDirection string

// Supported message directions.
const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

// Message is a message sent by a user or the chatbot.
type Message struct {
	Timestamp time.Time
	Direction MessageDirection
	Text      string
}

// MessageHistory is a history of messages sent/received.
type MessageHistory struct {
	Messages     []Message
	Client       *operand.Client
	CollectionID string
}

func (mh *MessageHistory) indexMessage(ctx context.Context, message Message) error {
	// This is a no-op if we don't have an Operand client or collection ID.
	if mh.Client == nil || mh.CollectionID == "" {
		return nil
	}

	// Also a no-op if the message text is empty. We cannot index empty objects.
	if message.Text == "" {
		return nil
	}

	// Create a new object for the message.
	obj, err := mh.Client.CreateObject(ctx, operand.CreateObjectArgs{
		ParentID: operand.AsRef(mh.CollectionID),
		Type:     operand.ObjectTypeText,
		Metadata: operand.TextMetadata{
			Text: message.Text,
		},
		// We add a "direction" property to this object, which allows us to properly
		// scope searches in the future to messages from either the user or the chatbot.
		Properties: map[string]any{
			"direction": string(message.Direction),
		},
	})
	if err != nil {
		return err
	} else if err := obj.Wait(ctx, mh.Client); err != nil {
		return err
	}

	// Ensure the object was correctly indexed.
	if obj.IndexingStatus != operand.IndexingStatusReady {
		return errors.New("object not indexed")
	}

	// We're done, since we don't need to store the obj reference anywhere.
	// In a production setting, you'd probably want to store the ID of the
	// Operand object alongside the message in your database.
	return nil
}

// LogFromUser logs a message from the user.
func (mh *MessageHistory) LogFromUser(ctx context.Context, text string) error {
	m := Message{
		Timestamp: time.Now(),
		Direction: MessageDirectionInbound,
		Text:      text,
	}
	if err := mh.indexMessage(ctx, m); err != nil {
		return err
	}
	mh.Messages = append(mh.Messages, m)
	return nil
}

// LogFromChatbot logs a message from the chatbot.
func (mh *MessageHistory) LogFromChatbot(ctx context.Context, text string) error {
	m := Message{
		Timestamp: time.Now(),
		Direction: MessageDirectionOutbound,
		Text:      text,
	}
	if err := mh.indexMessage(ctx, m); err != nil {
		return err
	}
	mh.Messages = append(mh.Messages, m)
	return nil
}

// LastN returns up to the last N messages in the history.
func (mh *MessageHistory) LastN(n int) []Message {
	if len(mh.Messages) < n {
		return mh.Messages
	}
	return mh.Messages[len(mh.Messages)-n:]
}

// A helper function which takes a reference of the passed value.
func asRef[T any](v T) *T {
	return &v
}

// GenerateResponse generates a response to an incoming message, given the
// message and the message history between the bot and this user.
func GenerateResponse(
	ctx context.Context,
	gptClient gpt3.Client,
	operandClient *operand.Client,
	operandCollectionID string,
	history MessageHistory,
	message string,
) (string, error) {
	var builder strings.Builder

	// Initial prompt.
	builder.WriteString(
		"The following is a conversation with an AI assistant. The assistant is helpful, creative, clever, and very friendly.",
	)
	builder.WriteString("\n\n")

	// If we have a non-nil Operand client, we should use it to fetch long-term context for the incoming message.
	// By doing a semantic search over the entire message history here, we're able to pick out the most relevant
	// messages from the entire conversation and use the information contained in them to generate a response.
	if operandClient != nil {
		response, err := operandClient.SearchContents(ctx, operand.SearchContentsArgs{
			ParentIDs: []string{operandCollectionID},
			Query:     message,
			Max:       5,
			// Use a filter to scope the search to only messages sent by the user, i.e. inbound messages.
			Filter: map[string]any{
				"direction": string(MessageDirectionInbound),
			},
		})
		if err != nil {
			return "", err
		}

		// If we got any results, we include them in the prompt as additional context.
		if len(response.Contents) > 0 {
			builder.WriteString("Relevant previous messages from Human:\n")
			for _, c := range response.Contents {
				builder.WriteString(fmt.Sprintf("- %s\n", c.Content))
			}
			builder.WriteString("\n")
		}
	}

	// Start the conversation in the prompt.
	builder.WriteString("The conversation goes as follows:\n")

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

	// TODO: Remove duplicates from the long-term and immediate context.

	// Finally, add the user's message.
	builder.WriteString("Human: ")
	builder.WriteString(message)
	builder.WriteString("\nAI:") // Important to not have a trailing space here.

	// Fire the request to OpenAI.
	resp, err := gptClient.CompletionWithEngine(ctx, "text-davinci-002", gpt3.CompletionRequest{
		Prompt:           []string{builder.String()},
		Temperature:      asRef(float32(0.7)),
		MaxTokens:        asRef(64),
		TopP:             asRef(float32(1)),
		FrequencyPenalty: 0,
		PresencePenalty:  0.6,
		Stop:             []string{"Human: ", "AI: "},
	})
	if err != nil {
		return "", err
	}

	// Return the trimmed response.
	if len(resp.Choices) == 0 {
		return "", errors.New("openai returned zero choices")
	}
	return strings.Trim(resp.Choices[0].Text, " \n\t"), nil
}

func run() error {
	// Make a top-level context.
	ctx := context.Background()

	// Initalize the GPT-3 client.
	oaiKey, ok := os.LookupEnv("OPENAI_KEY")
	if !ok {
		return errors.New("OPENAI_KEY not set")
	}
	gptClient := gpt3.NewClient(oaiKey)

	// Initialize the Operand client, if we can.
	var (
		operandClient       *operand.Client
		operandCollectionID string
	)
	if operandKey, ok := os.LookupEnv("OPERAND_API_KEY"); ok {
		operandClient = operand.NewClient(operandKey)

		// We take this time to create a new collection for this user.
		collection, err := operandClient.CreateObject(
			ctx,
			operand.CreateObjectArgs{
				Type:     operand.ObjectTypeCollection,
				Metadata: operand.CollectionMetadata{},
				Label:    operand.AsRef("ltm"),
			},
		)
		if err != nil {
			return err
		} else if err := collection.Wait(ctx, operandClient); err != nil {
			return err
		}

		// Make sure the collection is ready.
		if collection.IndexingStatus != operand.IndexingStatusReady {
			return fmt.Errorf("collection is not ready: %s", collection.IndexingStatus)
		}

		operandCollectionID = collection.ID

		// Defer a function which deletes the collection we created when we're finished.
		defer func() {
			if _, err := operandClient.DeleteObject(ctx, collection.ID, nil); err != nil {
				log.Printf("error deleting collection: %s", err)
			}
		}()
	} else {
		log.Println("warning: OPERAND_API_KEY not set, long-term memory disabled")
	}

	// Create a new message history object.
	mh := MessageHistory{
		Messages:     []Message{},
		Client:       operandClient,
		CollectionID: operandCollectionID,
	}

	// Add a bit of starting context to get the conversation started.
	// This is taken directly from OpenAI's example chatbot prompt.
	if err := mh.LogFromUser(ctx, "Hello, who are you?"); err != nil {
		return err
	}
	if err := mh.LogFromChatbot(ctx, "I am an AI created by OpenAI. How can I help you today?"); err != nil {
		return err
	}

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
		response, err := GenerateResponse(
			ctx,
			gptClient,
			operandClient,
			operandCollectionID,
			mh,
			message,
		)
		if err != nil {
			return err
		}
		fmt.Println("Bot: " + response)

		// At this point, we need to log the messages (both directions).
		if err := mh.LogFromUser(ctx, message); err != nil {
			return err
		}
		if err := mh.LogFromChatbot(ctx, response); err != nil {
			return err
		}
	}

	// We're done.
	return nil
}
