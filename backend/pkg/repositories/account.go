package repositories

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/efucloud/token-router/pkg/utils"

	"github.com/efucloud/token-router/pkg/models"
	"github.com/efucloud/token-router/pkg/models/daos"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"gorm.io/gorm"
)

type AccountRepository struct {
	DB *gorm.DB
}

func (repo *AccountRepository) GetAccountByUsername(ctx context.Context, username string) (result dtos.AccountDetail, errorData common.ErrorData) {

	//用户名，手机号码，邮箱，工号登录
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("username = ? OR ( phone = ? OR email = ? OR job_number = ? )",
		username, username, username, username).Find(&result).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeGetRecordFailed
	} else if len(result.ID) == 0 {
		errorData.MsgCode = config.MsgCodeGetRecordFailed
		errorData.ResponseCode = http.StatusNotFound
		errorData.Err = fmt.Errorf("not found account by username: %s", username)

	}
	return
}
func (repo *AccountRepository) SetOrgRole(ctx context.Context, model dtos.AccountRole) (errorData common.ErrorData) {
	columns := make(map[string]interface{})
	columns["role"] = model.Role
	columns["updated_at"] = model.UpdatedAt
	columns["updater_id"] = config.GetOperatorFromCtx(ctx)
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id IN (?)", model.Ids).
		UpdateColumns(columns).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeUpdateRecordFailed
	}
	return
}
func (repo *AccountRepository) ChangeStatusAccount(ctx context.Context, model dtos.AccountStatus) (errorData common.ErrorData) {
	columns := make(map[string]interface{})
	columns["enable"] = model.Enable
	columns["updated_at"] = model.UpdatedAt
	columns["updater_id"] = config.GetOperatorFromCtx(ctx)
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id IN (?)", model.Ids).
		UpdateColumns(columns).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeUpdateRecordFailed
	} else {
		//todo 多条更新
	}
	return
}

func (repo *AccountRepository) UpdateQuota(ctx context.Context, id string, model dtos.AccountQuotaUpdate) (errorData common.ErrorData) {
	result := repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id = ?", id).Updates(map[string]interface{}{
		"token_limit": model.TokenLimit, "request_limit": model.RequestLimit,
		"updated_at": time.Now(), "updater_id": config.GetOperatorFromCtx(ctx),
	})
	errorData.Err = result.Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeUpdateRecordFailed
	} else if result.RowsAffected == 0 {
		errorData.Err = gorm.ErrRecordNotFound
		errorData.ResponseCode = http.StatusNotFound
		errorData.MsgCode = config.MsgCodeGetRecordFailed
	}
	return
}

func (repo *AccountRepository) GetAccountsByIds(ctx context.Context, ids []string) (results dtos.AccountDetailList, errorData common.ErrorData) {
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id IN (?)", ids).Find(&results.Data).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeGetRecordFailed
	}
	return
}

func (repo *AccountRepository) UpdateAccountAvatar(ctx context.Context, userId string, avatarAddress string) (errorData common.ErrorData) {
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id = ?", userId).Update("avatar", avatarAddress).Error
	return
}
func (repo *AccountRepository) GetAccountByID(ctx context.Context, id string) (result dtos.AccountDetail, errorData common.ErrorData) {
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id = ?", id).Find(&result).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeGetRecordFailed
	} else if len(result.ID) == 0 {
		errorData.MsgCode = config.MsgCodeGetRecordFailed
		errorData.ResponseCode = http.StatusNotFound
		errorData.Err = fmt.Errorf("not found account by id: %s", id)

	}
	return
}
func (repo *AccountRepository) ListAccountIds(ctx context.Context, accountIds []string) (results dtos.AccountDetailList, errorData common.ErrorData) {
	db := repo.DB.WithContext(ctx).Table(models.AccountTableName)

	errorData.Err = db.Where("id IN (?)", accountIds).Find(&results.Data).Error

	return
}
func (repo *AccountRepository) ListAccount(ctx context.Context, current, pageSize int, order, query string, queryArgs []interface{}) (results dtos.AccountDetailList, errorData common.ErrorData) {
	db := repo.DB.WithContext(ctx).Table(models.AccountTableName)
	if len(query) > 0 {
		db = db.Where(query, queryArgs...)
	}
	errorData.Err = db.Where("deleted_at is Null").Count(&results.Total).Error
	if errorData.IsNil() {
		if len(order) > 0 {
			db = db.Order(order)
		}
		errorData.Err = db.Offset((current - 1) * pageSize).Limit(pageSize).Find(&results.Data).Error
	} else {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeGetRecordFailed
	}
	return
}
func (repo *AccountRepository) AddAccount(ctx context.Context, model dtos.AccountCreate) (result dtos.AccountDetail, errorData common.ErrorData) {

	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Create(&model).Scan(&result).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeCreateRecordFailed
	}
	return result, errorData
}

func (repo *AccountRepository) DeleteAccount(ctx context.Context, ids []string) (errorData common.ErrorData) {
	deleterId := config.GetOperatorFromCtx(ctx)
	delFlag := utils.GenerateDatabaseId()
	_ = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id IN (?) AND del_flag = ?", ids, "active").UpdateColumns(map[string]interface{}{"deleter_id": deleterId, "del_flag": delFlag})
	errorData.Err = repo.DB.WithContext(ctx).Where("id IN (?) AND del_flag = ?", ids, delFlag).Delete(&daos.Account{}).Error
	if errorData.IsNotNil() {
		errorData = daos.ParserDatabaseError(ctx, errorData)
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeDeleteRecordFailed
	}
	return
}
func (repo *AccountRepository) UpdateAccount(ctx context.Context, model dtos.AccountUpdate) (result dtos.AccountDetail, errorData common.ErrorData) {
	errorData.Err = repo.DB.WithContext(ctx).Table(models.AccountTableName).Where("id = ?", model.ID).Save(&model).Find(&result).Error
	if errorData.IsNotNil() {
		errorData.ResponseCode = http.StatusInternalServerError
		errorData.MsgCode = config.MsgCodeUpdateRecordFailed
	}
	return result, errorData
}
