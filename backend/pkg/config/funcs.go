package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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
	writeSyncer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   conf.Filename,
		MaxSize:    conf.MaxSize,
		MaxBackups: conf.MaxBackups,
		MaxAge:     conf.MaxAge,
		Compress:   conf.Compress,
	})
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
	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncer, zapcore.AddSync(os.Stdout)), level)
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
