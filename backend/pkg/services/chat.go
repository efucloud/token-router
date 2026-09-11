package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/utils"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChatService struct{}

func chatError(err error, status int) common.ErrorData {
	return common.ErrorData{Err: err, ResponseCode: status, MsgCode: config.MsgCodeRequestDataInvalid}
}

func chatAccountID(ctx context.Context) (string, error) {
	accountID := config.GetOperatorFromCtx(ctx)
	if accountID == "" || accountID == "unknown" {
		return "", errors.New("authenticated account is missing")
	}
	return accountID, nil
}

func defaultChatMCPServers(ctx context.Context) []string {
	settings := config.ApplicationConfig.Chat.BuiltinTools
	if settings.Enabled && settings.DefaultEnabled {
		return []string{builtinChatServerName}
	}
	return []string{}
}

func chatConversationMCPServers(model daos.ChatConversation, defaults []string) []string {
	selected := chatSelection(model.MCPServers)
	if model.MCPServersInitialized {
		return selected
	}
	seen := make(map[string]struct{}, len(selected))
	for _, name := range selected {
		seen[name] = struct{}{}
	}
	for _, name := range defaults {
		if _, exists := seen[name]; !exists {
			selected = append(selected, name)
		}
	}
	return selected
}

func chatConversationSummary(model daos.ChatConversation, defaultMCPServers []string) dtos.ChatConversationSummary {
	return dtos.ChatConversationSummary{
		ID: model.ID, Title: model.Title, Model: model.Model,
		Skills: chatSelection(model.Skills), MCPServers: chatConversationMCPServers(model, defaultMCPServers),
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func chatSelection(value string) []string {
	result := make([]string, 0)
	if json.Unmarshal([]byte(value), &result) != nil {
		return []string{}
	}
	return result
}

func encodeChatSelection(value []string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func normalizeChatSelections(ctx context.Context, skills, mcpServers []string) ([]string, []string, error) {
	availableSkills, availableMCP := chatCapabilityNames(ctx)
	normalize := func(kind string, values []string, available map[string]struct{}) ([]string, error) {
		result := make([]string, 0, len(values))
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			name := strings.TrimSpace(value)
			if _, exists := seen[name]; exists {
				return nil, fmt.Errorf("duplicate %s %q", kind, name)
			}
			if _, exists := available[name]; !exists {
				return nil, fmt.Errorf("unknown %s %q", kind, name)
			}
			seen[name] = struct{}{}
			result = append(result, name)
		}
		return result, nil
	}
	normalizedSkills, err := normalize("skill", skills, availableSkills)
	if err != nil {
		return nil, nil, err
	}
	normalizedMCP, err := normalize("MCP server", mcpServers, availableMCP)
	return normalizedSkills, normalizedMCP, err
}

func chatMessageDetail(model daos.ChatMessage) dtos.ChatMessageDetail {
	return dtos.ChatMessageDetail{
		ID: model.ID, ConversationID: model.ConversationID, Role: model.Role,
		Content: model.Content, TotalTokens: model.TotalTokens,
		Sequence: model.Sequence, CreatedAt: model.CreatedAt,
	}
}

func chatTitle(content string) string {
	normalized := strings.Join(strings.Fields(content), " ")
	runes := []rune(normalized)
	if len(runes) <= 48 {
		return normalized
	}
	return string(runes[:48]) + "…"
}

func (ChatService) List(ctx context.Context) ([]dtos.ChatConversationSummary, common.ErrorData) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return nil, chatError(err, http.StatusUnauthorized)
	}
	var conversations []daos.ChatConversation
	if err = config.DBConnect.WithContext(ctx).Where("account_id = ?", accountID).
		Order("updated_at DESC").Limit(200).Find(&conversations).Error; err != nil {
		return nil, chatError(err, http.StatusInternalServerError)
	}
	result := make([]dtos.ChatConversationSummary, 0, len(conversations))
	defaultMCPServers := defaultChatMCPServers(ctx)
	for _, conversation := range conversations {
		result = append(result, chatConversationSummary(conversation, defaultMCPServers))
	}
	return result, common.ErrorData{}
}

func (ChatService) Create(ctx context.Context, input dtos.ChatConversationCreate) (dtos.ChatConversationSummary, common.ErrorData) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusUnauthorized)
	}
	input.Model = strings.TrimSpace(input.Model)
	input.Skills, input.MCPServers, err = normalizeChatSelections(ctx, input.Skills, input.MCPServers)
	if err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusBadRequest)
	}
	if err = validator.New().Struct(input); err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusBadRequest)
	}
	conversation := daos.ChatConversation{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
		AccountID:     accountID, Model: input.Model, Skills: encodeChatSelection(input.Skills),
		MCPServers: encodeChatSelection(input.MCPServers), MCPServersInitialized: true,
	}
	if err = config.DBConnect.WithContext(ctx).Create(&conversation).Error; err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusInternalServerError)
	}
	return chatConversationSummary(conversation, nil), common.ErrorData{}
}

func (ChatService) Get(ctx context.Context, id string) (dtos.ChatConversationDetail, common.ErrorData) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return dtos.ChatConversationDetail{}, chatError(err, http.StatusUnauthorized)
	}
	var conversation daos.ChatConversation
	if err = config.DBConnect.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).First(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtos.ChatConversationDetail{}, chatError(err, http.StatusNotFound)
		}
		return dtos.ChatConversationDetail{}, chatError(err, http.StatusInternalServerError)
	}
	var messages []daos.ChatMessage
	if err = config.DBConnect.WithContext(ctx).Where("conversation_id = ? AND account_id = ?", id, accountID).
		Order("sequence ASC").Find(&messages).Error; err != nil {
		return dtos.ChatConversationDetail{}, chatError(err, http.StatusInternalServerError)
	}
	result := dtos.ChatConversationDetail{ChatConversationSummary: chatConversationSummary(conversation, defaultChatMCPServers(ctx)), Messages: make([]dtos.ChatMessageDetail, 0, len(messages))}
	for _, message := range messages {
		result.Messages = append(result.Messages, chatMessageDetail(message))
	}
	return result, common.ErrorData{}
}

func (ChatService) Update(ctx context.Context, id string, input dtos.ChatConversationUpdate) (dtos.ChatConversationSummary, common.ErrorData) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusUnauthorized)
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Model = strings.TrimSpace(input.Model)
	input.Skills, input.MCPServers, err = normalizeChatSelections(ctx, input.Skills, input.MCPServers)
	if err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusBadRequest)
	}
	if err = validator.New().Struct(input); err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusBadRequest)
	}
	updates := map[string]any{
		"model": input.Model, "skills": encodeChatSelection(input.Skills),
		"mcp_servers": encodeChatSelection(input.MCPServers), "mcp_servers_initialized": true, "updated_at": time.Now(),
	}
	if input.Title != "" {
		updates["title"] = input.Title
	}
	result := config.DBConnect.WithContext(ctx).Model(&daos.ChatConversation{}).
		Where("id = ? AND account_id = ?", id, accountID).Updates(updates)
	if result.Error != nil {
		return dtos.ChatConversationSummary{}, chatError(result.Error, http.StatusInternalServerError)
	}
	if result.RowsAffected == 0 {
		return dtos.ChatConversationSummary{}, chatError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	var conversation daos.ChatConversation
	if err = config.DBConnect.WithContext(ctx).Where("id = ? AND account_id = ?", id, accountID).First(&conversation).Error; err != nil {
		return dtos.ChatConversationSummary{}, chatError(err, http.StatusInternalServerError)
	}
	return chatConversationSummary(conversation, nil), common.ErrorData{}
}

func (ChatService) AppendMessage(ctx context.Context, conversationID string, input dtos.ChatMessageCreate) (dtos.ChatMessageDetail, common.ErrorData) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return dtos.ChatMessageDetail{}, chatError(err, http.StatusUnauthorized)
	}
	input.Content = strings.TrimSpace(input.Content)
	if err = validator.New().Struct(input); err != nil {
		return dtos.ChatMessageDetail{}, chatError(err, http.StatusBadRequest)
	}
	var message daos.ChatMessage
	err = config.DBConnect.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation daos.ChatConversation
		if txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND account_id = ?", conversationID, accountID).First(&conversation).Error; txErr != nil {
			return txErr
		}
		var sequence int
		if txErr := tx.Model(&daos.ChatMessage{}).Where("conversation_id = ?", conversationID).
			Select("COALESCE(MAX(sequence), 0)").Scan(&sequence).Error; txErr != nil {
			return txErr
		}
		message = daos.ChatMessage{
			GatewayRecord:  daos.GatewayRecord{ID: utils.GenerateDatabaseId()},
			ConversationID: conversationID, AccountID: accountID,
			Role: input.Role, Content: input.Content, TotalTokens: input.TotalTokens,
			Sequence: sequence + 1,
		}
		if txErr := tx.Create(&message).Error; txErr != nil {
			return txErr
		}
		updates := map[string]any{"updated_at": time.Now()}
		if conversation.Title == "" && input.Role == "user" {
			updates["title"] = chatTitle(input.Content)
		}
		return tx.Model(&daos.ChatConversation{}).Where("id = ?", conversationID).Updates(updates).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dtos.ChatMessageDetail{}, chatError(err, http.StatusNotFound)
		}
		return dtos.ChatMessageDetail{}, chatError(err, http.StatusInternalServerError)
	}
	return chatMessageDetail(message), common.ErrorData{}
}

func (ChatService) Delete(ctx context.Context, id string) common.ErrorData {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return chatError(err, http.StatusUnauthorized)
	}
	err = config.DBConnect.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND account_id = ?", id, accountID).Delete(&daos.ChatConversation{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("conversation_id = ? AND account_id = ?", id, accountID).Delete(&daos.ChatMessage{}).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return chatError(err, http.StatusNotFound)
		}
		return chatError(err, http.StatusInternalServerError)
	}
	return common.ErrorData{}
}
