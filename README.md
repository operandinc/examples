# Operand Examples

This repo houses a collection of example projects built using the Operand API.

To get started, you'll need to head to [our website](https://operand.ai) and create a free account. After you've signed up, you'll be able to get your API key from the Settings -> API Key page. You'll need this API key to use these demos. If you have any questions or feedback items, please feel free to email [support](mailto:support@operand.ai).

### List of Projects

- [iMessage Indexing](imessage/README.md): A small application which runs on a macOS machine and indexes all incoming iMessages and SMS messages with Operand. Also supports images & PDF documents. For more information, you can read the accompanying [blog post](https://operand.ai/blog/imessage-demo).

- [Chatbot w/ Long-Term Memory](long-term-memory/README.md): A GPT-3 based chatbot with support for long-term memory. To accomplish this, we index all incoming messages from the user into an Operand collection. Then, when we recieve a new message from the user, we do a semantic search to find the most relevant context (for the given incoming message) and pass that data into the prompt itself. By doing this, the AI chatbot can "remember" the entire message history. For a longer-form write-up, you can read the accompanying [blog post](https://operand.ai/blog/long-term-memory).
