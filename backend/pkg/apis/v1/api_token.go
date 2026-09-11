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

type APITokenResource struct {
	Svc services.APITokenService
}

func (r APITokenResource) AddWebService(ws *restful.WebService) {
	apiInfo := common.ApiInfo{Tag: "api-token", Description: "用户 API Key"}
	common.RegisterApiInfo(apiInfo)
	base := config.APIPrefix + "/tokens"
	filtersForRoute := func(route *restful.RouteBuilder) *restful.RouteBuilder {
		return route.Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
			Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags())
	}
	ws.Route(filtersForRoute(ws.GET(base).
		Doc("获取我的 API Key").To(r.list).
		Returns(http.StatusOK, "成功", dtos.APITokenList{})).
		Metadata(config.FrontApiTag, "listMyAPITokens"))
	ws.Route(filtersForRoute(ws.POST(base).
		Doc("创建 API Key").Notes("明文 API Key 仅在本响应中返回一次").To(r.create).
		Reads(dtos.APITokenCreate{}).Returns(http.StatusOK, "成功", dtos.APITokenCreated{})).
		Metadata(config.FrontApiTag, "createMyAPIToken"))
	ws.Route(filtersForRoute(ws.PUT(base+"/{id}").
		Doc("更新我的 API Key").Param(ws.PathParameter("id", "API Key ID")).To(r.update).
		Reads(dtos.APITokenUpdate{}).Returns(http.StatusOK, "成功", dtos.APITokenDetail{})).
		Metadata(config.FrontApiTag, "updateMyAPIToken"))
	ws.Route(filtersForRoute(ws.DELETE(base+"/{id}").
		Doc("删除我的 API Key").Param(ws.PathParameter("id", "API Key ID")).To(r.delete).
		Returns(http.StatusOK, "成功", "success")).
		Metadata(config.FrontApiTag, "deleteMyAPIToken"))
}

func (r APITokenResource) list(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, errorData := r.Svc.List(ctx)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r APITokenResource) create(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	var input dtos.APITokenCreate
	if err := req.ReadEntity(&input); err != nil {
		writeControlError(ctx, req, resp, err)
		return
	}
	result, errorData := r.Svc.Create(ctx, input)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r APITokenResource) update(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	var input dtos.APITokenUpdate
	if err := req.ReadEntity(&input); err != nil {
		writeControlError(ctx, req, resp, err)
		return
	}
	result, errorData := r.Svc.Update(ctx, req.PathParameter("id"), input)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r APITokenResource) delete(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	errorData := r.Svc.Delete(ctx, req.PathParameter("id"))
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, "success")
}

func writeControlError(ctx context.Context, req *restful.Request, resp *restful.Response, err error) {
	common.ResponseErrorMessage(ctx, req, resp, config.Bundle, common.ErrorData{Err: err, ResponseCode: http.StatusBadRequest, MsgCode: config.MsgCodeJsonDecodeFailed})
}
