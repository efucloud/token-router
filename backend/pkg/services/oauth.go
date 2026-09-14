package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"golang.org/x/oauth2"
)

type OAuthService struct {
}

type oidcAccountClaims struct {
	ID                string `json:"id"`
	EAuthID           string `json:"eAuthId"`
	Subject           string `json:"sub"`
	Username          string `json:"username"`
	PreferredUsername string `json:"preferred_username"`
	Nickname          string `json:"nickname"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	PhoneNumber       string `json:"phone_number"`
	Language          string `json:"language"`
	Locale            string `json:"locale"`
	Avatar            string `json:"avatar"`
	Picture           string `json:"picture"`
}

func firstOIDCValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func normalizeOIDCLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	switch {
	case strings.HasPrefix(language, "zh"):
		return "zh"
	case strings.HasPrefix(language, "en"):
		return "en-US"
	default:
		return ""
	}
}

func (claims oidcAccountClaims) accountCreate(expectedAccountID string, adminEmails []string) (dtos.AccountCreate, error) {
	accountID := firstOIDCValue(expectedAccountID, claims.EAuthID, claims.ID, claims.Subject)
	if accountID == "" {
		return dtos.AccountCreate{}, fmt.Errorf("OIDC userinfo does not contain a user identity")
	}
	email := strings.TrimSpace(claims.Email)
	username := firstOIDCValue(claims.Username, claims.PreferredUsername, email, claims.Subject, accountID)
	return dtos.AccountCreate{
		ID:       accountID,
		Username: username,
		Nickname: firstOIDCValue(claims.Nickname, claims.Name),
		Role:     oidcRoleForEmail(email, adminEmails),
		Email:    email,
		Phone:    firstOIDCValue(claims.Phone, claims.PhoneNumber),
		Language: normalizeOIDCLanguage(firstOIDCValue(claims.Language, claims.Locale)),
		Avatar:   firstOIDCValue(claims.Avatar, claims.Picture),
	}, nil
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

func (svc *OAuthService) accountFromToken(ctx context.Context, accessToken, expectedAccountID string) (create dtos.AccountCreate, errorData common.ErrorData) {
	if config.AuthProvider == nil {
		errorData.Err = fmt.Errorf("OIDC provider is not configured")
		return
	}
	if strings.TrimSpace(accessToken) == "" {
		errorData.Err = fmt.Errorf("OIDC access token is empty")
		return
	}

	userInfo, err := config.AuthProvider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
	}))
	if err != nil {
		errorData.Err = err
		return
	}
	var claims oidcAccountClaims
	if errorData.Err = userInfo.Claims(&claims); errorData.IsNotNil() {
		return
	}
	if claims.Subject == "" {
		claims.Subject = userInfo.Subject
	}
	var adminEmails []string
	if config.ApplicationConfig != nil {
		adminEmails = config.ApplicationConfig.AdminEmails
	}
	create, errorData.Err = claims.accountCreate(expectedAccountID, adminEmails)
	return
}

// ProvisionAccountFromToken creates the local account for a verified OIDC
// identity on its first API request. Existing accounts are deliberately not
// updated so that a disabled account cannot be re-enabled by userinfo data.
func (svc *OAuthService) ProvisionAccountFromToken(ctx context.Context, accessToken, accountID string) (result dtos.AccountDetail, errorData common.ErrorData) {
	var create dtos.AccountCreate
	create, errorData = svc.accountFromToken(ctx, accessToken, accountID)
	if errorData.IsNotNil() {
		return
	}
	ctx = context.WithValue(ctx, config.RequestUserId, create.ID)
	accountSvc := AccountService{}
	return accountSvc.CreateAccountIfAbsent(ctx, create)
}

func (svc *OAuthService) syncAccountFromToken(ctx context.Context, accessToken string) (result dtos.AccountDetail, errorData common.ErrorData) {
	var create dtos.AccountCreate
	create, errorData = svc.accountFromToken(ctx, accessToken, "")
	if errorData.IsNotNil() {
		return
	}
	ctx = context.WithValue(ctx, config.RequestUserId, create.ID)
	accountSvc := AccountService{}
	return accountSvc.CreateOrUpdateAccount(ctx, create)
}

func (svc *OAuthService) LoginByOIDC(ctx context.Context, loginParam dtos.LoginByOIDC) (response dtos.AccessTokenResponse, errorData common.ErrorData) {
	var (
		token *oauth2.Token
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
		_, errorData = svc.syncAccountFromToken(ctx, token.AccessToken)
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
