package messages

import (
	"strconv"
	"testing"

	"github.com/parakeet-nest/parakeet/llm"
)

func TestAddMessage(t *testing.T) {
	tests := []struct {
		role    string
		content string
	}{
		{"user", "Hello, world!"},
		{"assistant", "How can I help you?"},
		{"system", "System message"},
		{"error", "An error occurred"},
	}

	for _, test := range tests {
		manager := NewManager()

		msg := llm.Message{Role: test.role, Content: test.content}
		addedMsg := manager.AddMessage(msg)

		if addedMsg != &msg {
			t.Errorf("AddMessage(%v) = %v, want %v", msg, addedMsg, &msg)
		}

		savedMsg, err := manager.history.Get(strconv.Itoa(manager.currentMessageID))
		if err != nil {
			t.Errorf("Error retrieving saved message: %v", err)
		}
		if savedMsg.Content != msg.Content {
			t.Errorf("Saved message = %v, want %v", savedMsg, msg)
		}
	}
}

func TestMessagesForSending(t *testing.T) {
	tests := []struct {
		messages []llm.Message
		want     []llm.Message
	}{
		{
			messages: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
			want: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
		},
		{
			messages: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "error", Content: "Error"},
			},
			want: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}

	for _, test := range tests {
		manager := NewManager()
		for _, msg := range test.messages {
			manager.AddMessage(msg)
		}

		got, err := manager.MessagesForSending()
		if err != nil {
			t.Errorf("MessagesForSending() error = %v", err)
		}
		if len(got) != len(test.want) {
			t.Errorf("MessagesForSending() = %v, want %v", got, test.want)
		}
		for i := range got {
			if got[i].Content != test.want[i].Content {
				t.Errorf("MessagesForSending()[%d] = %v, want %v", i, got[i], test.want[i])
			}
		}
	}
}

func TestStyledMessages(t *testing.T) {
	tests := []struct {
		messages []llm.Message
		want     []string
	}{
		{
			messages: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi"},
			},
			want: []string{
				"User: Hello\n",
				"Assistant: Hi\n",
			},
		},
		{
			messages: []llm.Message{
				{Role: "system", Content: "System message"},
				{Role: "error", Content: "Error"},
			},
			want: []string{
				"System: System message\n",
				"Error: Error\n",
			},
		},
	}

	for _, test := range tests {
		manager := NewManager()
		for _, msg := range test.messages {
			manager.AddMessage(msg)
		}

		got := manager.StyledMessages()
		if len(got) != len(test.want) {
			t.Logf("Got: %v", got)
			t.Logf("Want: %v", test.want)
			t.Errorf("Len StyledMessages() = %d, want %d", len(got), len(test.want))

		}
		// for i := range got {
		// 	if got[i] != test.want[i] {
		// 		t.Errorf("StyledMessages()[%d] = %v, want %v", i, got[i], test.want[i])
		// 	}
		// }
	}
}
