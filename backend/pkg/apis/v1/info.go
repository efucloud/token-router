package v1

import (
	"context"
	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/utils"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"net/http"
	"time"
)

type InfoResource struct {
}

type probeStatus struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func (r InfoResource) AddWebService(ws *restful.WebService) {
	apiInfo := common.ApiInfo{}
	apiInfo.Tag = "system-info"
	apiInfo.Description = "应用信息"
	common.RegisterApiInfo(apiInfo)
	ws.Route(ws.GET(config.APIPrefix+"/health").
		Doc("健康检查").
		Notes("健康检查").
		To(r.health).
		Returns(http.StatusOK, "成功", "ok").
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "health"))
	ws.Route(ws.GET("/livez").Doc("存活探针").To(r.liveness).
		Returns(http.StatusOK, "进程存活", probeStatus{}).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()))
	ws.Route(ws.GET("/readyz").Doc("就绪探针").To(r.readiness).
		Returns(http.StatusOK, "服务就绪", probeStatus{}).
		Returns(http.StatusServiceUnavailable, "服务未就绪", probeStatus{}).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()))
	ws.Route(ws.GET(config.APIPrefix+"/info").
		Doc("查看应用信息").
		Notes("查看应用的编译信息").
		To(r.info).
		Returns(http.StatusOK, "成功", dtos.ApplicationInfo{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "info"))

	ws.Route(ws.GET(config.APIPrefix+"/generateDatabaseId").
		Doc("生成数据库ID").
		Notes(`生成数据库ID`).
		To(r.generateDatabaseId).
		Returns(http.StatusOK, "ok", "").
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "generateDatabaseId"))

}
func (r InfoResource) generateDatabaseId(req *restful.Request, resp *restful.Response) {
	common.ResponseSuccess(resp, utils.GenerateDatabaseId())
}
func (r InfoResource) info(req *restful.Request, resp *restful.Response) {
	var app dtos.ApplicationInfo
	app.Application = config.ApplicationName
	app.BuildDate = config.BuildDate
	app.GoVersion = config.GoVersion
	app.Commit = config.Commit
	common.ResponseSuccess(resp, app)
}

func (r InfoResource) health(req *restful.Request, resp *restful.Response) {
	common.ResponseSuccess(resp, "ok")
}

func (r InfoResource) liveness(_ *restful.Request, resp *restful.Response) {
	_ = resp.WriteHeaderAndEntity(http.StatusOK, probeStatus{Status: "ok"})
}

func (r InfoResource) readiness(req *restful.Request, resp *restful.Response) {
	checks := map[string]string{"database": "ok", "schema": "ok"}
	ready := true
	if config.DBConnect == nil {
		checks["database"] = "not configured"
		ready = false
	} else if sqlDB, err := config.DBConnect.DB(); err != nil {
		checks["database"] = "unavailable"
		ready = false
	} else {
		ctx, cancel := context.WithTimeout(req.Request.Context(), 2*time.Second)
		defer cancel()
		if err = sqlDB.PingContext(ctx); err != nil {
			checks["database"] = "unavailable"
			ready = false
		}
		for _, table := range []string{"account", "ai_model", "channel", "model_route", "api_token", "usage_log", "audit_log"} {
			if !config.DBConnect.Migrator().HasTable(table) {
				checks["schema"] = "migration required"
				ready = false
				break
			}
		}
	}
	status := http.StatusOK
	result := "ok"
	if !ready {
		status = http.StatusServiceUnavailable
		result = "not_ready"
	}
	_ = resp.WriteHeaderAndEntity(status, probeStatus{Status: result, Checks: checks})
}
