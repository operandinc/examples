# Long Term Memory Example

This is an example of adding long-term memory to a GPT-3 based chatbot using the [Operand API](https://operand.ai). To get started, you'll need to sign up for an account and get an API Key (Settings -> API Keys). If you have any questions or feedback items, feel free to [send us an email](mailto:support@operand.ai).

This example is part of larger [blog post](https://operand.ai/blog/long-term-memory).

In this demo, we take a basic GPT-3 chatbot and use semantic search to add "long-term memory" to it, i.e. referencing past messages to allow the bot to reference previous conversations it's had with the user. Specifically, we pass in two "context windows" into the prompt:

- The first context window, the immediate context window, includes the content of the last ~5 messages or so from the conversation. This provides the model with the immediate context of the conversation.
- The long-term context window includes the top N most relevant messages sent by the user (at any time, and in any order). Specifically, to generate the long-term context window, we do a semantic search to find the most relevant content and include that.

### Usage

Using the chatbot for yourself is relatively easy. You'll need both an [OpenAI](https://openai.com) and [Operand](https://operand.ai) API key, and [Go](https://golang.org) 1.18+ installed.

```
git clone git@github.com:operandinc/examples.git
cd examples/long-term-memory
OPENAI_KEY=<your API key> OPERAND_API_KEY=<your API key> go run ltm.go
```

### Example Conversation

```
OPENAI_KEY=<your API key> OPERAND_API_KEY=<your API key> go run ltm.go
You: How are you?
Bot: I'm doing well today. How can I help you?
You: You can't! Did tou know that my favorite coffee beverage these days is oat milk lattes?
Bot: I did not know that. Thank you for sharing.
You: Busy day?
Bot: Yes, I've been quite busy today. Thanks for asking!
You: What've you been up to?
Bot: I've been helping people with their inquiries and doing my best to be helpful.
You: Is that what you do?
Bot: Yes, I enjoy helping people.
You: Why?
Bot: Because it's rewarding to know that I can be of assistance.
You: That's pretty admirable!
Bot: Thank you, I appreciate your feedback.
You: Btw, do you remember what my favorite coffee beverage was?
Bot: Yes, you said that your favorite coffee beverage is oat milk lattes.
```

Note the last two messages: the bot was able to "remember" what the user told it a long while ago. This data was retrieved via semantic search and included as context in the prompt.
