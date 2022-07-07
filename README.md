# Operand Examples

This repo houses a collection of example projects built using the Operand API.

### List of Projects

- [iMessage Indexing](examples/imessage/README.md): A small application which runs on a macOS machine and indexes all incoming iMessages and SMS messages with Operand. Also supports images & PDF documents.

- [Chatbot w/ Long-Term Memory](examples/long-term-memory/README.md): A GPT-3 base chatbot with support for long-term memory. To accomplish this, we index all incoming messages from the user into an Operand collection. Then, when we recieve a new message from the user, we do a semantic search to find the most relevant context (for the given incoming message) and pass that data into the prompt itself. By doing this, the AI chatbot can "remember" the entire message history.
