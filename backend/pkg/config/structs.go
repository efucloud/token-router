package config

import (
	"context"
)

type Config struct {
	Mysql       *MysqlConfig  `json:"mysql" yaml:"mysql"`
	LogConfig   *LogConfig    `json:"logConfig" yaml:"logConfig"`
	OidcConfig  OidcConfig    `json:"oidcConfig" yaml:"oidcConfig" description:"认证配置"`
	Gateway     GatewayConfig `json:"gateway" yaml:"gateway" description:"AI网关配置"`
	Chat        ChatConfig    `json:"chat" yaml:"chat" description:"对话编排配置"`
	AdminEmails []string      `json:"adminEmails" yaml:"adminEmails" description:"管理员邮箱列表"`
}

type ChatConfig struct {
	ContextWindowTokens int             `json:"contextWindowTokens" yaml:"contextWindowTokens" description:"默认上下文窗口Token数"`
	CompactThreshold    float64         `json:"compactThreshold" yaml:"compactThreshold" description:"自动压缩触发比例"`
	CompactKeepRecent   int             `json:"compactKeepRecent" yaml:"compactKeepRecent" description:"压缩时保留的最近消息数"`
	MaxRetries          int             `json:"maxRetries" yaml:"maxRetries" description:"单次模型请求最大尝试次数"`
	RetryBaseMillis     int             `json:"retryBaseMillis" yaml:"retryBaseMillis" description:"重试基础等待毫秒数"`
	RetryMaxMillis      int             `json:"retryMaxMillis" yaml:"retryMaxMillis" description:"重试最大等待毫秒数"`
	SkillDirectories    []string        `json:"skillDirectories" yaml:"skillDirectories" description:"Skill白名单目录"`
	MCPServers          []ChatMCPConfig `json:"mcpServers" yaml:"mcpServers" description:"MCP服务端配置"`
}

type ChatMCPConfig struct {
	Name           string            `json:"name" yaml:"name" description:"服务名称"`
	Type           string            `json:"type" yaml:"type" description:"streamable-http或stdio"`
	Enabled        bool              `json:"enabled" yaml:"enabled" description:"是否发布"`
	URL            string            `json:"url" yaml:"url" description:"Streamable HTTP地址"`
	Headers        map[string]string `json:"headers" yaml:"headers" description:"远程请求头"`
	Command        string            `json:"command" yaml:"command" description:"stdio命令"`
	Args           []string          `json:"args" yaml:"args" description:"stdio参数"`
	WorkingDir     string            `json:"workingDir" yaml:"workingDir" description:"stdio工作目录"`
	Environment    map[string]string `json:"environment" yaml:"environment" description:"stdio环境变量"`
	TimeoutSeconds int               `json:"timeoutSeconds" yaml:"timeoutSeconds" description:"连接和调用超时秒数"`
}

func (c *ChatConfig) Default() {
	if c.ContextWindowTokens <= 0 {
		c.ContextWindowTokens = 32768
	}
	if c.CompactThreshold <= 0 || c.CompactThreshold >= 1 {
		c.CompactThreshold = 0.75
	}
	if c.CompactKeepRecent <= 0 {
		c.CompactKeepRecent = 8
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryBaseMillis <= 0 {
		c.RetryBaseMillis = 800
	}
	if c.RetryMaxMillis <= 0 {
		c.RetryMaxMillis = 8000
	}
}

type GatewayConfig struct {
	// SecretKey is a base64 encoded 32-byte AES-256 key used for upstream credentials.
	SecretKey              string `json:"secretKey" yaml:"secretKey" description:"上游凭据加密主密钥"`
	UpstreamTimeoutSeconds int    `json:"upstreamTimeoutSeconds" yaml:"upstreamTimeoutSeconds" description:"默认上游超时秒数"`
	MaxAttempts            int    `json:"maxAttempts" yaml:"maxAttempts" description:"单次请求最大上游尝试次数"`
	FailureThreshold       int    `json:"failureThreshold" yaml:"failureThreshold" description:"进入冷却的连续失败次数"`
	CooldownSeconds        int    `json:"cooldownSeconds" yaml:"cooldownSeconds" description:"渠道冷却秒数"`
	MaxRequestBodyBytes    int64  `json:"maxRequestBodyBytes" yaml:"maxRequestBodyBytes" description:"最大请求体字节数"`
	MaxResponseBodyBytes   int64  `json:"maxResponseBodyBytes" yaml:"maxResponseBodyBytes" description:"最大普通响应体字节数"`
	MaxReservedTokens      int64  `json:"maxReservedTokens" yaml:"maxReservedTokens" description:"单请求最大预留Token数"`
}

func (g *GatewayConfig) Default() {
	if g.UpstreamTimeoutSeconds <= 0 {
		g.UpstreamTimeoutSeconds = 120
	}
	if g.MaxAttempts <= 0 {
		g.MaxAttempts = 3
	}
	if g.FailureThreshold <= 0 {
		g.FailureThreshold = 3
	}
	if g.CooldownSeconds <= 0 {
		g.CooldownSeconds = 60
	}
	if g.MaxRequestBodyBytes <= 0 {
		g.MaxRequestBodyBytes = 16 << 20
	}
	if g.MaxResponseBodyBytes <= 0 {
		g.MaxResponseBodyBytes = 32 << 20
	}
	if g.MaxReservedTokens <= 0 {
		g.MaxReservedTokens = 131072
	}
}

type OidcConfig struct {
	ClientId          string `json:"clientId" yaml:"clientId" description:"客户端ID"`
	ClientSecret      string `json:"clientSecret" yaml:"clientSecret" description:"客户端密钥"`
	Issuer            string `json:"issuer" yaml:"issuer" description:"发行者"`
	SkipClientIDCheck bool   `json:"skipClientIDCheck" yaml:"skipClientIDCheck" description:"是否允许同一Issuer下其他客户端签发的Token"`
}
type LogConfig struct {
	Level string `json:"level" yaml:"level" description:"日志级别"`
	//Filename is the file to write logs to.  Backup log files will be retained
	//in the same directory.  It uses <processname>-lumberjack.log in
	//os.TempDir() if empty.
	Filename string `json:"filename" yaml:"filename"`

	//MaxSize is the maximum size in megabytes of the log file before it gets
	//rotated. It defaults to 100 megabytes.
	MaxSize int `json:"maxsize" yaml:"maxsize"`

	//MaxAge is the maximum number of days to retain old log files based on the
	//timestamp encoded in their filename.  Note that a day is defined as 24
	//hours and may not exactly correspond to calendar days due to daylight
	//savings, leap seconds, etc. The default is not to remove old log files
	//based on age.
	MaxAge int `json:"maxage" yaml:"maxage"`

	//MaxBackups is the maximum number of old log files to retain.  The default
	//is to retain all old log files (though MaxAge may still cause them to get
	//deleted.)
	MaxBackups int `json:"maxbackups" yaml:"maxbackups"`

	//LocalTime determines if the time used for formatting the timestamps in
	//backup files is the computer's local time.  The default is to use UTC
	//time.
	LocalTime bool `json:"localtime" yaml:"localtime"`

	//Compress determines if the rotated log files should be compressed
	//using gzip. The default is not to perform compression.
	Compress   bool `json:"compress" yaml:"compress"`
	Production bool `json:"production" yaml:"production"`
}

type RabbitMQConfig struct {
	Address string `json:"address" yaml:"address" description:"地址"`
}
type RocketMQConfig struct {
	Address []string `json:"address" yaml:"address" description:"地址"`
}
type KafkaConfig struct {
	Address []string `json:"address" yaml:"address" description:"地址"`
}
type PulsarConfig struct {
	Address []string `json:"address" yaml:"address" description:"地址"`
}

type MysqlConfig struct {
	Host     string `json:"host" yaml:"host"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	Dbname   string `json:"dbname" yaml:"dbname"`
	//utf8
	Charset string `json:"charset" yaml:"charset"`
	//Local
	Loc string `json:"loc" yaml:"loc"`
	//string 类型字段的默认长度
	DefaultStringSize uint `json:"defaultStringSize" yaml:"defaultStringSize"`
	//禁用 datetime 精度，MySQL 5.6 之前的数据库不支持
	DisableDatetimePrecision bool `json:"disableDatetimePrecision" yaml:"disableDatetimePrecision"`
	//重命名索引时采用删除并新建的方式，MySQL 5.7 之前的数据库和 MariaDB 不支持重命名索引
	DontSupportRenameIndex bool `json:"dontSupportRenameIndex" yaml:"dontSupportRenameIndex"`
	//用 `change` 重命名列，MySQL 8 之前的数据库和 MariaDB 不支持重命名列
	DontSupportRenameColumn bool `json:"dontSupportRenameColumn" yaml:"dontSupportRenameColumn"`
	//根据当前 MySQL 版本自动配置
	SkipInitializeWithVersion bool `json:"skipInitializeWithVersion" yaml:"skipInitializeWithVersion"`
}

func (m *MysqlConfig) Default(ctx context.Context) {
	if len(m.Charset) == 0 {
		m.Charset = "utf8mb4"
	}
	if len(m.Loc) == 0 {
		m.Loc = "Local"
	}
	if m.DefaultStringSize == 0 {
		m.DefaultStringSize = 255
	}
}
