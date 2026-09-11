package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
)

type APITokenService struct{}

func apiTokenError(err error, status int) common.ErrorData {
	return common.ErrorData{Err: err, ResponseCode: status, MsgCode: config.MsgCodeRequestDataInvalid}
}

func apiTokenAccountID(ctx context.Context) (string, error) {
	accountID := config.GetOperatorFromCtx(ctx)
	if accountID == "" || accountID == "unknown" {
		return "", errors.New("authenticated account is missing")
	}
	return accountID, nil
}

func apiTokenDetail(model daos.APIToken) dtos.APITokenDetail {
	allowed := []string{}
	_ = json.Unmarshal([]byte(model.AllowedModels), &allowed)
	ipAllowlist := []string{}
	_ = json.Unmarshal([]byte(model.IPAllowlist), &ipAllowlist)
	return dtos.APITokenDetail{
		ID: model.ID, Name: model.Name, KeyPrefix: model.KeyPrefix, Status: model.Status,
		AllowedModels: allowed, TokenLimit: model.TokenLimit, RequestLimit: model.RequestLimit,
		IPAllowlist: ipAllowlist, RPM: model.RPM, TPM: model.TPM, MaxConcurrency: model.MaxConcurrency,
		UsedTokens: model.UsedTokens, UsedRequests: model.UsedRequests, ExpiresAt: model.ExpiresAt,
		CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
	}
}

func normalizedIPAllowlist(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if net.ParseIP(value) == nil {
			if _, _, err := net.ParseCIDR(value); err != nil {
				return nil, fmt.Errorf("invalid IP or CIDR: %s", value)
			}
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result, nil
}

func (APITokenService) List(ctx context.Context) (dtos.APITokenList, common.ErrorData) {
	accountID, err := apiTokenAccountID(ctx)
	if err != nil {
		return dtos.APITokenList{}, apiTokenError(err, http.StatusUnauthorized)
	}
	var models []daos.APIToken
	db := dashboardDB(ctx).Where("account_id = ?", accountID).Order("created_at DESC")
	if err = db.Find(&models).Error; err != nil {
		return dtos.APITokenList{}, apiTokenError(err, http.StatusInternalServerError)
	}
	result := dtos.APITokenList{Data: make([]dtos.APITokenDetail, 0, len(models)), Total: int64(len(models))}
	for _, model := range models {
		result.Data = append(result.Data, apiTokenDetail(model))
	}
	return result, common.ErrorData{}
}

func (APITokenService) Create(ctx context.Context, input dtos.APITokenCreate) (dtos.APITokenCreated, common.ErrorData) {
	accountID, err := apiTokenAccountID(ctx)
	if err != nil {
		return dtos.APITokenCreated{}, apiTokenError(err, http.StatusUnauthorized)
	}
	if err = validator.New().Struct(input); err != nil {
		return dtos.APITokenCreated{}, apiTokenError(err, http.StatusBadRequest)
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return dtos.APITokenCreated{}, apiTokenError(errors.New("expiresAt must be in the future"), http.StatusBadRequest)
	}
	plaintext, hash, prefix, err := utils.GenerateAPIToken()
	if err != nil {
		return dtos.APITokenCreated{}, apiTokenError(err, http.StatusInternalServerError)
	}
	allowlist, err := normalizedIPAllowlist(input.IPAllowlist)
	if err != nil {
		return dtos.APITokenCreated{}, apiTokenError(err, http.StatusBadRequest)
	}
	allowed, _ := json.Marshal(input.AllowedModels)
	allowedIPs, _ := json.Marshal(allowlist)
	model := daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, AccountID: accountID,
		Name: input.Name, KeyHash: hash, KeyPrefix: prefix, Status: "active", AllowedModels: string(allowed),
		TokenLimit: input.TokenLimit, RequestLimit: input.RequestLimit, ExpiresAt: input.ExpiresAt,
		IPAllowlist: string(allowedIPs), RPM: input.RPM, TPM: input.TPM, MaxConcurrency: input.MaxConcurrency,
	}
	if err = dashboardDB(ctx).Create(&model).Error; err != nil {
		return dtos.APITokenCreated{}, apiTokenError(err, http.StatusInternalServerError)
	}
	return dtos.APITokenCreated{APITokenDetail: apiTokenDetail(model), Token: plaintext}, common.ErrorData{}
}

func (APITokenService) Update(ctx context.Context, id string, input dtos.APITokenUpdate) (dtos.APITokenDetail, common.ErrorData) {
	accountID, err := apiTokenAccountID(ctx)
	if err != nil {
		return dtos.APITokenDetail{}, apiTokenError(err, http.StatusUnauthorized)
	}
	if err = validator.New().Struct(input); err != nil {
		return dtos.APITokenDetail{}, apiTokenError(err, http.StatusBadRequest)
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now()) {
		return dtos.APITokenDetail{}, apiTokenError(errors.New("expiresAt must be in the future"), http.StatusBadRequest)
	}
	allowlist, err := normalizedIPAllowlist(input.IPAllowlist)
	if err != nil {
		return dtos.APITokenDetail{}, apiTokenError(err, http.StatusBadRequest)
	}
	allowed, _ := json.Marshal(input.AllowedModels)
	allowedIPs, _ := json.Marshal(allowlist)
	db := dashboardDB(ctx)
	result := db.Model(&daos.APIToken{}).Where("id = ? AND account_id = ?", id, accountID).Updates(map[string]any{
		"name": input.Name, "allowed_models": string(allowed), "token_limit": input.TokenLimit,
		"request_limit": input.RequestLimit, "expires_at": input.ExpiresAt, "status": input.Status,
		"ip_allowlist": string(allowedIPs), "rpm": input.RPM, "tpm": input.TPM,
		"max_concurrency": input.MaxConcurrency,
	})
	if result.Error != nil {
		return dtos.APITokenDetail{}, apiTokenError(result.Error, http.StatusInternalServerError)
	}
	if result.RowsAffected == 0 {
		return dtos.APITokenDetail{}, apiTokenError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	var model daos.APIToken
	if err = db.Where("id = ? AND account_id = ?", id, accountID).First(&model).Error; err != nil {
		return dtos.APITokenDetail{}, apiTokenError(err, http.StatusInternalServerError)
	}
	return apiTokenDetail(model), common.ErrorData{}
}

func (APITokenService) Delete(ctx context.Context, id string) common.ErrorData {
	accountID, err := apiTokenAccountID(ctx)
	if err != nil {
		return apiTokenError(err, http.StatusUnauthorized)
	}
	result := dashboardDB(ctx).Where("id = ? AND account_id = ?", id, accountID).Delete(&daos.APIToken{})
	if result.Error != nil {
		return apiTokenError(result.Error, http.StatusInternalServerError)
	}
	if result.RowsAffected == 0 {
		return apiTokenError(gorm.ErrRecordNotFound, http.StatusNotFound)
	}
	return common.ErrorData{}
}
