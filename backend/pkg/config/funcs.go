package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	chatSkillDirectoriesEnv      = "TOKEN_ROUTER_CHAT_SKILL_DIRECTORIES"
	chatMCPConfigFileEnv         = "TOKEN_ROUTER_CHAT_MCP_CONFIG_FILE"
	chatBuiltinToolsEnabledEnv   = "TOKEN_ROUTER_CHAT_BUILTIN_TOOLS_ENABLED"
	chatBuiltinDefaultEnabledEnv = "TOKEN_ROUTER_CHAT_BUILTIN_DEFAULT_ENABLED"
	chatBuiltinCommandEnabledEnv = "TOKEN_ROUTER_CHAT_BUILTIN_COMMAND_ENABLED"
	chatBuiltinWorkspaceEnv      = "TOKEN_ROUTER_CHAT_BUILTIN_WORKSPACE"
	chatWorkspaceMaxUploadEnv    = "TOKEN_ROUTER_CHAT_WORKSPACE_MAX_UPLOAD_BYTES"
)

type chatMCPConfigDocument struct {
	MCPServers []ChatMCPConfig `json:"mcpServers" yaml:"mcpServers"`
}

func (c *ChatConfig) applyEnvironment() error {
	if value, configured := os.LookupEnv(chatSkillDirectoriesEnv); configured {
		c.SkillDirectories = make([]string, 0)
		for _, directory := range filepath.SplitList(value) {
			if directory = strings.TrimSpace(directory); directory != "" {
				c.SkillDirectories = append(c.SkillDirectories, directory)
			}
		}
	}
	for _, item := range []struct {
		name   string
		target *bool
	}{
		{name: chatBuiltinToolsEnabledEnv, target: &c.BuiltinTools.Enabled},
		{name: chatBuiltinDefaultEnabledEnv, target: &c.BuiltinTools.DefaultEnabled},
		{name: chatBuiltinCommandEnabledEnv, target: &c.BuiltinTools.CommandEnabled},
	} {
		if value, configured := os.LookupEnv(item.name); configured {
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("parse %s: %w", item.name, err)
			}
			*item.target = enabled
		}
	}
	if value, configured := os.LookupEnv(chatBuiltinWorkspaceEnv); configured {
		if value = strings.TrimSpace(value); value == "" {
			return fmt.Errorf("%s must not be empty", chatBuiltinWorkspaceEnv)
		}
		c.BuiltinTools.WorkspaceDirectory = value
	}
	if value, configured := os.LookupEnv(chatWorkspaceMaxUploadEnv); configured {
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("%s must be a positive integer", chatWorkspaceMaxUploadEnv)
		}
		c.BuiltinTools.MaxUploadBytes = parsed
	}

	path := strings.TrimSpace(os.Getenv(chatMCPConfigFileEnv))
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read MCP config file %q: %w", path, err)
	}
	var document chatMCPConfigDocument
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err = decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode MCP config file %q: %w", path, err)
	}
	if document.MCPServers == nil {
		return fmt.Errorf("decode MCP config file %q: mcpServers is required", path)
	}
	var trailing any
	if err = decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode MCP config file %q: multiple YAML documents are not supported", path)
		}
		return fmt.Errorf("decode MCP config file %q: %w", path, err)
	}
	c.MCPServers = document.MCPServers
	return nil
}

// createDBConnection  create database connection
func createDBConnection() (err error) {
	ctx := context.Background()
	if ApplicationConfig.Mysql != nil {
		Logger.Info("database is Mysql")
		ApplicationConfig.Mysql.Default(ctx)
		c := ApplicationConfig.Mysql
		dns := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&parseTime=True&loc=%s",
			c.User, c.Password, c.Host, c.Dbname, c.Charset, c.Loc)
		DBConnect, err = gorm.Open(mysql.New(mysql.Config{
			DSN:                       dns,
			DefaultStringSize:         c.DefaultStringSize,
			DisableDatetimePrecision:  c.DisableDatetimePrecision,
			DontSupportRenameIndex:    c.DontSupportRenameIndex,
			DontSupportRenameColumn:   c.DontSupportRenameColumn,
			SkipInitializeWithVersion: c.SkipInitializeWithVersion,
		}), &gorm.Config{
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		})
		if err == nil {
			sqlDB, _ := DBConnect.DB()
			//SetMaxIdleConns 设置空闲连接池中连接的最大数量
			sqlDB.SetMaxIdleConns(1)
			//SetMaxOpenConns 设置打开数据库连接的最大数量。
			sqlDB.SetMaxOpenConns(100)
			//SetConnMaxLifetime 设置了连接可复用的最大时间。
			sqlDB.SetConnMaxLifetime(5 * time.Minute)
		} else {
			Logger.Errorf("database connect failed, err: %s", err.Error())
		}
	} else {
		Logger.Info("database is sqlite")
		DBConnect, err = gorm.Open(sqlite.Open("eauth.db"), &gorm.Config{
			NowFunc: func() time.Time {
				return time.Now().Local()
			},
		})
	}
	if err != nil {
		Logger.Errorf("create database connect failed, err: %s", err.Error())
	}
	return err
}
func logConfig(conf *LogConfig) {
	writeSyncers := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	if strings.TrimSpace(conf.Filename) != "" {
		writeSyncers = append(writeSyncers, zapcore.AddSync(&lumberjack.Logger{
			Filename:   conf.Filename,
			MaxSize:    conf.MaxSize,
			MaxBackups: conf.MaxBackups,
			MaxAge:     conf.MaxAge,
			Compress:   conf.Compress,
		}))
	}
	var encoderConfig zapcore.EncoderConfig
	if conf.Production {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	var level zapcore.Level
	switch conf.Level {
	case "info":
		level = zapcore.InfoLevel
	case "debug":
		level = zapcore.DebugLevel
	default:
		level = zapcore.InfoLevel
	}
	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncers...), level)
	logger := zap.New(core, zap.AddCaller())
	Logger = logger.Sugar()

}

func (c *Config) Init() {
	if value := os.Getenv("TOKEN_ROUTER_OIDC_ISSUER"); value != "" {
		c.OidcConfig.Issuer = value
	}
	if value := os.Getenv("TOKEN_ROUTER_OIDC_CLIENT_ID"); value != "" {
		c.OidcConfig.ClientId = value
	}
	if value := os.Getenv("TOKEN_ROUTER_OIDC_CLIENT_SECRET"); value != "" {
		c.OidcConfig.ClientSecret = value
	}
	if value := os.Getenv("TOKEN_ROUTER_OIDC_SKIP_CLIENT_ID_CHECK"); value != "" {
		if enabled, err := strconv.ParseBool(value); err == nil {
			c.OidcConfig.SkipClientIDCheck = enabled
		}
	}
	if value := os.Getenv("TOKEN_ROUTER_GATEWAY_SECRET_KEY"); value != "" {
		c.Gateway.SecretKey = value
	}
	c.Gateway.Default()
	c.Chat.Default()
	if c.LogConfig == nil {
		c.LogConfig = new(LogConfig)
		c.LogConfig.Filename = "./log/token-router.log"
		c.LogConfig.MaxAge = 30
		c.LogConfig.MaxSize = 1
		c.LogConfig.MaxBackups = 10
		c.LogConfig.Compress = false
	}
	logConfig(c.LogConfig)
	if err := c.Chat.applyEnvironment(); err != nil {
		Logger.Fatalf("load chat capability config failed: %s", err.Error())
	}
	c.OidcConfig.Issuer = strings.TrimSuffix(c.OidcConfig.Issuer, "/")
	if err := createDBConnection(); err != nil {
		Logger.Fatalf("create database connect failed, err: %s", err.Error())
	}

}
func GetLangFromCtx(ctx context.Context) (lang string) {
	lang = "zh"
	lan := ctx.Value(RequestLanguage)
	if lan != nil {
		lang = lan.(string)
	}
	return lang
}
func GetOperatorFromCtx(ctx context.Context) (operator string) {
	operator = "unknown"
	requesterId := ctx.Value(RequestUserId)
	if requesterId != nil {
		operator = requesterId.(string)
	}
	return operator
}
