package dtos

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// AccountDetailList  账户列表响应
type AccountDetailList struct {
	//当前页数据
	Data []*AccountDetail `json:"data"`
	//数据库满足条件的数据总数
	Total int64 `json:"total,omitempty" validate:"required"`
}

// AccountQuotaUpdate 更新用户聚合限额；0 表示不限制。
type AccountQuotaUpdate struct {
	TokenLimit   int64 `json:"tokenLimit" validate:"gte=0" description:"Token总量限额，0表示不限制"`
	RequestLimit int64 `json:"requestLimit" validate:"gte=0" description:"成功请求次数限额，0表示不限制"`
}

type SimpleAccountDetail struct {
	//主键
	ID string `gorm:"primarykey;column:id;type:varchar(50)" json:"id" description:"记录ID"`
	//用户名
	Username string `gorm:"type:varchar(255);column:username" json:"username" validate:"alphanum" description:"用户名"`
	//昵称，如中文名
	Nickname string `gorm:"type:varchar(255);column:nickname" json:"nickname" validate:"max=255" description:"昵称"`
	//工号
	JobNumber string `gorm:"type:varchar(255);column:job_number" json:"jobNumber" description:"工号"`
	//系统角色
	Role string `gorm:"type:varchar(255);column:role;default:none" json:"role" validate:"oneof=admin view edit none" enum:"admin|view|edit|none" description:"系统角色"`
	//是否有效
	Enable bool `gorm:"column:enable;default:true" json:"enable" description:"是否有效"`
	//账户类型，企业用户时role才可以不为none
	Category string `gorm:"type:varchar(255)" json:"category" validate:"oneof=enterprise customer provider customer_and_provider consumer" enum:"enterprise|customer|provider|customer_and_provider|consumer" description:"账户类型"`
	//邮箱
	Email string `gorm:"type:varchar(255);column:email" json:"email" validate:"email" description:"邮箱"`
	//手机号码
	Phone string `gorm:"type:varchar(255);column:phone" json:"phone" validate:"required" description:"电话"`
	//默认语言
	Language string `gorm:"type:varchar(255);column:language" json:"language" validate:"oneof=zh en" enum:"zh|en" description:"默认语言"`
	//头像
	Avatar string `gorm:"type:varchar(1000);column:avatar" json:"avatar" description:"头像"`
	//Token 总量限额，0 表示不限制
	TokenLimit int64 `gorm:"column:token_limit" json:"tokenLimit" description:"Token总量限额"`
	//成功请求次数限额，0 表示不限制
	RequestLimit int64 `gorm:"column:request_limit" json:"requestLimit" description:"成功请求次数限额"`
	//已使用 Token 数
	UsedTokens int64 `gorm:"column:used_tokens" json:"usedTokens" description:"已使用Token数"`
	//已使用成功请求数
	UsedRequests int64 `gorm:"column:used_requests" json:"usedRequests" description:"已使用成功请求数"`
}

func (md *SimpleAccountDetail) TableName() string {
	return models.AccountTableName
}

// AccountDetail 账户详情
type AccountDetail struct {
	//主键
	ID string `gorm:"type:varchar(50);column:id" json:"id" validate:"required" description:"记录ID"`
	//创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at;<-:create" json:"createdAt" description:"创建时间"`
	//更新时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updatedAt" description:"更新时间"`
	//创建者
	CreatorId string `gorm:"type:varchar(50);column:creator_id;<-:create" validate:"required" json:"creatorId" description:"创建者"`
	//更新者
	UpdaterId string `gorm:"type:varchar(50);column:updater_id;<-:update" json:"updaterId" validate:"required" description:"更新者"`
	//软删除
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"deletedAt,omitempty"`
	//删除状态
	DelFlag string `gorm:"type:varchar(50);column:del_flag;default:active" json:"-" description:"删除状态"`
	//用户名
	Username string `gorm:"type:varchar(255);column:username" json:"username" validate:"alphanum" description:"用户名"`
	//昵称，如中文名
	Nickname string `gorm:"type:varchar(255);column:nickname" json:"nickname" validate:"max=255" description:"昵称"`
	//工号
	JobNumber string `gorm:"type:varchar(255);column:job_number" json:"jobNumber" description:"工号"`
	//系统角色
	Role string `gorm:"type:varchar(255);column:role;default:none" json:"role" validate:"oneof=admin view edit none" enum:"admin|view|edit|none" description:"系统角色"`
	//是否有效
	Enable bool `gorm:"column:enable;default:true" json:"enable" description:"是否有效"`
	//邮箱
	Email string `gorm:"type:varchar(255);column:email" json:"email" validate:"email" description:"邮箱"`
	//手机号码
	Phone string `gorm:"type:varchar(255);column:phone" json:"phone" validate:"required" description:"电话"`
	//默认语言
	Language string `gorm:"type:varchar(255);column:language" json:"language" validate:"oneof=zh en" enum:"zh|en" description:"默认语言"`
	//头像
	Avatar string `gorm:"type:varchar(1000);column:avatar" json:"avatar" description:"头像"`
	//Token 总量限额，0 表示不限制
	TokenLimit int64 `gorm:"column:token_limit" json:"tokenLimit" description:"Token总量限额"`
	//成功请求次数限额，0 表示不限制
	RequestLimit int64 `gorm:"column:request_limit" json:"requestLimit" description:"成功请求次数限额"`
	//已使用 Token 数
	UsedTokens int64 `gorm:"column:used_tokens" json:"usedTokens" description:"已使用Token数"`
	//已使用成功请求数
	UsedRequests int64 `gorm:"column:used_requests" json:"usedRequests" description:"已使用成功请求数"`
}

func (md *AccountDetail) TableName() string {
	return models.AccountTableName
}

// AccountCreate 账户信息创建
// 未来账户信息修改只能从eauth中
type AccountCreate struct {
	//记录ID
	ID string `gorm:"type:varchar(50);column:id" json:"id" validate:"required" description:"记录ID"`
	//创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at;<-:create" json:"-" description:"创建时间"`
	//创建者
	CreatorId string `gorm:"type:varchar(50);column:creator_id;<-:create" validate:"required" json:"-" description:"创建者"`
	//用户名
	Username string `gorm:"type:varchar(255)" json:"username" validate:"required,max=255" description:"用户名"`
	//昵称，如中文名
	Nickname string `gorm:"type:varchar(255)" json:"nickname" validate:"max=255" description:"昵称"`
	//组织角色
	Role string `gorm:"type:varchar(255);default:none" json:"role" validate:"oneof=admin view edit none" enum:"admin|view|edit|none" description:"组织角色"`
	//是否有效
	Enable bool `gorm:"column:enable;default:true" json:"enable" description:"是否有效"`
	//邮箱
	Email string `gorm:"type:varchar(255)" json:"email" validate:"max=255" description:"邮箱"`
	//手机号码
	Phone string `gorm:"type:varchar(255)" json:"phone" validate:"max=50" description:"电话"`
	//默认语言
	Language string `gorm:"type:varchar(255);column:language;default:zh" json:"language" validate:"oneof=zh en-US" enum:"zh|en-US" description:"默认语言"`
	//头像
	Avatar string `gorm:"type:varchar(1000);column:avatar" json:"-" description:"头像"`
}

func (md *AccountCreate) Default(ctx context.Context) {
	md.CreatorId = config.GetOperatorFromCtx(ctx)
	md.CreatedAt = time.Now()
	if len(md.Language) == 0 {
		md.Language = "zh"
	}
}
func (md *AccountCreate) Validate(ctx context.Context) (err error) {
	validate := validator.New()
	lang := common.GetLangFromCtx(ctx, "")
	validate.RegisterTagNameFunc(common.TagNameI18N(lang))
	trans := common.LoadValidateTranslator(lang, validate)
	err = validate.Struct(md)
	if err != nil {
		var lines []string
		var errs validator.ValidationErrors
		errors.As(err, &errs)
		for _, v := range errs.Translate(trans) {
			lines = append(lines, v)
		}
		if len(lines) > 0 {
			err = errors.New(strings.Join(lines, "\n"))
		}
	}
	return
}

// AccountStatus 账户信息禁用/启用
// 账户禁用后，用户将不能登陆该系统
type AccountStatus struct {
	//主键
	Ids []string `json:"ids" validate:"required" description:"主键"`
	//更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at;<-:update" json:"-" description:"更新时间"`
	//更新者
	UpdaterId string `gorm:"type:varchar(50);column:updater_id;<-:update" json:"-" validate:"required" description:"更新者"`
	//是否有效
	Enable bool `gorm:"column:enable;default:true" json:"enable" description:"是否有效"`
}

func (md *AccountStatus) Default(ctx context.Context) {
	md.UpdaterId = config.GetOperatorFromCtx(ctx)
	md.UpdatedAt = time.Now()
}
func (md *AccountStatus) Validate(ctx context.Context) (err error) {
	validate := validator.New()
	lang := common.GetLangFromCtx(ctx, "")
	validate.RegisterTagNameFunc(common.TagNameI18N(lang))
	trans := common.LoadValidateTranslator(lang, validate)
	err = validate.Struct(md)
	if err != nil {
		var lines []string
		var errs validator.ValidationErrors
		errors.As(err, &errs)
		for _, v := range errs.Translate(trans) {
			lines = append(lines, v)
		}
		if len(lines) > 0 {
			err = errors.New(strings.Join(lines, "\n"))
		}
	}
	return
}

// AccountRole 账户系统角色设置
// 设置账户在系统中的角色
type AccountRole struct {
	//主键
	Ids []string `json:"ids" validate:"required" description:"主键"`
	//更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at;<-:update" json:"-" description:"更新时间"`
	//更新者
	UpdaterId string `gorm:"type:varchar(50);column:updater_id;<-:update" json:"-" validate:"required" description:"更新者"`
	//角色
	Role string `gorm:"type:varchar(255);default:none;column:role" json:"role" validate:"oneof=admin view edit none" enum:"admin|view|edit|none" description:"组织角色"`
}

func (md *AccountRole) Default(ctx context.Context) {
	md.UpdaterId = config.GetOperatorFromCtx(ctx)
	md.UpdatedAt = time.Now()
}
func (md *AccountRole) Validate(ctx context.Context) (err error) {
	validate := validator.New()
	lang := common.GetLangFromCtx(ctx, "")
	validate.RegisterTagNameFunc(common.TagNameI18N(lang))
	trans := common.LoadValidateTranslator(lang, validate)
	err = validate.Struct(md)
	if err != nil {
		var lines []string
		var errs validator.ValidationErrors
		errors.As(err, &errs)
		for _, v := range errs.Translate(trans) {
			lines = append(lines, v)
		}
		if len(lines) > 0 {
			err = errors.New(strings.Join(lines, "\n"))
		}
	}
	return
}

// AccountUpdate 账户信息更新
// 更新账户信息，未来只能在eauth中更新
type AccountUpdate struct {
	//主键
	ID string `gorm:"type:varchar(50);column:id" json:"id" validate:"required" description:"记录ID"`
	//更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at;<-:update" json:"-" description:"更新时间"`
	//更新者
	UpdaterId string `gorm:"type:varchar(50);column:updater_id;<-:update" json:"-" validate:"required" description:"更新者"`
	//用户名
	Username string `gorm:"type:varchar(255)" json:"username" validate:"required,max=255" description:"用户名"`
	//昵称，如中文名
	Nickname string `gorm:"type:varchar(255)" json:"nickname" validate:"max=255" description:"昵称"`
	//邮箱
	Email string `gorm:"type:varchar(255)" json:"email" validate:"max=255" description:"邮箱"`
	//手机号码
	Phone string `gorm:"type:varchar(255)" json:"phone" validate:"max=50" description:"电话"`
	//默认语言
	Language string `gorm:"type:varchar(255);column:language;default:zh" json:"language" validate:"oneof=zh en-US" enum:"zh|en-US" description:"默认语言"`
	//头像
	Avatar string `gorm:"type:varchar(1000);column:avatar" json:"-" description:"头像"`
}

func (md *AccountUpdate) Default(ctx context.Context) {
	md.UpdaterId = config.GetOperatorFromCtx(ctx)
	if len(md.Language) == 0 {
		md.Language = "zh"
	}
}
func (md *AccountUpdate) Validate(ctx context.Context) (err error) {
	validate := validator.New()
	lang := common.GetLangFromCtx(ctx, "")
	validate.RegisterTagNameFunc(common.TagNameI18N(lang))
	trans := common.LoadValidateTranslator(lang, validate)
	err = validate.Struct(md)
	if err != nil {
		var lines []string
		var errs validator.ValidationErrors
		errors.As(err, &errs)
		for _, v := range errs.Translate(trans) {
			lines = append(lines, v)
		}
		if len(lines) > 0 {
			err = errors.New(strings.Join(lines, "\n"))
		}
	}
	return
}

// AccountResetPassword 账户修改密码
type AccountResetPassword struct {
	//主键
	ID string `gorm:"type:varchar(50);column:id" json:"id" validate:"required" description:"记录ID"`

	//密码
	NewPassword string `json:"newPassword" validate:"required" description:"密码"`
}

func (md *AccountResetPassword) Default(ctx context.Context) {
}
func (md *AccountResetPassword) Validate(ctx context.Context) (err error) {
	validate := validator.New()
	lang := common.GetLangFromCtx(ctx, "")
	validate.RegisterTagNameFunc(common.TagNameI18N(lang))
	trans := common.LoadValidateTranslator(lang, validate)
	err = validate.Struct(md)
	if err != nil {
		var lines []string
		var errs validator.ValidationErrors
		errors.As(err, &errs)
		for _, v := range errs.Translate(trans) {
			lines = append(lines, v)
		}
		if len(lines) > 0 {
			err = errors.New(strings.Join(lines, "\n"))
		}
	}
	return
}
