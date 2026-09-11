package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/services"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
)

type ControlPlaneResource struct {
	Svc services.ControlPlaneService
}

func adminControlRoute(route *restful.RouteBuilder, apiInfo common.ApiInfo) *restful.RouteBuilder {
	return route.Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags())
}

func (r ControlPlaneResource) AddWebService(ws *restful.WebService) {
	modelInfo := common.ApiInfo{Tag: "model-management", Description: "统一模型管理"}
	providerInfo := common.ApiInfo{Tag: "provider-management", Description: "供应商管理"}
	channelInfo := common.ApiInfo{Tag: "channel-management", Description: "上游渠道管理"}
	routeInfo := common.ApiInfo{Tag: "route-management", Description: "模型路由管理"}
	usageInfo := common.ApiInfo{Tag: "usage-management", Description: "网关用量与路由追踪"}
	auditInfo := common.ApiInfo{Tag: "audit-management", Description: "管理操作审计"}
	diagnosticInfo := common.ApiInfo{Tag: "diagnostic-management", Description: "管理员模型演练诊断"}
	for _, info := range []common.ApiInfo{modelInfo, providerInfo, channelInfo, routeInfo, usageInfo, auditInfo, diagnosticInfo} {
		common.RegisterApiInfo(info)
	}

	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/models").Doc("查询统一模型").
		Param(ws.QueryParameter("search", "名称搜索")).Param(ws.QueryParameter("status", "状态")).
		Param(ws.QueryParameter("channelId", "按渠道筛选")).
		To(r.listModels).Returns(http.StatusOK, "成功", dtos.AIModelList{}), modelInfo).
		Metadata(config.FrontApiTag, "listModels"))
	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/models").Doc("创建统一模型").
		Reads(dtos.AIModelInput{}).To(r.createModel).Returns(http.StatusOK, "成功", dtos.AIModelDetail{}), modelInfo).
		Metadata(config.FrontApiTag, "createModel"))
	ws.Route(adminControlRoute(ws.PUT(config.APIPrefix+"/models/{id}").Doc("更新统一模型").
		Param(ws.PathParameter("id", "模型ID").Required(true)).Reads(dtos.AIModelInput{}).
		To(r.updateModel).Returns(http.StatusOK, "成功", dtos.AIModelDetail{}), modelInfo).
		Metadata(config.FrontApiTag, "updateModel"))
	ws.Route(adminControlRoute(ws.DELETE(config.APIPrefix+"/models/{id}").Doc("删除统一模型").
		Param(ws.PathParameter("id", "模型ID").Required(true)).To(r.deleteModel).
		Returns(http.StatusOK, "成功", "success"), modelInfo).Metadata(config.FrontApiTag, "deleteModel"))

	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/providers").Doc("查询供应商").
		Param(ws.QueryParameter("search", "名称搜索")).Param(ws.QueryParameter("status", "状态")).
		To(r.listProviders).Returns(http.StatusOK, "成功", dtos.ProviderList{}), providerInfo).
		Metadata(config.FrontApiTag, "listProviders"))
	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/providers").Doc("创建供应商").
		Reads(dtos.ProviderInput{}).To(r.createProvider).Returns(http.StatusOK, "成功", dtos.ProviderDetail{}), providerInfo).
		Metadata(config.FrontApiTag, "createProvider"))
	ws.Route(adminControlRoute(ws.PUT(config.APIPrefix+"/providers/{id}").Doc("更新供应商").
		Param(ws.PathParameter("id", "供应商ID").Required(true)).Reads(dtos.ProviderInput{}).
		To(r.updateProvider).Returns(http.StatusOK, "成功", dtos.ProviderDetail{}), providerInfo).
		Metadata(config.FrontApiTag, "updateProvider"))
	ws.Route(adminControlRoute(ws.DELETE(config.APIPrefix+"/providers/{id}").Doc("删除供应商").
		Param(ws.PathParameter("id", "供应商ID").Required(true)).To(r.deleteProvider).
		Returns(http.StatusOK, "成功", "success"), providerInfo).Metadata(config.FrontApiTag, "deleteProvider"))

	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/channels").Doc("查询渠道").
		Param(ws.QueryParameter("search", "名称、地址或供应商搜索")).
		Param(ws.QueryParameter("providerId", "供应商ID")).Param(ws.QueryParameter("status", "状态")).
		To(r.listChannels).Returns(http.StatusOK, "成功", dtos.ChannelList{}), channelInfo).
		Metadata(config.FrontApiTag, "listChannels"))
	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/channels").Doc("创建渠道").
		Reads(dtos.ChannelInput{}).To(r.createChannel).Returns(http.StatusOK, "成功", dtos.ChannelDetail{}), channelInfo).
		Metadata(config.FrontApiTag, "createChannel"))
	ws.Route(adminControlRoute(ws.PUT(config.APIPrefix+"/channels/{id}").Doc("更新渠道").
		Param(ws.PathParameter("id", "渠道ID").Required(true)).Reads(dtos.ChannelInput{}).
		To(r.updateChannel).Returns(http.StatusOK, "成功", dtos.ChannelDetail{}), channelInfo).
		Metadata(config.FrontApiTag, "updateChannel"))
	ws.Route(adminControlRoute(ws.DELETE(config.APIPrefix+"/channels/{id}").Doc("删除渠道").
		Param(ws.PathParameter("id", "渠道ID").Required(true)).To(r.deleteChannel).
		Returns(http.StatusOK, "成功", "success"), channelInfo).Metadata(config.FrontApiTag, "deleteChannel"))
	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/channels/{id}/test").Doc("测试渠道连接并发现模型").
		Param(ws.PathParameter("id", "渠道ID").Required(true)).To(r.testChannel).
		Returns(http.StatusOK, "成功", dtos.ChannelTestResult{}), channelInfo).
		Metadata(config.FrontApiTag, "testChannel"))
	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/channels/{id}/models/import").Doc("导入渠道发现的模型和路由").
		Param(ws.PathParameter("id", "渠道ID").Required(true)).Reads(dtos.ChannelModelImportInput{}).To(r.importChannelModels).
		Returns(http.StatusOK, "成功", dtos.ChannelModelImportResult{}), channelInfo).
		Metadata(config.FrontApiTag, "importChannelModels"))
	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/channels/{id}/models/sync").Doc("同步渠道模型并清理失效模型").
		Param(ws.PathParameter("id", "渠道ID").Required(true)).Reads(dtos.ChannelModelSyncInput{}).To(r.syncChannelModels).
		Returns(http.StatusOK, "成功", dtos.ChannelModelSyncResult{}), channelInfo).
		Metadata(config.FrontApiTag, "syncChannelModels"))

	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/routes").Doc("查询模型路由").
		Param(ws.QueryParameter("modelId", "统一模型ID")).Param(ws.QueryParameter("channelId", "渠道ID")).
		Param(ws.QueryParameter("status", "状态")).To(r.listRoutes).
		Returns(http.StatusOK, "成功", dtos.ModelRouteList{}), routeInfo).Metadata(config.FrontApiTag, "listRoutes"))
	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/routes").Doc("创建模型路由").
		Reads(dtos.ModelRouteInput{}).To(r.createRoute).Returns(http.StatusOK, "成功", dtos.ModelRouteDetail{}), routeInfo).
		Metadata(config.FrontApiTag, "createRoute"))
	ws.Route(adminControlRoute(ws.PUT(config.APIPrefix+"/routes/{id}").Doc("更新模型路由").
		Param(ws.PathParameter("id", "路由ID").Required(true)).Reads(dtos.ModelRouteInput{}).
		To(r.updateRoute).Returns(http.StatusOK, "成功", dtos.ModelRouteDetail{}), routeInfo).
		Metadata(config.FrontApiTag, "updateRoute"))
	ws.Route(adminControlRoute(ws.DELETE(config.APIPrefix+"/routes/{id}").Doc("删除模型路由").
		Param(ws.PathParameter("id", "路由ID").Required(true)).To(r.deleteRoute).
		Returns(http.StatusOK, "成功", "success"), routeInfo).Metadata(config.FrontApiTag, "deleteRoute"))

	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/usage").Doc("查询用量日志").
		Param(ws.QueryParameter("current", "页码").DataType("number")).Param(ws.QueryParameter("pageSize", "每页大小").DataType("number")).
		Param(ws.QueryParameter("requestId", "请求ID")).Param(ws.QueryParameter("accountId", "用户ID")).
		Param(ws.QueryParameter("modelId", "模型ID")).Param(ws.QueryParameter("channelId", "渠道ID")).
		Param(ws.QueryParameter("status", "状态")).Param(ws.QueryParameter("start", "开始时间，RFC3339")).
		Param(ws.QueryParameter("end", "结束时间，RFC3339")).To(r.listUsage).
		Returns(http.StatusOK, "成功", dtos.UsageLogList{}), usageInfo).Metadata(config.FrontApiTag, "listUsage"))
	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/usage/{requestId}/attempts").Doc("查询请求的路由尝试").
		Param(ws.PathParameter("requestId", "请求ID").Required(true)).To(r.listRequestAttempts).
		Returns(http.StatusOK, "成功", dtos.RouteAttemptList{}), usageInfo).Metadata(config.FrontApiTag, "listUsageAttempts"))
	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/route-attempts").Doc("查询全部路由尝试").
		Param(ws.QueryParameter("current", "页码").DataType("number")).Param(ws.QueryParameter("pageSize", "每页大小").DataType("number")).
		Param(ws.QueryParameter("requestId", "请求ID")).Param(ws.QueryParameter("channelId", "渠道ID")).
		To(r.listAttempts).Returns(http.StatusOK, "成功", dtos.RouteAttemptList{}), usageInfo).
		Metadata(config.FrontApiTag, "listRouteAttempts"))

	ws.Route(adminControlRoute(ws.GET(config.APIPrefix+"/audit-logs").Doc("查询管理操作审计日志").
		Param(ws.QueryParameter("current", "页码").DataType("number")).Param(ws.QueryParameter("pageSize", "每页大小").DataType("number")).
		Param(ws.QueryParameter("operatorId", "操作者ID")).Param(ws.QueryParameter("action", "操作")).
		Param(ws.QueryParameter("resourceType", "资源类型")).Param(ws.QueryParameter("result", "执行结果")).
		To(r.listAuditLogs).Returns(http.StatusOK, "成功", dtos.AuditLogList{}), auditInfo).
		Metadata(config.FrontApiTag, "listAuditLogs"))

	ws.Route(adminControlRoute(ws.POST(config.APIPrefix+"/model-diagnostics").Doc("运行管理员模型演练诊断").
		Reads(dtos.ModelDiagnosticInput{}).To(r.runModelDiagnostic).
		Returns(http.StatusOK, "成功", dtos.ModelDiagnosticResult{}), diagnosticInfo).
		Metadata(config.FrontApiTag, "runModelDiagnostic"))
}

func controlContext(req *restful.Request) context.Context {
	if value := req.Attribute(config.RequestContext); value != nil {
		return value.(context.Context)
	}
	return context.Background()
}

func controlRead(req *restful.Request, resp *restful.Response, target any) bool {
	if err := req.ReadEntity(target); err != nil {
		ctx := controlContext(req)
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, common.ErrorData{Err: err, ResponseCode: http.StatusBadRequest, MsgCode: config.MsgCodeJsonDecodeFailed})
		return false
	}
	encoded, _ := json.Marshal(target)
	fields := map[string]any{}
	if json.Unmarshal(encoded, &fields) == nil {
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		req.SetAttribute("auditChangedFields", strings.Join(names, ","))
	}
	return true
}

func controlRespond(req *restful.Request, resp *restful.Response, result any, errorData common.ErrorData) {
	ctx := controlContext(req)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ControlPlaneResource) listModels(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListModels(controlContext(req), req.QueryParameter("search"), req.QueryParameter("status"), req.QueryParameter("channelId"))
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) createModel(req *restful.Request, resp *restful.Response) {
	var input dtos.AIModelInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.CreateModel(controlContext(req), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) updateModel(req *restful.Request, resp *restful.Response) {
	var input dtos.AIModelInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.UpdateModel(controlContext(req), req.PathParameter("id"), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) deleteModel(req *restful.Request, resp *restful.Response) {
	controlRespond(req, resp, "success", r.Svc.DeleteModel(controlContext(req), req.PathParameter("id")))
}

func (r ControlPlaneResource) listProviders(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListProviders(controlContext(req), req.QueryParameter("search"), req.QueryParameter("status"))
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) createProvider(req *restful.Request, resp *restful.Response) {
	var input dtos.ProviderInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.CreateProvider(controlContext(req), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) updateProvider(req *restful.Request, resp *restful.Response) {
	var input dtos.ProviderInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.UpdateProvider(controlContext(req), req.PathParameter("id"), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) deleteProvider(req *restful.Request, resp *restful.Response) {
	controlRespond(req, resp, "success", r.Svc.DeleteProvider(controlContext(req), req.PathParameter("id")))
}

func (r ControlPlaneResource) listChannels(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListChannels(controlContext(req), req.QueryParameter("search"), req.QueryParameter("providerId"), req.QueryParameter("status"))
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) createChannel(req *restful.Request, resp *restful.Response) {
	var input dtos.ChannelInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.CreateChannel(controlContext(req), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) updateChannel(req *restful.Request, resp *restful.Response) {
	var input dtos.ChannelInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.UpdateChannel(controlContext(req), req.PathParameter("id"), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) deleteChannel(req *restful.Request, resp *restful.Response) {
	controlRespond(req, resp, "success", r.Svc.DeleteChannel(controlContext(req), req.PathParameter("id")))
}

func (r ControlPlaneResource) testChannel(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.TestChannel(controlContext(req), req.PathParameter("id"))
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) importChannelModels(req *restful.Request, resp *restful.Response) {
	var input dtos.ChannelModelImportInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.ImportChannelModels(controlContext(req), req.PathParameter("id"), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) syncChannelModels(req *restful.Request, resp *restful.Response) {
	var input dtos.ChannelModelSyncInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.SyncChannelModels(controlContext(req), req.PathParameter("id"))
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) listRoutes(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListRoutes(controlContext(req), req.QueryParameter("modelId"), req.QueryParameter("channelId"), req.QueryParameter("status"))
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) createRoute(req *restful.Request, resp *restful.Response) {
	var input dtos.ModelRouteInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.CreateRoute(controlContext(req), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) updateRoute(req *restful.Request, resp *restful.Response) {
	var input dtos.ModelRouteInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.UpdateRoute(controlContext(req), req.PathParameter("id"), input)
		controlRespond(req, resp, result, errorData)
	}
}

func (r ControlPlaneResource) deleteRoute(req *restful.Request, resp *restful.Response) {
	controlRespond(req, resp, "success", r.Svc.DeleteRoute(controlContext(req), req.PathParameter("id")))
}

func queryInt(req *restful.Request, name string, fallback int) int {
	value, err := strconv.Atoi(req.QueryParameter(name))
	if err != nil {
		return fallback
	}
	return value
}

func queryTime(req *restful.Request, name string) (*time.Time, error) {
	value := req.QueryParameter(name)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	return &parsed, err
}

func (r ControlPlaneResource) listUsage(req *restful.Request, resp *restful.Response) {
	start, startErr := queryTime(req, "start")
	end, endErr := queryTime(req, "end")
	if startErr != nil || endErr != nil {
		controlRespond(req, resp, nil, common.ErrorData{Err: errors.New("start and end must use RFC3339"), ResponseCode: http.StatusBadRequest, MsgCode: config.MsgCodeRequestDataInvalid})
		return
	}
	result, errorData := r.Svc.ListUsage(controlContext(req), queryInt(req, "current", 1), queryInt(req, "pageSize", 20), req.QueryParameter("requestId"), req.QueryParameter("accountId"), req.QueryParameter("modelId"), req.QueryParameter("channelId"), req.QueryParameter("status"), start, end)
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) listRequestAttempts(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListAttempts(controlContext(req), 1, 200, req.PathParameter("requestId"), "")
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) listAttempts(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListAttempts(controlContext(req), queryInt(req, "current", 1), queryInt(req, "pageSize", 20), req.QueryParameter("requestId"), req.QueryParameter("channelId"))
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) listAuditLogs(req *restful.Request, resp *restful.Response) {
	result, errorData := r.Svc.ListAuditLogs(
		controlContext(req), queryInt(req, "current", 1), queryInt(req, "pageSize", 20),
		req.QueryParameter("operatorId"), req.QueryParameter("action"),
		req.QueryParameter("resourceType"), req.QueryParameter("result"),
	)
	controlRespond(req, resp, result, errorData)
}

func (r ControlPlaneResource) runModelDiagnostic(req *restful.Request, resp *restful.Response) {
	var input dtos.ModelDiagnosticInput
	if controlRead(req, resp, &input) {
		result, errorData := r.Svc.RunModelDiagnostic(controlContext(req), input)
		controlRespond(req, resp, result, errorData)
	}
}
