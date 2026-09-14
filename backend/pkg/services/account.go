package services

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/repositories"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type AccountService struct {
	repo repositories.AccountRepository
}

func (svc *AccountService) init(ctx context.Context) {
	db, ok := ctx.Value(config.ContextDBTx).(*gorm.DB)
	if ok {
		svc.repo = repositories.AccountRepository{DB: db}
	} else {
		svc.repo = repositories.AccountRepository{DB: config.DBConnect}
	}
}

func (svc *AccountService) GetAccountByUsername(ctx context.Context, name string) (result dtos.AccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	result, errorData = svc.repo.GetAccountByUsername(ctx, name)
	if errorData.IsNotNil() {
		config.Logger.Errorf("get account by username: %s failed, err: %s", name, errorData.Err.Error())
		return
	}
	return result, errorData
}

func (svc *AccountService) ListAccountIds(ctx context.Context, accountIds []string) (results dtos.AccountDetailList, errorData common.ErrorData) {
	svc.init(ctx)
	results, errorData = svc.repo.ListAccountIds(ctx, accountIds)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s list Account  failed, err: %s", config.GetOperatorFromCtx(ctx), errorData.Err.Error())
		return
	}
	return results, errorData
}
func (svc *AccountService) ListAccount(ctx context.Context, current, pageSize int, order, query string, queryArgs []interface{}) (results dtos.AccountDetailList, errorData common.ErrorData) {
	svc.init(ctx)
	results, errorData = svc.repo.ListAccount(ctx, current, pageSize, order, query, queryArgs)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s list Account  query: [%s] queryArgs: [%+v] failed, err: %s", config.GetOperatorFromCtx(ctx), query, queryArgs, errorData.Err.Error())
		return
	}
	return results, errorData
}
func (svc *AccountService) UpdateAccount(ctx context.Context, model dtos.AccountUpdate) (result dtos.AccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	model.Default(ctx)
	errorData.Err = model.Validate(ctx)
	if errorData.IsNotNil() {
		errorData.MsgCode = config.MsgCodeRequestDataInvalid
		config.Logger.Errorf("operator: %s update Account: %s failed, err: %s", config.GetOperatorFromCtx(ctx), model.Username, errorData.Err.Error())
		return
	}
	result, errorData = svc.repo.UpdateAccount(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s update Account: %s failed, err: %s", config.GetOperatorFromCtx(ctx), model.Username, errorData.Err.Error())
		return
	}
	return
}
func (svc *AccountService) CreateOrUpdateAccount(ctx context.Context, model dtos.AccountCreate) (result dtos.AccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	result, _ = svc.GetAccountByID(ctx, model.ID)
	if len(result.ID) == 0 {
		model.Default(ctx)
		errorData.Err = model.Validate(ctx)
		if errorData.IsNotNil() {
			errorData.MsgCode = config.MsgCodeRequestDataInvalid
			config.Logger.Errorf("operator: %s create Account: %s failed, err: %s", config.GetOperatorFromCtx(ctx), model.Username, errorData.Err.Error())
			return
		}
		result, errorData = svc.repo.AddAccount(ctx, model)
		if errorData.IsNotNil() {
			config.Logger.Errorf("operator: %s create Account: %s failed, err: %s", config.GetOperatorFromCtx(ctx), model.Username, errorData.Err.Error())
			return
		}
	} else {
		var update dtos.AccountUpdate
		data, _ := json.Marshal(model)
		_ = json.Unmarshal(data, &update)
		update.Email = result.Email
		result, _ = svc.UpdateAccount(ctx, update)
		svc.SetRole(ctx, dtos.AccountRole{Role: model.Role, Ids: []string{result.ID}})
	}
	return
}

// CreateAccountIfAbsent creates an account without changing an existing one.
// The second lookup makes concurrent first requests idempotent when another
// request inserts the same OIDC identity between the lookup and insert.
func (svc *AccountService) CreateAccountIfAbsent(ctx context.Context, model dtos.AccountCreate) (result dtos.AccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	result, errorData = svc.repo.GetAccountByID(ctx, model.ID)
	if errorData.IsNil() && result.ID != "" {
		return result, errorData
	}
	if errorData.ResponseCode != http.StatusNotFound {
		return result, errorData
	}

	model.Default(ctx)
	if errorData.Err = model.Validate(ctx); errorData.IsNotNil() {
		errorData.MsgCode = config.MsgCodeRequestDataInvalid
		return result, errorData
	}
	result, errorData = svc.repo.AddAccount(ctx, model)
	if errorData.IsNil() {
		return result, errorData
	}

	// A concurrent request may have won the insert race. In that case the
	// desired account now exists and is safe to return.
	existing, lookupError := svc.repo.GetAccountByID(ctx, model.ID)
	if lookupError.IsNil() && existing.ID != "" {
		return existing, lookupError
	}
	return result, errorData
}

func (svc *AccountService) AddAccount(ctx context.Context, model dtos.AccountCreate) (result dtos.AccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	model.Default(ctx)
	errorData.Err = model.Validate(ctx)
	if errorData.IsNotNil() {
		errorData.MsgCode = config.MsgCodeRequestDataInvalid
		config.Logger.Errorf("operator: %s create Account: %s failed, err: %s", config.GetOperatorFromCtx(ctx), model.Username, errorData.Err.Error())
		return
	}
	result, errorData = svc.repo.AddAccount(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s create Account: %s failed, err: %s", config.GetOperatorFromCtx(ctx), model.Username, errorData.Err.Error())
		return
	}
	return
}
func (svc *AccountService) DeleteAccount(ctx context.Context, ids []string) (errorData common.ErrorData) {
	svc.init(ctx)
	errorData = svc.repo.DeleteAccount(ctx, ids)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s delete Account by ids: %v failed, err: %s", config.GetOperatorFromCtx(ctx), ids, errorData.Err.Error())
		return
	}
	return
}
func (svc *AccountService) SetRole(ctx context.Context, model dtos.AccountRole) (errorData common.ErrorData) {
	svc.init(ctx)
	model.Default(ctx)
	if errorData.Err = model.Validate(ctx); errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusBadRequest
		errorData.MsgCode = config.MsgCodeRequestDataInvalid
		return
	}
	errorData = svc.repo.SetOrgRole(ctx, model)
	return
}

func (svc *AccountService) ChangeStatusAccount(ctx context.Context, model dtos.AccountStatus) (errorData common.ErrorData) {
	svc.init(ctx)
	model.Default(ctx)
	if errorData.Err = model.Validate(ctx); errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusBadRequest
		errorData.MsgCode = config.MsgCodeRequestDataInvalid
		return
	}
	errorData = svc.repo.ChangeStatusAccount(ctx, model)
	return
}

func (svc *AccountService) UpdateQuota(ctx context.Context, id string, model dtos.AccountQuotaUpdate) (errorData common.ErrorData) {
	svc.init(ctx)
	if err := validator.New().Struct(model); err != nil {
		return common.ErrorData{Err: err, ResponseCode: http.StatusBadRequest, MsgCode: config.MsgCodeRequestDataInvalid}
	}
	return svc.repo.UpdateQuota(ctx, id, model)
}
func (svc *AccountService) GetSimpleAccountByID(ctx context.Context, id string) (result dtos.SimpleAccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	var account dtos.AccountDetail
	account, errorData = svc.repo.GetAccountByID(ctx, id)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s get Account by id : %s failed, err: %s", config.GetOperatorFromCtx(ctx), id, errorData.Err.Error())
		return
	}
	data, _ := json.Marshal(account)
	_ = json.Unmarshal(data, &result)
	return result, errorData
}
func (svc *AccountService) GetAccountByID(ctx context.Context, id string) (result dtos.AccountDetail, errorData common.ErrorData) {
	svc.init(ctx)
	result, errorData = svc.repo.GetAccountByID(ctx, id)
	if errorData.IsNotNil() {
		config.Logger.Errorf("operator: %s get Account by id : %s failed, err: %s", config.GetOperatorFromCtx(ctx), id, errorData.Err.Error())
		return
	}
	return result, errorData
}
func (svc *AccountService) UpdateAccountAvatar(ctx context.Context, userId string, avatarAddress string) (errorData common.ErrorData) {
	svc.init(ctx)
	return svc.repo.UpdateAccountAvatar(ctx, userId, avatarAddress)
}
