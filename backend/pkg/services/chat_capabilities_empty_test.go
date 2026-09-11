package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
)

func TestChatCapabilitiesSerializeEmptyMCPServersAsArray(t *testing.T) {
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	ctx := context.WithValue(context.Background(), config.RequestUserId, "chat-user")
	capabilities, err := (ChatService{}).Capabilities(ctx)
	if err != nil {
		t.Fatalf("get capabilities: %v", err)
	}
	encoded, err := json.Marshal(capabilities)
	if err != nil {
		t.Fatalf("marshal capabilities: %v", err)
	}
	if !strings.Contains(string(encoded), `"mcpServers":[]`) {
		t.Fatalf("expected an empty MCP server array, got %s", encoded)
	}
}
