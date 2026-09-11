package daos

import "github.com/efucloud/token-router/pkg/models"

type ChatConversation struct {
	GatewayRecord
	AccountID  string `gorm:"column:account_id;type:varchar(50);not null" json:"accountId"`
	Title      string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Model      string `gorm:"column:model;type:varchar(255);not null" json:"model"`
	Skills     string `gorm:"column:skills;type:longtext;not null;default:'[]'" json:"skills"`
	MCPServers string `gorm:"column:mcp_servers;type:longtext;not null;default:'[]'" json:"mcpServers"`
}

func (*ChatConversation) TableName() string { return models.ChatConversationTableName }

func (*ChatConversation) Indexes() map[string][]string {
	return map[string][]string{"idx_chat_conversation_account_updated": {"account_id", "updated_at"}}
}

func (*ChatConversation) UniqueIndexes() map[string][]string { return map[string][]string{} }

type ChatMessage struct {
	GatewayRecord
	ConversationID string `gorm:"column:conversation_id;type:varchar(50);not null" json:"conversationId"`
	AccountID      string `gorm:"column:account_id;type:varchar(50);not null" json:"accountId"`
	Role           string `gorm:"column:role;type:varchar(20);not null" json:"role"`
	Content        string `gorm:"column:content;type:longtext;not null" json:"content"`
	TotalTokens    int64  `gorm:"column:total_tokens;default:0;not null" json:"totalTokens"`
	Sequence       int    `gorm:"column:sequence;not null" json:"sequence"`
}

func (*ChatMessage) TableName() string { return models.ChatMessageTableName }

func (*ChatMessage) Indexes() map[string][]string {
	return map[string][]string{
		"idx_chat_message_conversation": {"conversation_id", "sequence"},
		"idx_chat_message_account":      {"account_id", "created_at"},
	}
}

func (*ChatMessage) UniqueIndexes() map[string][]string {
	return map[string][]string{"uniq_idx_chat_message_sequence": {"conversation_id", "sequence"}}
}
