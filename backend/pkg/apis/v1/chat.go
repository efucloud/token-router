package v1

import (
	"errors"
	"mime"
	"net/http"
	"path/filepath"

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
	capabilities := config.APIPrefix + "/chat/capabilities"
	mcpTools := config.APIPrefix + "/chat/mcp/{server}/tools/{tool}/call"
	workspace := config.APIPrefix + "/chat/workspace"
	filtersForRoute := func(route *restful.RouteBuilder) *restful.RouteBuilder {
		return route.Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
			Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags())
	}
	ws.Route(filtersForRoute(ws.GET(base).Doc("获取我的对话记录").To(r.list).
		Returns(http.StatusOK, "成功", []dtos.ChatConversationSummary{})).
		Metadata(config.FrontApiTag, "listChatConversations"))
	ws.Route(filtersForRoute(ws.GET(capabilities).Doc("获取对话Skills与MCP能力").To(r.capabilities).
		Returns(http.StatusOK, "成功", dtos.ChatCapabilities{})).
		Metadata(config.FrontApiTag, "getChatCapabilities"))
	ws.Route(filtersForRoute(ws.POST(mcpTools).Doc("调用已发布的MCP工具").To(r.callMCPTool).
		Param(ws.PathParameter("server", "MCP服务名称")).Param(ws.PathParameter("tool", "工具名称")).
		Reads(dtos.ChatMCPToolCall{}).Returns(http.StatusOK, "成功", dtos.ChatMCPToolResult{})).
		Metadata(config.FrontApiTag, "callChatMCPTool"))
	ws.Route(filtersForRoute(ws.GET(workspace).Doc("浏览当前用户个人工作区").To(r.listWorkspace).
		Param(ws.QueryParameter("path", "用户工作区内的相对目录")).
		Returns(http.StatusOK, "成功", dtos.ChatWorkspaceListing{})).
		Metadata(config.FrontApiTag, "listChatWorkspace"))
	ws.Route(filtersForRoute(ws.POST(workspace+"/upload").Doc("上传文件到当前用户个人工作区").To(r.uploadWorkspace).
		Consumes("multipart/form-data").Param(ws.QueryParameter("path", "用户工作区内的相对目录")).
		Returns(http.StatusOK, "成功", dtos.ChatWorkspaceEntry{})).
		Metadata(config.FrontApiTag, "uploadChatWorkspaceFile"))
	ws.Route(filtersForRoute(ws.GET(workspace+"/download").Doc("下载当前用户个人工作区文件").To(r.downloadWorkspace).
		Produces("application/octet-stream").Param(ws.QueryParameter("path", "用户工作区内的相对文件路径"))).
		Metadata(config.FrontApiTag, "downloadChatWorkspaceFile"))
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

func workspaceServiceError(err error) common.ErrorData {
	return chatServiceError(err, services.WorkspaceHTTPStatus(err))
}

func (r ChatResource) listWorkspace(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, err := r.Svc.ListWorkspace(ctx, req.QueryParameter("path"))
	if err != nil {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, workspaceServiceError(err))
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ChatResource) uploadWorkspace(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	maxBytes := config.ApplicationConfig.Chat.BuiltinTools.MaxUploadBytes
	req.Request.Body = http.MaxBytesReader(resp.ResponseWriter, req.Request.Body, maxBytes+(1<<20))
	file, header, err := req.Request.FormFile("file")
	if err != nil {
		status := http.StatusBadRequest
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			status = http.StatusRequestEntityTooLarge
		}
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, chatServiceError(errors.New("invalid upload request"), status))
		return
	}
	defer file.Close()
	if req.Request.MultipartForm != nil {
		defer req.Request.MultipartForm.RemoveAll()
	}
	result, err := r.Svc.UploadWorkspaceFile(ctx, req.QueryParameter("path"), header.Filename, file)
	if err != nil {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, workspaceServiceError(err))
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ChatResource) downloadWorkspace(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	file, entry, err := r.Svc.OpenWorkspaceFile(ctx, req.QueryParameter("path"))
	if err != nil {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, workspaceServiceError(err))
		return
	}
	defer file.Close()
	contentType := mime.TypeByExtension(filepath.Ext(entry.Name))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	resp.Header().Set("Content-Type", contentType)
	resp.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": entry.Name}))
	resp.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(resp.ResponseWriter, req.Request, entry.Name, entry.UpdatedAt, file)
}

func (r ChatResource) capabilities(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	result, err := r.Svc.Capabilities(ctx)
	if err != nil {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, chatServiceError(err, http.StatusInternalServerError))
		return
	}
	common.ResponseSuccess(resp, result)
}

func (r ChatResource) callMCPTool(req *restful.Request, resp *restful.Response) {
	ctx := dashboardContext(req)
	var input dtos.ChatMCPToolCall
	if err := req.ReadEntity(&input); err != nil {
		writeControlError(ctx, req, resp, err)
		return
	}
	if input.Arguments == nil {
		input.Arguments = map[string]any{}
	}
	result, err := r.Svc.CallMCPTool(ctx, req.PathParameter("server"), req.PathParameter("tool"), input.Arguments)
	if err != nil {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, chatServiceError(err, http.StatusBadGateway))
		return
	}
	common.ResponseSuccess(resp, result)
}

func chatServiceError(err error, status int) common.ErrorData {
	return common.ErrorData{Err: err, ResponseCode: status, MsgCode: config.MsgCodeRequestDataInvalid}
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
