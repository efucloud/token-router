package v1

import (
	"context"
	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/services"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"net/http"
)

type AccountResource struct {
	Svc services.AccountService
}

func (r AccountResource) AddWebService(ws *restful.WebService) {
	apiInfo := common.ApiInfo{}
	apiInfo.Tag = "account"
	apiInfo.Description = "账户"
	common.RegisterApiInfo(apiInfo)
	apiExtend := "/account"
	ws.Route(ws.GET(config.APIPrefix+apiExtend).
		Doc("获取用户列表").
		Notes("获取用户信息").
		Param(ws.HeaderParameter(config.AuthHeader, "请求Token")).
		Param(ws.QueryParameter("page", "页码").DataType("integer")).
		Param(ws.QueryParameter("size", "每页大小").DataType("integer")).
		Param(ws.QueryParameter("order", "排序")).
		Param(ws.QueryParameter("username", "账户名英文").DataType("string")).
		Param(ws.QueryParameter("nickname", "昵称").DataType("string")).
		Param(ws.QueryParameter("phone", "电话号码").DataType("string")).
		Param(ws.QueryParameter("email", "邮箱").DataType("string")).
		To(r.list).
		Returns(http.StatusOK, "成功", dtos.AccountDetailList{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", common.AuthRedirectInfo{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", common.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", common.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "listAccount"))
	ws.Route(ws.GET(config.APIPrefix+apiExtend+"/{id}").
		Doc("获取用户详情").
		Notes("获取用户信息详情").
		Param(ws.HeaderParameter(config.AuthHeader, "请求Token")).
		To(r.get).
		Param(ws.PathParameter("id", "记录ID").DataType("string")).
		Returns(http.StatusOK, "成功", dtos.AccountDetail{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", common.AuthRedirectInfo{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", common.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", common.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "getAccount"))
	ws.Route(ws.DELETE(config.APIPrefix+apiExtend+"/{id}").
		Doc("删除用户").
		Notes("删除用户信息详情").
		Param(ws.HeaderParameter(config.AuthHeader, "请求Token")).
		To(r.delete).
		Param(ws.PathParameter("id", "记录ID").DataType("string")).
		Returns(http.StatusOK, "成功", "成功").
		Returns(http.StatusUnauthorized, "用户需要先登录", common.AuthRedirectInfo{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", common.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", common.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "deleteAccount"))
	ws.Route(ws.POST(config.APIPrefix+apiExtend+"/enable/{id}").
		Doc("启用禁用").
		Notes("启用禁用").
		Param(ws.PathParameter("id", "记录ID").DataType("string")).
		Param(ws.HeaderParameter(config.AuthHeader, "请求Token")).
		To(r.enable).
		Reads(dtos.AccountStatus{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", common.AuthRedirectInfo{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", common.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", common.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "enableAccount"))
	ws.Route(ws.POST(config.APIPrefix+apiExtend+"/role/{id}").
		Doc("组织角色设置").
		Notes("组织角色设置,admin: 管理员，view: 查看者， edit: 编辑者， none: 无权限，仅为组织成员").
		Param(ws.HeaderParameter(config.AuthHeader, "请求Token")).
		Param(ws.PathParameter("id", "记录ID").DataType("string")).
		To(r.orgRole).
		Reads(dtos.AccountRole{}).
		Returns(http.StatusOK, "成功", dtos.AccountDetail{}).
		Returns(http.StatusUnauthorized, "用户需要先登录", common.AuthRedirectInfo{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", common.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", common.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", common.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "setAccountRole"))

}

func (r AccountResource) enable(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
		model     dtos.AccountStatus
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	model.Ids = []string{req.PathParameter("id")}
	errorData = r.Svc.ChangeStatusAccount(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("enable account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	common.ResponseSuccess(resp, "success")
}

func (r AccountResource) orgRole(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
		model     dtos.AccountRole
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	model.Ids = []string{req.PathParameter("id")}
	errorData = r.Svc.SetRole(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("enable account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	common.ResponseSuccess(resp, "success")
}
func (r AccountResource) get(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	id := req.PathParameter("id")
	result, err := r.Svc.GetAccountByID(ctx, id)
	if !err.IsNil() || len(result.ID) == 0 {
		config.Logger.Errorf("getByCode account failed, err: %s", err.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	if len(result.ID) == 0 {
		errorData.MsgCode = config.MsgCodeRecordNotExist
		errorData.ResponseCode = http.StatusNotFound
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}
func (r AccountResource) delete(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	id := req.PathParameter("id")
	errorData = r.Svc.DeleteAccount(ctx, []string{id})
	if errorData.IsNotNil() {
		config.Logger.Errorf("delete account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, "删除成功")
}

func (r AccountResource) list(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
		result    dtos.AccountDetailList
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}

	page, size, order := common.GetRequestPaginationInformation(req)
	queryParam := &common.QueryParam{}
	common.RequestQuery("username", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("phone", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("email", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("nickname", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	result, errorData = r.Svc.ListAccount(ctx, page, size, order, queryParam.WhereQuery, queryParam.WhereArgs)
	if errorData.IsNotNil() {
		config.Logger.Errorf("list account failed, err: %s", errorData.Err)
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}
