# iMessage Indexing Demo

This repository allows anyone to index their personal iMessage history with [Operand](https://operand.ai).

To get started, you'll need to get an API key from the Operand Dashboard. You can sign up [here](https://operand.ai/auth), and if you have any questions or feedback, please don't hesistate to [reach out](mailto:morgan@operand.ai)!

### Setup Instructions

This demo requires a machine running macOS. A laptop will work, or a dedicated server if you're feeling fancy.

1. Install [Go](https://golang.org) if you haven't already.
2. Clone [ABCS](https://github.com/operandinc/abcs), which provides the underlying functionality for iMessage sending and receiving.
3. Clone this repository, and configure the following environment variables (in a `.env` file):

```
OPERAND_ENDPOINT=https://prod.operand.ai
OPERAND_PARENT_ID=<collection id>
OPERAND_API_KEY=<api key>
```

The parent ID is an optional variable which essentially tells the system what folder within Operand you'd like to put the iMessages (and associated objects). You can create a new collection with the object browser, and then copy it's ID using the secondary action menu. The API key can be seen on the dashboard by navigating to Settings -> API Keys.

4. (Optional) Configure an S3 bucket.

We use S3 to store the underlying data for attachments. If the following variables are included in the environment, S3 will be used for attachments and the system will automatically index images, pdfs, etc.

Example configuration:
```
S3_ENDPOINT=https://nyc3.digitaloceanspaces.com
S3_REGION=us-east-1
S3_BUCKET=<your bucket name>
S3_KEY=<your s3 key>
S3_SECRET=<your s3 secret>
```

5. Start the system by running the following commands (in seperate terminals) on your Mac machine. In order for messages to be received, `Terminal` must be given `Full Disk Access` (Settings -> Security).

Listen for incoming messages:
```
./abcs -listen=127.0.0.1:11106 -endpoint=http://127.0.0.1:8080
```

Handle incoming messages:
```
PORT=8080 go run imd.go
```

### Some other notes

- The system is currently rather limited, as this is more of a proof-of-concept rather than something "production ready".
- We recommend [MacStadium](https://www.macstadium.com/) for hosting, we've had a great experience with it in the past.
- We'd happily accept PRs to add additional functionality here, or to [ABCS](https://github.com/operandinc/abcs). Specifically, adding the option to index outgoing messages (rather than just incoming) and to support additional object types.


