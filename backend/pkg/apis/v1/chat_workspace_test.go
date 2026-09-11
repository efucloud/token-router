package v1

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/services"
	restful "github.com/emicklei/go-restful/v3"
)

func workspaceAPIRequest(method, target, contentType string, body *bytes.Buffer, accountID string) (*restful.Request, *restful.Response, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(method, target, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	restfulRequest := restful.NewRequest(request)
	ctx := context.WithValue(context.Background(), config.RequestUserId, accountID)
	restfulRequest.SetAttribute(config.RequestContext, ctx)
	recorder := httptest.NewRecorder()
	return restfulRequest, restful.NewResponse(recorder), recorder
}

func TestChatWorkspaceUploadListDownloadAndAccountIsolation(t *testing.T) {
	original := config.ApplicationConfig.Chat
	config.ApplicationConfig.Chat = config.ChatConfig{BuiltinTools: config.ChatBuiltinToolsConfig{
		Enabled: true, WorkspaceDirectory: t.TempDir(), MaxUploadBytes: 1024,
	}}
	config.ApplicationConfig.Chat.Default()
	t.Cleanup(func() { config.ApplicationConfig.Chat = original })
	resource := ChatResource{Svc: services.ChatService{}}

	var upload bytes.Buffer
	writer := multipart.NewWriter(&upload)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, _ = part.Write([]byte("hello workspace"))
	_ = writer.Close()
	req, resp, recorder := workspaceAPIRequest("POST", "/api/v1/chat/workspace/upload?path=", writer.FormDataContentType(), &upload, "user-a")
	resource.uploadWorkspace(req, resp)
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "hello.txt") {
		t.Fatalf("unexpected upload response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	req, resp, recorder = workspaceAPIRequest("GET", "/api/v1/chat/workspace?path=", "", &bytes.Buffer{}, "user-a")
	resource.listWorkspace(req, resp)
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), "hello.txt") {
		t.Fatalf("unexpected list response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	req, resp, recorder = workspaceAPIRequest("GET", "/api/v1/chat/workspace/download?path=hello.txt", "", &bytes.Buffer{}, "user-a")
	resource.downloadWorkspace(req, resp)
	if recorder.Code != 200 || recorder.Body.String() != "hello workspace" || !strings.Contains(recorder.Header().Get("Content-Disposition"), "hello.txt") {
		t.Fatalf("unexpected download response: status=%d disposition=%q body=%q", recorder.Code, recorder.Header().Get("Content-Disposition"), recorder.Body.String())
	}

	req, resp, recorder = workspaceAPIRequest("GET", "/api/v1/chat/workspace?path=", "", &bytes.Buffer{}, "user-b")
	resource.listWorkspace(req, resp)
	if recorder.Code != 200 || strings.Contains(recorder.Body.String(), "hello.txt") {
		t.Fatalf("user B saw user A file: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
