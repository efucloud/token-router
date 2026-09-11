package services

import (
	"context"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
)

func TestChatConversationLifecycleAndAccountIsolation(t *testing.T) {
	originalChat := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat.MCPServers = []config.ChatMCPConfig{{Name: "workspace", Enabled: true}}
	t.Cleanup(func() { config.ApplicationConfig.Chat = originalChat })
	db := dashboardTestDatabase(t)
	accounts := []daos.Account{
		{ID: "chat-user-1", Username: "chat-user-1", Enable: true},
		{ID: "chat-user-2", Username: "chat-user-2", Enable: true},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatalf("create accounts: %v", err)
	}
	service := ChatService{}
	user1 := context.WithValue(context.Background(), config.RequestUserId, accounts[0].ID)
	user2 := context.WithValue(context.Background(), config.RequestUserId, accounts[1].ID)

	conversation, errorData := service.Create(user1, dtos.ChatConversationCreate{Model: "model-a", MCPServers: []string{"workspace"}})
	if errorData.IsNotNil() {
		t.Fatalf("create conversation: %v", errorData.Err)
	}
	if conversation.Title != "" || conversation.Model != "model-a" || len(conversation.MCPServers) != 1 || conversation.MCPServers[0] != "workspace" {
		t.Fatalf("unexpected new conversation: %#v", conversation)
	}
	userMessage, errorData := service.AppendMessage(user1, conversation.ID, dtos.ChatMessageCreate{
		Role: "user", Content: "请帮我分析这段很长的测试内容，并给出可靠结论。",
	})
	if errorData.IsNotNil() || userMessage.Sequence != 1 {
		t.Fatalf("append user message: message=%#v error=%v", userMessage, errorData.Err)
	}
	assistantMessage, errorData := service.AppendMessage(user1, conversation.ID, dtos.ChatMessageCreate{
		Role: "assistant", Content: "这是分析结论。", TotalTokens: 12,
	})
	if errorData.IsNotNil() || assistantMessage.Sequence != 2 {
		t.Fatalf("append assistant message: message=%#v error=%v", assistantMessage, errorData.Err)
	}
	detail, errorData := service.Get(user1, conversation.ID)
	if errorData.IsNotNil() {
		t.Fatalf("get conversation: %v", errorData.Err)
	}
	if detail.Title == "" || len(detail.Messages) != 2 || detail.Messages[1].TotalTokens != 12 || len(detail.MCPServers) != 1 {
		t.Fatalf("conversation was not persisted: %#v", detail)
	}
	if _, errorData = service.Get(user2, conversation.ID); !errorData.IsNotNil() || errorData.ResponseCode != 404 {
		t.Fatalf("another account accessed the conversation: %#v", errorData)
	}
	list, errorData := service.List(user1)
	if errorData.IsNotNil() || len(list) != 1 || list[0].ID != conversation.ID {
		t.Fatalf("list conversations: list=%#v error=%v", list, errorData.Err)
	}
	if errorData = service.Delete(user1, conversation.ID); errorData.IsNotNil() {
		t.Fatalf("delete conversation: %v", errorData.Err)
	}
	var messageCount int64
	if err := db.Model(&daos.ChatMessage{}).Where("conversation_id = ?", conversation.ID).Count(&messageCount).Error; err != nil {
		t.Fatalf("count deleted messages: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("conversation messages were not deleted: %d", messageCount)
	}
}

func TestChatTitleIsNormalizedAndBounded(t *testing.T) {
	title := chatTitle("  第一行\n\n第二行  " + "这是为了验证标题长度不会无限增长的一段额外文本这是为了验证标题长度不会无限增长的一段额外文本")
	if title == "" || len([]rune(title)) > 49 {
		t.Fatalf("unexpected generated title: %q", title)
	}
}

func TestLegacyConversationUsesDefaultToolsUntilUserChangesSelection(t *testing.T) {
	originalChat := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, DefaultEnabled: true, AllowedRoles: []string{"*"}, WorkspaceDirectory: t.TempDir(),
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = originalChat })

	db := dashboardTestDatabase(t)
	account := daos.Account{ID: "legacy-chat-user", Username: "legacy-chat-user", Enable: true}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	legacy := daos.ChatConversation{
		GatewayRecord: daos.GatewayRecord{ID: "legacy-conversation"},
		AccountID:     account.ID, Model: "model-a", Skills: "[]", MCPServers: "[]",
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("create legacy conversation: %v", err)
	}

	ctx := context.WithValue(context.Background(), config.RequestUserId, account.ID)
	service := ChatService{}
	detail, errorData := service.Get(ctx, legacy.ID)
	if errorData.IsNotNil() || len(detail.MCPServers) != 1 || detail.MCPServers[0] != builtinChatServerName {
		t.Fatalf("legacy conversation did not receive default tools: detail=%#v error=%v", detail, errorData.Err)
	}

	updated, errorData := service.Update(ctx, legacy.ID, dtos.ChatConversationUpdate{Model: legacy.Model, MCPServers: []string{}})
	if errorData.IsNotNil() || len(updated.MCPServers) != 0 {
		t.Fatalf("explicit tool disable was not preserved: conversation=%#v error=%v", updated, errorData.Err)
	}
	var stored daos.ChatConversation
	if err := db.First(&stored, "id = ?", legacy.ID).Error; err != nil {
		t.Fatalf("read updated conversation: %v", err)
	}
	if !stored.MCPServersInitialized {
		t.Fatal("tool selection was not marked as initialized")
	}
	detail, errorData = service.Get(ctx, legacy.ID)
	if errorData.IsNotNil() || len(detail.MCPServers) != 0 {
		t.Fatalf("default tools were reapplied after explicit disable: detail=%#v error=%v", detail, errorData.Err)
	}
}
