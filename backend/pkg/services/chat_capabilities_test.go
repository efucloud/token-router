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
	normalized, _, err := normalizeChatSelections(context.Background(), []string{" release-notes "}, nil)
	if err != nil || len(normalized) != 1 || normalized[0] != "release-notes" {
		t.Fatalf("normalize known skill: skills=%#v err=%v", normalized, err)
	}
	if _, _, err = normalizeChatSelections(context.Background(), []string{"missing"}, nil); err == nil {
		t.Fatal("unknown skill was accepted")
	}
	if _, _, err = normalizeChatSelections(context.Background(), []string{"release-notes", "release-notes"}, nil); err == nil {
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

func TestChatMCPRejectsRelativeCommand(t *testing.T) {
	_, err := chatMCP.session(context.Background(), "chat-user", config.ChatMCPConfig{
		Name: "relative", Type: "stdio", Enabled: true, Command: "node",
	})
	if err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("expected absolute command path error, got %v", err)
	}
}

func TestBuiltinChatToolsAndCommandDiscovery(t *testing.T) {
	workspace := t.TempDir()
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, DefaultEnabled: true, WorkspaceDirectory: workspace,
		CommandEnabled: true, CommandTimeoutSeconds: 2, MaxOutputBytes: 4096,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	ctx := context.WithValue(context.Background(), config.RequestUserId, "chat-user")
	capabilities, err := (ChatService{}).Capabilities(ctx)
	if err != nil {
		t.Fatalf("get capabilities: %v", err)
	}
	if len(capabilities.MCPServers) != 1 || capabilities.MCPServers[0].Name != builtinChatServerName ||
		capabilities.MCPServers[0].Kind != "builtin" || !capabilities.MCPServers[0].DefaultEnabled ||
		len(capabilities.MCPServers[0].Tools) != 8 {
		t.Fatalf("unexpected builtin capability: %#v", capabilities.MCPServers)
	}

	service := ChatService{}
	result, err := service.CallMCPTool(ctx, builtinChatServerName, "write_file", map[string]any{
		"path": "manifests/nginx.yaml", "content": "kind: Deployment\nimage: nginx\n",
	})
	if err != nil || result.IsError {
		t.Fatalf("write file: result=%#v err=%v", result, err)
	}
	result, err = service.CallMCPTool(ctx, builtinChatServerName, "edit_file", map[string]any{
		"path": "manifests/nginx.yaml", "oldText": "image: nginx", "newText": "image: nginx:stable",
	})
	if err != nil || result.IsError {
		t.Fatalf("edit file: result=%#v err=%v", result, err)
	}
	result, err = service.CallMCPTool(ctx, builtinChatServerName, "apply_patch", map[string]any{
		"patchText": "*** Begin Patch\n*** Update File: manifests/nginx.yaml\n@@\n-image: nginx:stable\n+image: nginx:alpine\n*** Add File: manifests/README.md\n+Managed by Token Router.\n*** End Patch",
	})
	if err != nil || result.IsError || !strings.Contains(result.Content, "README.md") {
		t.Fatalf("apply patch: result=%#v err=%v", result, err)
	}
	result, err = service.CallMCPTool(ctx, builtinChatServerName, "read_file", map[string]any{
		"path": "manifests/nginx.yaml",
	})
	if err != nil || result.IsError || !strings.Contains(result.Content, "nginx:alpine") {
		t.Fatalf("read file: result=%#v err=%v", result, err)
	}
	result, err = service.CallMCPTool(ctx, builtinChatServerName, "search_files", map[string]any{
		"query": "nginx:alpine",
	})
	if err != nil || result.IsError || !strings.Contains(result.Content, "manifests/nginx.yaml") {
		t.Fatalf("search files: result=%#v err=%v", result, err)
	}
	result, err = service.CallMCPTool(ctx, builtinChatServerName, "command", map[string]any{
		"command": "printf command-ok",
	})
	if err != nil || result.IsError || !strings.Contains(result.Content, "command-ok") {
		t.Fatalf("run command: result=%#v err=%v", result, err)
	}
	result, err = service.CallMCPTool(ctx, builtinChatServerName, "discover_commands", map[string]any{
		"query": "sh",
	})
	if err != nil || result.IsError || !strings.Contains(result.Content, `"commands"`) {
		t.Fatalf("discover commands: result=%#v err=%v", result, err)
	}
}

func TestBuiltinFileToolsRejectWorkspaceEscape(t *testing.T) {
	workspace := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(workspace, "outside")); err != nil {
		t.Fatalf("create outside symlink: %v", err)
	}
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, WorkspaceDirectory: workspace,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	ctx := context.WithValue(context.Background(), config.RequestUserId, "chat-user")
	for _, path := range []string{"../secret.txt", "outside/secret.txt", filepath.Join(outside, "secret.txt")} {
		result, err := (ChatService{}).CallMCPTool(ctx, builtinChatServerName, "read_file", map[string]any{"path": path})
		if err != nil || !result.IsError {
			t.Fatalf("expected path %q to be rejected: result=%#v err=%v", path, result, err)
		}
	}
}
