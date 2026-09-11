package v1

import (
	"context"
	"net/http"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/services"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
)

type DashboardResource struct {
	Svc services.DashboardService
}

func (r DashboardResource) AddWebService(ws *restful.WebService) {
	apiInfo := common.ApiInfo{Tag: "dashboard", Description: "双维度运行看板"}
	common.RegisterApiInfo(apiInfo)

	ws.Route(ws.GET(config.APIPrefix+"/dashboard/me").
		Doc("获取当前用户工作台").
		Notes("只返回当前 OIDC 用户的限额、Token 和调用聚合数据").
		Param(ws.HeaderParameter(config.AuthHeader, "OIDC Token")).
		Param(ws.QueryParameter("range", "时间范围: 24h、7d、30d").DataType("string")).
		To(r.me).
		Returns(http.StatusOK, "成功", dtos.MyDashboard{}).
		Returns(http.StatusBadRequest, "时间范围无效", dtos.ResponseError{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "getMyDashboard"))

	ws.Route(ws.GET(config.APIPrefix+"/dashboard/admin").
		Doc("获取管理员运行看板").
		Notes("返回企业网关全局用量、渠道健康、路由切换和限额风险").
		Param(ws.HeaderParameter(config.AuthHeader, "OIDC Token")).
		Param(ws.QueryParameter("range", "时间范围: 24h、7d、30d").DataType("string")).
		To(r.admin).
		Returns(http.StatusOK, "成功", dtos.AdminDashboard{}).
		Returns(http.StatusBadRequest, "时间范围无效", dtos.ResponseError{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "仅管理员可访问", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "getAdminDashboard"))

	ws.Route(ws.GET(config.APIPrefix+"/dashboard/users/{id}").
		Doc("获取指定用户工作台").
		Notes("管理员按用户查看与该用户个人工作台口径一致的限额、Token 和调用聚合数据").
		Param(ws.HeaderParameter(config.AuthHeader, "OIDC Token")).
		Param(ws.PathParameter("id", "用户 ID").DataType("string")).
		Param(ws.QueryParameter("range", "时间范围: 24h、7d、30d").DataType("string")).
		To(r.user).
		Returns(http.StatusOK, "成功", dtos.MyDashboard{}).
		Returns(http.StatusBadRequest, "请求参数无效", dtos.ResponseError{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "仅管理员可访问", dtos.ResponseError{}).
		Returns(http.StatusNotFound, "用户不存在", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "getUserDashboard"))
}

func dashboardContext(req *restful.Request) context.Context {
	ctx := context.Background()
	if requestContext := req.Attribute(config.RequestContext); requestContext != nil {
		ctx = requestContext.(context.Context)
	}
	return ctx
}

func (r DashboardResource) me(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, errorData := r.Svc.GetMyDashboard(ctx, req.QueryParameter("range"))
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r DashboardResource) admin(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, errorData := r.Svc.GetAdminDashboard(ctx, req.QueryParameter("range"))
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r DashboardResource) user(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, errorData := r.Svc.GetUserDashboard(ctx, req.PathParameter("id"), req.QueryParameter("range"))
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}
