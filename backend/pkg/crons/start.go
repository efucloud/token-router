package crons

import (
	"context"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/services"
	"github.com/robfig/cron/v3"
)

func StartCronJob() {
	config.Logger.Info("start cron job")
	c := cron.New()
	probe := cron.FuncJob(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		healthy, unhealthy := (services.ControlPlaneService{}).ProbeEnabledChannels(ctx)
		config.Logger.Infof("channel health probe completed: healthy=%d unhealthy=%d", healthy, unhealthy)
	})
	wrappedProbe := cron.SkipIfStillRunning(cron.DefaultLogger)(probe)
	if _, err := c.AddJob("@every 1m", wrappedProbe); err != nil {
		config.Logger.Errorf("register channel health probe failed: %v", err)
	}
	c.Start()
	go wrappedProbe.Run()
}
