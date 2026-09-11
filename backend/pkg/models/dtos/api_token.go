package dtos

import "time"

type APITokenCreate struct {
	Name           string     `json:"name" validate:"required,max=255" description:"Token名称"`
	AllowedModels  []string   `json:"allowedModels" description:"允许的统一模型，空数组表示全部"`
	TokenLimit     int64      `json:"tokenLimit" validate:"gte=0" description:"Token总量限额，0表示不限制"`
	RequestLimit   int64      `json:"requestLimit" validate:"gte=0" description:"成功请求次数限额，0表示不限制"`
	IPAllowlist    []string   `json:"ipAllowlist" description:"允许的IP或CIDR，空数组表示不限制"`
	RPM            int        `json:"rpm" validate:"gte=0" description:"每分钟请求数，0表示不限制"`
	TPM            int64      `json:"tpm" validate:"gte=0" description:"每分钟Token数，0表示不限制"`
	MaxConcurrency int        `json:"maxConcurrency" validate:"gte=0" description:"最大并发请求数，0表示不限制"`
	ExpiresAt      *time.Time `json:"expiresAt" description:"过期时间"`
}

type APITokenUpdate struct {
	Name           string     `json:"name" validate:"required,max=255" description:"Token名称"`
	AllowedModels  []string   `json:"allowedModels" description:"允许的统一模型，空数组表示全部"`
	TokenLimit     int64      `json:"tokenLimit" validate:"gte=0" description:"Token总量限额，0表示不限制"`
	RequestLimit   int64      `json:"requestLimit" validate:"gte=0" description:"成功请求次数限额，0表示不限制"`
	IPAllowlist    []string   `json:"ipAllowlist" description:"允许的IP或CIDR，空数组表示不限制"`
	RPM            int        `json:"rpm" validate:"gte=0" description:"每分钟请求数，0表示不限制"`
	TPM            int64      `json:"tpm" validate:"gte=0" description:"每分钟Token数，0表示不限制"`
	MaxConcurrency int        `json:"maxConcurrency" validate:"gte=0" description:"最大并发请求数，0表示不限制"`
	ExpiresAt      *time.Time `json:"expiresAt" description:"过期时间"`
	Status         string     `json:"status" validate:"oneof=active disabled" enum:"active|disabled" description:"状态"`
}

type APITokenDetail struct {
	ID             string     `json:"id" description:"记录ID"`
	Name           string     `json:"name" description:"Token名称"`
	KeyPrefix      string     `json:"keyPrefix" description:"Token识别前缀"`
	Status         string     `json:"status" description:"状态"`
	AllowedModels  []string   `json:"allowedModels" description:"允许的统一模型"`
	TokenLimit     int64      `json:"tokenLimit" description:"Token总量限额"`
	RequestLimit   int64      `json:"requestLimit" description:"请求次数限额"`
	IPAllowlist    []string   `json:"ipAllowlist" description:"允许的IP或CIDR"`
	RPM            int        `json:"rpm" description:"每分钟请求数"`
	TPM            int64      `json:"tpm" description:"每分钟Token数"`
	MaxConcurrency int        `json:"maxConcurrency" description:"最大并发请求数"`
	UsedTokens     int64      `json:"usedTokens" description:"已使用Token"`
	UsedRequests   int64      `json:"usedRequests" description:"已使用请求数"`
	ExpiresAt      *time.Time `json:"expiresAt" description:"过期时间"`
	CreatedAt      time.Time  `json:"createdAt" description:"创建时间"`
	UpdatedAt      time.Time  `json:"updatedAt" description:"更新时间"`
}

type APITokenCreated struct {
	APITokenDetail
	Token string `json:"token" description:"只显示一次的 API Key 明文"`
}

type APITokenList struct {
	Data  []APITokenDetail `json:"data"`
	Total int64            `json:"total"`
}
