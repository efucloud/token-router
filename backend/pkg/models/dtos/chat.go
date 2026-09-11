package dtos

import "time"

type ChatConversationCreate struct {
	Model string `json:"model" validate:"required,max=255" description:"对话使用的统一模型"`
}

type ChatConversationUpdate struct {
	Title string `json:"title" validate:"max=255" description:"对话标题，留空表示保留"`
	Model string `json:"model" validate:"required,max=255" description:"对话使用的统一模型"`
}

type ChatConversationSummary struct {
	ID        string    `json:"id" description:"会话ID"`
	Title     string    `json:"title" description:"会话标题"`
	Model     string    `json:"model" description:"统一模型"`
	CreatedAt time.Time `json:"createdAt" description:"创建时间"`
	UpdatedAt time.Time `json:"updatedAt" description:"更新时间"`
}

type ChatMessageCreate struct {
	Role        string `json:"role" validate:"required,oneof=system user assistant" description:"消息角色"`
	Content     string `json:"content" validate:"required,max=1048576" description:"消息正文"`
	TotalTokens int64  `json:"totalTokens" validate:"gte=0" description:"该轮累计Token数"`
}

type ChatMessageDetail struct {
	ID             string    `json:"id" description:"消息ID"`
	ConversationID string    `json:"conversationId" description:"会话ID"`
	Role           string    `json:"role" description:"消息角色"`
	Content        string    `json:"content" description:"消息正文"`
	TotalTokens    int64     `json:"totalTokens" description:"该轮累计Token数"`
	Sequence       int       `json:"sequence" description:"会话内顺序"`
	CreatedAt      time.Time `json:"createdAt" description:"创建时间"`
}

type ChatConversationDetail struct {
	ChatConversationSummary
	Messages []ChatMessageDetail `json:"messages"`
}
