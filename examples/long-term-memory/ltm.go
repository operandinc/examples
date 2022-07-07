package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
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
func (mh MessageHistory) LogFromUser(text string) {
	mh = append(mh, Message{
		Timestamp: time.Now(),
		Direction: MessageDirectionInbound,
		Text:      text,
	})
}

// LogFromChatbot logs a message from the chatbot.
func (mh MessageHistory) LogFromChatbot(text string) {
	mh = append(mh, Message{
		Timestamp: time.Now(),
		Direction: MessageDirectionOutbound,
		Text:      text,
	})
}

// LastN returns up to the last N messages in the history.
func (mh MessageHistory) LastN(n int) []Message {
	if len(mh) < n {
		return mh
	}
	return mh[len(mh)-n:]
}

func run() error {
	// Create a new message history.
	mh := MessageHistory{}

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
		response := message
		fmt.Println("Bot: " + response)

		// At this point, we need to log the messages (both directions).
		mh.LogFromUser(message)
		mh.LogFromChatbot(response)
	}

	// We're done.
	return nil
}
