package daos

import (
	"time"

	"github.com/efucloud/token-router/pkg/models"
	"gorm.io/gorm"
)

// Account 组织中的用户
type Account struct {
	//主键
	ID string `gorm:"primarykey;column:id;type:varchar(50)" json:"-" description:"记录ID"`
	//创建时间
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at;<-:create" json:"-" description:"创建时间"`
	//修改时间
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at;<-:update" json:"updatedAt,omitempty" description:"更新时间"`
	//软删除
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"deletedAt,omitempty"`
	//删除状态
	DelFlag string `gorm:"type:varchar(50);column:del_flag;default:active" json:"-" description:"删除状态"`
	//创建者
	CreatorId string `gorm:"type:varchar(50);column:creator_id;<-:create" validate:"required" json:"creatorId" description:"创建者"`
	//更新者
	UpdaterId string `gorm:"type:varchar(50);column:updater_id;<-:update" json:"updaterId" validate:"required" description:"更新者"`
	//删除者
	DeleterId string `gorm:"type:varchar(50);column:deleter_id" json:"-" description:"删除者"`
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
	TokenLimit int64 `gorm:"column:token_limit;default:0;not null" json:"tokenLimit" description:"Token总量限额"`
	//成功请求次数限额，0 表示不限制
	RequestLimit int64 `gorm:"column:request_limit;default:0;not null" json:"requestLimit" description:"成功请求次数限额"`
	//已使用 Token 数
	UsedTokens int64 `gorm:"column:used_tokens;default:0;not null" json:"usedTokens" description:"已使用Token数"`
	//已使用成功请求数
	UsedRequests int64 `gorm:"column:used_requests;default:0;not null" json:"usedRequests" description:"已使用成功请求数"`
	//并发请求已预留 Token 数
	ReservedTokens int64 `gorm:"column:reserved_tokens;default:0;not null" json:"-" description:"已预留Token数"`
	//并发请求已预留请求数
	ReservedRequests int64 `gorm:"column:reserved_requests;default:0;not null" json:"-" description:"已预留请求数"`
}

func (t *Account) Indexes() (results map[string][]string) {
	results = make(map[string][]string)

	return
}
func (t *Account) UniqueIndexes() (results map[string][]string) {
	results = make(map[string][]string)
	results["uniq_idx_username"] = []string{"username", "del_flag"}
	results["uniq_idx_phone"] = []string{"phone", "del_flag"}
	results["uniq_idx_email"] = []string{"email", "del_flag"}
	return
}

func (t *Account) TableName() string {
	return models.AccountTableName
}
