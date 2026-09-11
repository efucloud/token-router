package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/efucloud/token-router/pkg/apis/filters"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/efucloud/token-router/pkg/services"

	"github.com/efucloud/common"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	restful "github.com/emicklei/go-restful/v3"
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
	ws.Route(ws.POST(config.APIPrefix+apiExtend).
		Doc("创建用户").
		Notes("创建用户信息").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		To(r.create).
		Reads(dtos.AccountCreate{}).
		Returns(http.StatusOK, "成功", dtos.AccountDetail{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "createAccount"))
	ws.Route(ws.GET(config.APIPrefix+apiExtend).
		Doc("获取用户列表").
		Notes("获取用户列表").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		Param(ws.QueryParameter("current", "页码").DataType("number")).
		Param(ws.QueryParameter("pageSize", "每页大小").DataType("number")).
		Param(ws.QueryParameter("order", "排序")).
		Param(ws.QueryParameter("role", "系统角色")).
		Param(ws.QueryParameter("username", "账户名英文").DataType("string")).
		Param(ws.QueryParameter("nickname", "昵称").DataType("string")).
		Param(ws.QueryParameter("phone", "电话号码").DataType("string")).
		Param(ws.QueryParameter("email", "邮箱").DataType("string")).
		Param(ws.QueryParameter("jobNumber", "工号").DataType("string")).
		Param(ws.QueryParameter("search", "搜索").DataType("string")).
		Param(ws.QueryParameter("ids", "数据库记录ID数组,逗号分隔").DataType("string")).
		To(r.list).
		Returns(http.StatusOK, "成功", dtos.AccountDetailList{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "listAccount"))
	ws.Route(ws.GET(config.APIPrefix+apiExtend+"/{id}").
		Doc("获取用户详情").
		Notes("获取用户信息详情").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		To(r.get).
		Param(ws.PathParameter("id", "记录ID").DataType("string")).
		Returns(http.StatusOK, "成功", dtos.AccountDetail{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "getAccount"))
	ws.Route(ws.PUT(config.APIPrefix+apiExtend).
		Doc("更新用户信息").
		Notes("更新用户信息").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		To(r.update).
		Reads(dtos.AccountUpdate{}).
		Returns(http.StatusOK, "成功", dtos.AccountDetail{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "updateAccount"))
	ws.Route(ws.DELETE(config.APIPrefix+apiExtend).
		Doc("删除用户").
		Notes("删除用户信息详情").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		To(r.delete).
		Reads(dtos.BatchOperationIds{}).
		Returns(http.StatusOK, "成功", "成功").
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "deleteAccount"))
	ws.Route(ws.POST(config.APIPrefix+apiExtend+"/status").
		Doc("启用禁用").
		Notes("启用禁用,修改账户状态").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		To(r.status).
		Reads(dtos.AccountStatus{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "changeAccountStatus"))
	ws.Route(ws.POST(config.APIPrefix+apiExtend+"/role").
		Doc("系统角色设置").
		Notes("系统角色设置,admin: 管理员，view: 查看者， edit: 编辑者， none: 无权限，仅为系统成员").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		To(r.systemRole).
		Reads(dtos.AccountRole{}).
		Returns(http.StatusOK, "成功", dtos.AccountDetail{}).
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Returns(http.StatusInternalServerError, "内部处理逻辑错误", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "setAccountRole"))
	ws.Route(ws.PUT(config.APIPrefix+apiExtend+"/{id}/quota").
		Doc("更新用户聚合限额").
		Notes("只修改限额，不修改累计用量；0表示不限制").
		Param(ws.HeaderParameter(config.AuthHeader, "系统用户Token")).
		Param(ws.PathParameter("id", "用户ID").Required(true)).
		Reads(dtos.AccountQuotaUpdate{}).
		To(r.quota).
		Returns(http.StatusOK, "成功", "success").
		Returns(http.StatusBadRequest, "请求数据无法处理", dtos.ResponseError{}).
		Returns(http.StatusForbidden, "用户没有权限", dtos.ResponseError{}).
		Filter(filters.I18n).Filter(filters.Log).Filter(filters.Auth).Filter(filters.Permission([]string{"admin"})).
		Metadata(restfulspec.KeyOpenAPITags, apiInfo.Tags()).
		Metadata(config.FrontApiTag, "updateAccountQuota"))
}

func (r AccountResource) quota(req *restful.Request, resp *restful.Response) {
	ctx := context.Background()
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	var model dtos.AccountQuotaUpdate
	if err := req.ReadEntity(&model); err != nil {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, common.ErrorData{Err: err, ResponseCode: http.StatusBadRequest, MsgCode: config.MsgCodeJsonDecodeFailed})
		return
	}
	errorData := r.Svc.UpdateQuota(ctx, req.PathParameter("id"), model)
	if errorData.IsNotNil() {
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, "success")
}

func (r AccountResource) status(req *restful.Request, resp *restful.Response) {
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
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	errorData = r.Svc.ChangeStatusAccount(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("enable account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	common.ResponseSuccess(resp, "success")
}

func (r AccountResource) systemRole(req *restful.Request, resp *restful.Response) {
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
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

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
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	var (
		errorData common.ErrorData
		result    dtos.AccountDetail
	)
	errorData.Lang = lang

	ctx := context.Background()
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	id := req.PathParameter("id")

	result, errorData = r.Svc.GetAccountByID(ctx, id)
	if errorData.IsNotNil() {
		config.Logger.Errorf("get account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	common.ResponseSuccess(resp, result)
}
func (r AccountResource) delete(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
		model     dtos.BatchOperationIds
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	errorData = r.Svc.DeleteAccount(ctx, model.Ids)
	if errorData.IsNotNil() {
		config.Logger.Errorf("delete account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, "删除成功")
}
func (r AccountResource) create(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
		result    dtos.AccountDetail
		model     dtos.AccountCreate
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	//判断组织是否允许创建用户

	result, errorData = r.Svc.AddAccount(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("add account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	common.ResponseSuccess(resp, result)
}

func (r AccountResource) update(req *restful.Request, resp *restful.Response) {
	var (
		errorData common.ErrorData
		model     dtos.AccountUpdate
		result    dtos.AccountDetail
	)
	lang := common.GetLanguageFromReq(req, config.RequestLanguage)
	errorData.Lang = lang
	ctx := context.WithValue(context.Background(), config.RequestLanguage, lang)
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	errorData.Err = req.ReadEntity(&model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("decode json format data failed, err: %s", errorData.Err.Error())
		errorData.MsgCode = config.MsgCodeJsonDecodeFailed
		errorData.ResponseCode = http.StatusBadRequest
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}

	result, errorData = r.Svc.UpdateAccount(ctx, model)
	if errorData.IsNotNil() {
		config.Logger.Errorf("update account failed, err: %s", errorData.Err.Error())
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
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
	if reqCtx := req.Attribute(config.RequestContext); reqCtx != nil {
		ctx = reqCtx.(context.Context)
	}
	ctx = context.WithValue(ctx, config.RequestLanguage, lang)

	current, pageSize, order := common.GetRequestPaginationInformation(req)
	queryParam := &common.QueryParam{}

	common.RequestQuery("search:username;nickname;phone;email;jobNumber", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("username", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("role", common.ParamTypeString, common.QueryTypeEqual, req, queryParam)
	common.RequestQuery("phone", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("email", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	common.RequestQuery("nickname", common.ParamTypeString, common.QueryTypeLike, req, queryParam)
	if ids := strings.Split(req.QueryParameter("ids"), ","); len(ids) > 0 {
		if ids = common.FilterEmptyStrings(ids); len(ids) > 0 {
			common.QueryIn("id", ids, queryParam)
		}
	}
	result, errorData = r.Svc.ListAccount(ctx, current, pageSize, order, queryParam.WhereQuery, queryParam.WhereArgs)
	if errorData.IsNotNil() {
		config.Logger.Errorf("list account failed, err: %s", errorData.Err)
		errorData.Lang = lang
		common.ResponseErrorMessage(ctx, req, resp, config.Bundle, errorData)
		return
	}
	common.ResponseSuccess(resp, result)
}
