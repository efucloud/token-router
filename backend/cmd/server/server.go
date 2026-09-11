package server

import (
	"context"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/efucloud/common"
	"github.com/efucloud/common/signals"
	"github.com/efucloud/token-router/cmd/server/options"
	"github.com/efucloud/token-router/pkg/apis"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/crons"
	"github.com/efucloud/token-router/pkg/embeds"
	"github.com/efucloud/token-router/pkg/migrations"
	"github.com/spf13/cobra"
	"net/http"
	"strings"
	"time"
)

// 选举配置常量
const (
	LeaderLockName = "token-router-leader"
	LeaseDuration  = 15 * time.Second
	RenewDeadline  = 10 * time.Second
	RetryPeriod    = 2 * time.Second
)

func oidcVerifierConfig(input config.OidcConfig) oidc.Config {
	return oidc.Config{
		ClientID:          input.ClientId,
		SkipClientIDCheck: input.SkipClientIDCheck,
	}
}

func removeLastTwoSegments(s string) string {
	lastIdx := strings.LastIndex(s, "-")
	if lastIdx == -1 {
		return s
	}
	temp := s[:lastIdx]
	secondLastIdx := strings.LastIndex(temp, "-")
	if secondLastIdx == -1 {
		return temp
	}
	return temp[:secondLastIdx]
}

func NewRunnerServerCommand() *cobra.Command {
	s := options.NewServerRunOptions()
	cmd := &cobra.Command{
		Use:          "server",
		Long:         `token-router server`,
		Short:        "token-router server",
		Example:      `token-router server`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return run(s, signals.SetupSignalHandler())
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&s.Config, "config", "c", "./config/config.yaml", "config file path")
	return cmd
}

func run(o *options.ServerRunOptions, stopCh <-chan struct{}) (err error) {
	// === 1. 基础初始化 (所有环境都执行) ===
	common.LoadConfig(o.Config, config.ApplicationConfig)
	config.ApplicationConfig.Init()
	config.Logger.Infof("build info GoVersion %s", config.GoVersion)
	config.Logger.Infof("build info Commit %s", config.Commit)
	config.Logger.Infof("build info BuildDate %s", config.BuildDate)

	config.Bundle, _ = common.I18nInit(embeds.I18nFiles, config.Logger)

	ctx := context.TODO()
	// 数据库迁移
	migrations.DatabaseMigrate()
	// 注册 API
	apis.AddResources()
	// OIDC 初始化
	if config.ApplicationConfig.OidcConfig.Issuer == "" {
		config.Logger.Warn("OIDC issuer is not configured; control-plane authentication is unavailable")
	} else {
		config.AuthProvider, err = oidc.NewProvider(ctx, config.ApplicationConfig.OidcConfig.Issuer)
		if err != nil {
			config.Logger.Fatalf("get oidc config from: %s failed, err: %s", config.ApplicationConfig.OidcConfig.Issuer, err)
		}
		if config.AuthProvider != nil {
			oidcConfig := oidcVerifierConfig(config.ApplicationConfig.OidcConfig)
			config.SystemVerifier = config.AuthProvider.Verifier(&oidcConfig)
		}
	}
	go func() {
		config.Logger.Infof("ready to start http server on port: %d", config.ServerPort)

		if err := http.ListenAndServe(fmt.Sprintf(":%d", config.ServerPort), nil); err != nil {
			config.Logger.Fatal("http server failed: " + err.Error())
		}
	}()

	config.Logger.Info("EfuCloudMode detected: starting CronJob directly (no election).")
	crons.StartCronJob()

	// ✅ 关键修复：EfuCloud 模式下没有选举阻塞主线程，需要手动等待停止信号
	<-stopCh
	config.Logger.Info("received shutdown signal, exiting EfuCloudMode")

	return nil
}
