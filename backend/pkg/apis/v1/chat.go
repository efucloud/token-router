package v1

import (
	"net/http"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/services"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
)

type ChatResource struct {
	Svc services.ChatService
}

func (r ChatResource) AddWebService(ws *restful.WebService) {
	apiInfo := common.ApiInfo{Tag: "chat-history", Description: "用户模型对话记录"}
	common.RegisterApiInfo(apiInfo)
	base := config.APIPrefix + "/chat/conversations"
	filtersForRoute := func(route *restful.RouteBuilder) *restful.RouteBuilder {
		return route.Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
			Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags())
	}
	ws.Route(filtersForRoute(ws.GET(base).Doc("获取我的对话记录").To(r.list).
		Returns(http.StatusOK, "成功", []dtos.ChatConversationSummary{})).
		Metadata(config.FrontApiTag, "listChatConversations"))
	ws.Route(filtersForRoute(ws.POST(base).Doc("新建对话").To(r.create).
		Reads(dtos.ChatConversationCreate{}).Returns(http.StatusOK, "成功", dtos.ChatConversationSummary{})).
		Metadata(config.FrontApiTag, "createChatConversation"))
	ws.Route(filtersForRoute(ws.GET(base+"/{id}").Doc("获取对话详情").To(r.get).
		Param(ws.PathParameter("id", "会话ID")).Returns(http.StatusOK, "成功", dtos.ChatConversationDetail{})).
		Metadata(config.FrontApiTag, "getChatConversation"))
	ws.Route(filtersForRoute(ws.PUT(base+"/{id}").Doc("更新对话").To(r.update).
		Param(ws.PathParameter("id", "会话ID")).Reads(dtos.ChatConversationUpdate{}).
		Returns(http.StatusOK, "成功", dtos.ChatConversationSummary{})).
		Metadata(config.FrontApiTag, "updateChatConversation"))
	ws.Route(filtersForRoute(ws.POST(base+"/{id}/messages").Doc("追加对话消息").To(r.appendMessage).
		Param(ws.PathParameter("id", "会话ID")).Reads(dtos.ChatMessageCreate{}).
		Returns(http.StatusOK, "成功", dtos.ChatMessageDetail{})).
		Metadata(config.FrontApiTag, "appendChatMessage"))
	ws.Route(filtersForRoute(ws.DELETE(base+"/{id}").Doc("删除对话").To(r.delete).
		Param(ws.PathParameter("id", "会话ID")).Returns(http.StatusOK, "成功", "success")).
		Metadata(config.FrontApiTag, "deleteChatConversation"))
}

func (r ChatResource) list(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, errorData := r.Svc.List(ctx)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ChatResource) create(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	var input dtos.ChatConversationCreate
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

func (r ChatResource) get(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, errorData := r.Svc.Get(ctx, req.PathParameter("id"))
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ChatResource) update(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	var input dtos.ChatConversationUpdate
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

func (r ChatResource) appendMessage(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	var input dtos.ChatMessageCreate
	if err := req.ReadEntity(&input); err != nil {
		writeControlError(ctx, req, resp, err)
		return
	}
	result, errorData := r.Svc.AppendMessage(ctx, req.PathParameter("id"), input)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ChatResource) delete(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	errorData := r.Svc.Delete(ctx, req.PathParameter("id"))
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, "success")
}
