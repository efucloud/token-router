package services

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
)

func workspaceTestContext(accountID string) context.Context {
	return context.WithValue(context.Background(), config.RequestUserId, accountID)
}

func TestPersonalWorkspaceDefaultsToWorkingDirectory(t *testing.T) {
	workingDirectory := t.TempDir()
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err = os.Chdir(workingDirectory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWorkingDirectory) })

	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{Enabled: true}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	root, err := personalWorkspaceRoot(workspaceTestContext("default-workspace-user"))
	if err != nil {
		t.Fatalf("create default workspace: %v", err)
	}
	expectedBase := filepath.Join(workingDirectory, "workspace")
	expectedBase, err = filepath.EvalSymlinks(expectedBase)
	if err != nil {
		t.Fatalf("resolve default workspace: %v", err)
	}
	if !pathWithin(expectedBase, root) {
		t.Fatalf("default workspace %q is outside %q", root, expectedBase)
	}
	if info, statErr := os.Stat(expectedBase); statErr != nil || !info.IsDir() {
		t.Fatalf("default workspace directory was not created: %v", statErr)
	}
}

func TestPersonalWorkspaceIsStableAndAccountIsolated(t *testing.T) {
	base := t.TempDir()
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, DefaultEnabled: true, WorkspaceDirectory: base, MaxUploadBytes: 1024,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	service := ChatService{}
	userA := workspaceTestContext("oidc/user-a")
	userB := workspaceTestContext("oidc/user-b")
	entry, err := service.UploadWorkspaceFile(userA, "", "notes.txt", strings.NewReader("only user a"))
	if err != nil || entry.Path != "notes.txt" {
		t.Fatalf("upload user A file: entry=%#v err=%v", entry, err)
	}

	rootA, err := personalWorkspaceRoot(userA)
	if err != nil {
		t.Fatalf("resolve user A root: %v", err)
	}
	rootAAgain, _ := personalWorkspaceRoot(userA)
	rootB, err := personalWorkspaceRoot(userB)
	if err != nil {
		t.Fatalf("resolve user B root: %v", err)
	}
	if rootA != rootAAgain || rootA == rootB || filepath.Base(rootA) == "oidc/user-a" || len(filepath.Base(rootA)) != 64 {
		t.Fatalf("unexpected workspace mapping: A=%q A2=%q B=%q", rootA, rootAAgain, rootB)
	}

	listingA, err := service.ListWorkspace(userA, "")
	if err != nil || len(listingA.Entries) != 1 || listingA.Entries[0].Name != "notes.txt" {
		t.Fatalf("list user A workspace: listing=%#v err=%v", listingA, err)
	}
	listingB, err := service.ListWorkspace(userB, "")
	if err != nil || len(listingB.Entries) != 0 {
		t.Fatalf("user B saw another workspace: listing=%#v err=%v", listingB, err)
	}

	file, downloaded, err := service.OpenWorkspaceFile(userA, "notes.txt")
	if err != nil {
		t.Fatalf("open user A file: %v", err)
	}
	content, readErr := io.ReadAll(file)
	_ = file.Close()
	if readErr != nil || downloaded.Name != "notes.txt" || string(content) != "only user a" {
		t.Fatalf("unexpected download: entry=%#v content=%q err=%v", downloaded, content, readErr)
	}
	if _, _, err = service.OpenWorkspaceFile(userB, "notes.txt"); WorkspaceHTTPStatus(err) != 404 {
		t.Fatalf("user B opened user A file: %v", err)
	}
}

func TestPersonalWorkspaceRejectsEscapeAndCrossUserSymlink(t *testing.T) {
	base := t.TempDir()
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, WorkspaceDirectory: base,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	userA := workspaceTestContext("user-a")
	userB := workspaceTestContext("user-b")
	rootA, _ := personalWorkspaceRoot(userA)
	rootB, _ := personalWorkspaceRoot(userB)
	if err := os.WriteFile(filepath.Join(rootB, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write user B secret: %v", err)
	}
	if err := os.Symlink(rootB, filepath.Join(rootA, "other-user")); err != nil {
		t.Fatalf("create cross-user symlink: %v", err)
	}

	for _, path := range []string{"../secret.txt", filepath.Join(rootB, "secret.txt"), "other-user/secret.txt"} {
		_, _, err := (ChatService{}).OpenWorkspaceFile(userA, path)
		if err == nil || WorkspaceHTTPStatus(err) != 400 {
			t.Fatalf("expected %q to be rejected, got %v", path, err)
		}
		if strings.Contains(err.Error(), base) || strings.Contains(err.Error(), rootB) {
			t.Fatalf("physical path leaked for %q: %v", path, err)
		}
	}
	if _, err := (ChatService{}).UploadWorkspaceFile(userA, "", "../secret.txt", strings.NewReader("x")); WorkspaceHTTPStatus(err) != 400 {
		t.Fatalf("unsafe upload filename was accepted: %v", err)
	}
}

func TestPersonalWorkspaceUploadLimitLeavesNoTemporaryFile(t *testing.T) {
	base := t.TempDir()
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, WorkspaceDirectory: base, MaxUploadBytes: 4,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	ctx := workspaceTestContext("limited-user")
	_, err := (ChatService{}).UploadWorkspaceFile(ctx, "", "large.txt", strings.NewReader("12345"))
	if WorkspaceHTTPStatus(err) != 413 {
		t.Fatalf("expected upload limit error, got %v", err)
	}
	root, _ := personalWorkspaceRoot(ctx)
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatalf("read workspace: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("oversized upload left files behind: %#v", entries)
	}
}

func TestCommandEnabledPublishesCommandToAuthenticatedUser(t *testing.T) {
	base := t.TempDir()
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, DefaultEnabled: true, WorkspaceDirectory: base,
		CommandEnabled: true,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })

	ctx := workspaceTestContext("ordinary-workspace-user")
	capabilities, err := (ChatService{}).Capabilities(ctx)
	if err != nil || len(capabilities.MCPServers) != 1 {
		t.Fatalf("personal file capability missing: %#v err=%v", capabilities, err)
	}
	commandPublished := false
	for _, tool := range capabilities.MCPServers[0].Tools {
		if tool.Name == "command" {
			commandPublished = true
		}
	}
	if !commandPublished {
		t.Fatal("command was not published when commandEnabled is true")
	}
	result, err := (ChatService{}).CallMCPTool(ctx, builtinChatServerName, "write_file", map[string]any{
		"path": "welcome.txt", "content": "hello",
	})
	if err != nil || result.IsError {
		t.Fatalf("personal file tool was denied: result=%#v err=%v", result, err)
	}
	result, err = (ChatService{}).CallMCPTool(ctx, builtinChatServerName, "command", map[string]any{"command": "true"})
	if err != nil || result.IsError {
		t.Fatalf("command call was rejected: result=%#v err=%v", result, err)
	}
}
