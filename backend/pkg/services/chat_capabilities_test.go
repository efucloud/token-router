package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDiscoverChatSkillsAndValidateSelection(t *testing.T) {
	directory := t.TempDir()
	skillDirectory := filepath.Join(directory, "release-notes")
	if err := os.Mkdir(skillDirectory, 0o755); err != nil {
		t.Fatalf("create skill directory: %v", err)
	}
	content := "---\nname: release-notes\ndescription: Draft reliable release notes\n---\n\nAlways cite the included changes.\n"
	if err := os.WriteFile(filepath.Join(skillDirectory, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{SkillDirectories: []string{directory}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	skills, issues := discoverChatSkills()
	if len(issues) != 0 || len(skills) != 1 || skills[0].Name != "release-notes" || skills[0].Content != content {
		t.Fatalf("unexpected skill discovery: skills=%#v issues=%#v", skills, issues)
	}
	normalized, _, err := normalizeChatSelections([]string{" release-notes "}, nil)
	if err != nil || len(normalized) != 1 || normalized[0] != "release-notes" {
		t.Fatalf("normalize known skill: skills=%#v err=%v", normalized, err)
	}
	if _, _, err = normalizeChatSelections([]string{"missing"}, nil); err == nil {
		t.Fatal("unknown skill was accepted")
	}
	if _, _, err = normalizeChatSelections([]string{"release-notes", "release-notes"}, nil); err == nil {
		t.Fatal("duplicate skill was accepted")
	}
}

func TestChatMCPDiscoveryAndCall(t *testing.T) {
	type echoInput struct {
		Text string `json:"text"`
	}
	type echoOutput struct {
		Echo string `json:"echo"`
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo text"}, func(_ context.Context, _ *mcp.CallToolRequest, input echoInput) (*mcp.CallToolResult, echoOutput, error) {
		return nil, echoOutput{Echo: input.Text}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	httpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Test-Auth") != "allowed" {
			http.Error(writer, "missing auth", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(writer, request)
	}))
	defer httpServer.Close()

	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{MCPServers: []config.ChatMCPConfig{{
		Name: "test-server", Type: "streamable-http", Enabled: true, URL: httpServer.URL,
		Headers: map[string]string{"X-Test-Auth": "allowed"}, TimeoutSeconds: 2,
	}}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() {
		chatMCP.evict("chat-user", "test-server")
		config.ApplicationConfig.Chat = original
	})

	ctx := context.WithValue(context.Background(), config.RequestUserId, "chat-user")
	capabilities, err := (ChatService{}).Capabilities(ctx)
	if err != nil {
		t.Fatalf("get capabilities: %v", err)
	}
	if len(capabilities.MCPServers) != 1 || capabilities.MCPServers[0].Status != "connected" || len(capabilities.MCPServers[0].Tools) != 1 {
		t.Fatalf("unexpected MCP capabilities: %#v", capabilities.MCPServers)
	}
	result, err := (ChatService{}).CallMCPTool(ctx, "test-server", "echo", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("call MCP tool: %v", err)
	}
	if result.IsError || !strings.Contains(result.Content, "hello") {
		t.Fatalf("unexpected MCP result: %#v", result)
	}
	if _, err = (ChatService{}).CallMCPTool(ctx, "test-server", "missing", map[string]any{}); err == nil {
		t.Fatal("unknown MCP tool was accepted")
	}
}
