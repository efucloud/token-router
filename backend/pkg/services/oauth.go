package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"golang.org/x/oauth2"
)

type OAuthService struct {
}

func oidcRoleForEmail(email string, adminEmails []string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return "none"
	}
	for _, adminEmail := range adminEmails {
		adminEmail = strings.TrimSpace(adminEmail)
		if adminEmail != "" && strings.EqualFold(email, adminEmail) {
			return "admin"
		}
	}
	return "none"
}

func (svc *OAuthService) Userinfo(ctx context.Context, userId string) (userinfo dtos.AuthedUserInfo, errorData common.ErrorData) {
	accSvc := AccountService{}
	var acc dtos.AccountDetail
	acc, errorData = accSvc.GetAccountByID(ctx, userId)
	data, _ := json.Marshal(acc)
	_ = json.Unmarshal(data, &userinfo)
	return
}

func (svc *OAuthService) LoginByOIDC(ctx context.Context, loginParam dtos.LoginByOIDC) (response dtos.AccessTokenResponse, errorData common.ErrorData) {
	var (
		token      *oauth2.Token
		userInfo   *oidc.UserInfo
		systemUser dtos.AuthedUserInfo
	)
	config.Logger.Infof("LoginByOIDC code: %s", loginParam.Code)
	oauthCfg := &oauth2.Config{
		ClientID:     config.ApplicationConfig.OidcConfig.ClientId,
		ClientSecret: config.ApplicationConfig.OidcConfig.ClientSecret,
		Endpoint:     config.AuthProvider.Endpoint(),
		RedirectURL:  loginParam.RedirectUri,
	}
	token, errorData.Err = oauthCfg.Exchange(ctx, loginParam.Code)
	if errorData.IsNotNil() {
		config.Logger.Error(errorData.Err)
		return
	} else {
		userInfo, errorData.Err = config.AuthProvider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token.AccessToken,
			TokenType:   "Bearer", // The UserInfo endpoint requires a bearer token as per RFC6750
		}))
		var (
			create dtos.AccountCreate
		)

		accSvc := AccountService{}
		errorData.Err = userInfo.Claims(&create)
		if errorData.IsNotNil() {
			config.Logger.Error(errorData.Err)
			return
		}
		create.Role = oidcRoleForEmail(create.Email, config.ApplicationConfig.AdminEmails)

		_, errorData = accSvc.CreateOrUpdateAccount(ctx, create)
		if errorData.IsNotNil() {
			config.Logger.Error(errorData.Err)
			return
		}
		errorData.Err = userInfo.Claims(&systemUser)
		if errorData.IsNotNil() {
			config.Logger.Error(errorData.Err)
			return
		}
	}

	response.AccessToken = token.AccessToken
	response.RefreshToken = token.RefreshToken
	response.ExpiresIn = token.Expiry.Unix()
	return
}
