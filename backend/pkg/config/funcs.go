package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func createRedisConnection() {
	RedisClient = nil
	if ApplicationConfig.Redis == nil || !ApplicationConfig.Redis.Enabled {
		Logger.Info("redis route cache is disabled")
		return
	}
	ApplicationConfig.Redis.Default()
	c := ApplicationConfig.Redis
	options := &redis.Options{
		Addr:         c.Address,
		Username:     c.Username,
		Password:     c.Password,
		DB:           c.DB,
		MaxRetries:   1,
		DialTimeout:  time.Duration(c.DialTimeoutMillis) * time.Millisecond,
		ReadTimeout:  time.Duration(c.ReadTimeoutMillis) * time.Millisecond,
		WriteTimeout: time.Duration(c.WriteTimeoutMillis) * time.Millisecond,
		PoolSize:     c.PoolSize,
	}
	if c.TLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	client := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.DialTimeoutMillis)*time.Millisecond)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		Logger.Warnf("redis route cache unavailable, continuing with database: %v", err)
		return
	}
	RedisClient = client
	Logger.Infof("redis route cache enabled at %s", c.Address)
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
	createRedisConnection()
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
