# My Golang Chatbot

This project is a chat bot front end built using the Bubbletea TUI framework in Go. It interacts with the Ollama chat bot to provide a user-friendly terminal interface for chatting.

## Project Structure

```
my-golang-chatbot
├── cmd
│   └── main.go          # Entry point of the application
├── internal
│   ├── tui
│   │   └── tui.go      # Implementation of the TUI using Bubbletea
│   └── bot
│       └── bot.go      # Logic for interacting with the Ollama chat bot
├── go.mod               # Module definition and dependencies
├── go.sum               # Checksums for module dependencies
└── README.md            # Project documentation
```

## Setup Instructions

1. Clone the repository:
   ```
   git clone <repository-url>
   cd my-golang-chatbot
   ```

2. Install the required dependencies:
   ```
   go mod tidy
   ```

3. Run the application:
   ```
   go run cmd/main.go
   ```

## Usage

Once the application is running, you can start chatting with the Ollama bot directly from the terminal interface. Type your messages and press Enter to send them.

## Contributing

Feel free to submit issues or pull requests to improve the project!