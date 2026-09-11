package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/dtos"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

const maxChatSkillBytes = 1 << 20

var chatCapabilityName = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type chatSkillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func discoverChatSkills() ([]dtos.ChatSkillCapability, []string) {
	result := make([]dtos.ChatSkillCapability, 0)
	issues := make([]string, 0)
	seen := map[string]string{}
	for _, configured := range config.ApplicationConfig.Chat.SkillDirectories {
		directory := strings.TrimSpace(configured)
		if directory == "" {
			continue
		}
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			issues = append(issues, fmt.Sprintf("Skill directory %q is unavailable", directory))
			continue
		}
		err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				issues = append(issues, fmt.Sprintf("Cannot inspect Skill path %q", path))
				return nil
			}
			if entry.IsDir() || !strings.EqualFold(entry.Name(), "SKILL.md") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil || len(content) > maxChatSkillBytes {
				issues = append(issues, fmt.Sprintf("Skill file %q is unreadable or too large", path))
				return nil
			}
			meta, parseErr := parseChatSkillFrontmatter(string(content))
			if parseErr != nil || !chatCapabilityName.MatchString(meta.Name) {
				issues = append(issues, fmt.Sprintf("Skill file %q has invalid frontmatter", path))
				return nil
			}
			if existing, duplicate := seen[meta.Name]; duplicate {
				issues = append(issues, fmt.Sprintf("Skill %q is duplicated in %q and %q", meta.Name, existing, path))
				return nil
			}
			seen[meta.Name] = path
			result = append(result, dtos.ChatSkillCapability{Name: meta.Name, Description: meta.Description, Content: string(content)})
			return nil
		})
		if err != nil {
			issues = append(issues, fmt.Sprintf("Cannot scan Skill directory %q", directory))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, issues
}

func parseChatSkillFrontmatter(content string) (chatSkillFrontmatter, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return chatSkillFrontmatter{}, errors.New("frontmatter is missing")
	}
	end := strings.Index(normalized[4:], "\n---\n")
	if end < 0 {
		return chatSkillFrontmatter{}, errors.New("frontmatter is incomplete")
	}
	var result chatSkillFrontmatter
	if err := yaml.Unmarshal([]byte(normalized[4:4+end]), &result); err != nil {
		return chatSkillFrontmatter{}, err
	}
	result.Name = strings.TrimSpace(result.Name)
	result.Description = strings.TrimSpace(result.Description)
	if result.Name == "" || result.Description == "" {
		return chatSkillFrontmatter{}, errors.New("name and description are required")
	}
	return result, nil
}

func chatCapabilityNames() (map[string]struct{}, map[string]struct{}) {
	skills, _ := discoverChatSkills()
	skillNames := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		skillNames[skill.Name] = struct{}{}
	}
	mcpNames := make(map[string]struct{})
	for _, server := range config.ApplicationConfig.Chat.MCPServers {
		if server.Enabled && chatCapabilityName.MatchString(server.Name) {
			mcpNames[server.Name] = struct{}{}
		}
	}
	return skillNames, mcpNames
}

type chatHeaderTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t chatHeaderTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	for key, value := range t.headers {
		clone.Header.Set(key, value)
	}
	return t.base.RoundTrip(clone)
}

type chatMCPSession struct {
	configKey string
	session   *mcp.ClientSession
}

type chatMCPRegistry struct {
	mu       sync.Mutex
	sessions map[string]chatMCPSession
}

var chatMCP = chatMCPRegistry{sessions: map[string]chatMCPSession{}}

func chatMCPConfigKey(server config.ChatMCPConfig) string {
	value, _ := json.Marshal(server)
	return string(value)
}

func chatMCPTimeout(server config.ChatMCPConfig) time.Duration {
	if server.TimeoutSeconds > 0 {
		return time.Duration(server.TimeoutSeconds) * time.Second
	}
	return 30 * time.Second
}

func chatMCPSessionKey(accountID, serverName string) string {
	return accountID + "\x00" + serverName
}

func (r *chatMCPRegistry) session(ctx context.Context, accountID string, server config.ChatMCPConfig) (*mcp.ClientSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := chatMCPConfigKey(server)
	sessionKey := chatMCPSessionKey(accountID, server.Name)
	if cached, ok := r.sessions[sessionKey]; ok && cached.configKey == key {
		return cached.session, nil
	} else if ok {
		_ = cached.session.Close()
		delete(r.sessions, sessionKey)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "token-router", Title: "Token Router", Version: "0.1.0"}, nil)
	var transport mcp.Transport
	switch server.Type {
	case "streamable-http":
		if strings.TrimSpace(server.URL) == "" {
			return nil, errors.New("MCP URL is required")
		}
		transport = &mcp.StreamableClientTransport{
			Endpoint: server.URL,
			HTTPClient: &http.Client{
				Timeout:   chatMCPTimeout(server),
				Transport: chatHeaderTransport{base: http.DefaultTransport, headers: server.Headers},
			},
			DisableStandaloneSSE: true,
		}
	case "stdio":
		if strings.TrimSpace(server.Command) == "" {
			return nil, errors.New("MCP command is required")
		}
		command := exec.Command(server.Command, server.Args...)
		command.Dir = server.WorkingDir
		command.Env = os.Environ()
		for key, value := range server.Environment {
			command.Env = append(command.Env, key+"="+value)
		}
		transport = &mcp.CommandTransport{Command: command}
	default:
		return nil, fmt.Errorf("unsupported MCP transport %q", server.Type)
	}
	connectContext, cancel := context.WithTimeout(ctx, chatMCPTimeout(server))
	defer cancel()
	session, err := client.Connect(connectContext, transport, nil)
	if err != nil {
		return nil, err
	}
	r.sessions[sessionKey] = chatMCPSession{configKey: key, session: session}
	return session, nil
}

func (r *chatMCPRegistry) evict(accountID, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := chatMCPSessionKey(accountID, name)
	if cached, ok := r.sessions[key]; ok {
		_ = cached.session.Close()
		delete(r.sessions, key)
	}
}

func listChatMCPTools(ctx context.Context, accountID string, server config.ChatMCPConfig) ([]*mcp.Tool, error) {
	session, err := chatMCP.session(ctx, accountID, server)
	if err != nil {
		return nil, err
	}
	callContext, cancel := context.WithTimeout(ctx, chatMCPTimeout(server))
	defer cancel()
	tools := make([]*mcp.Tool, 0)
	cursor := ""
	for {
		page, listErr := session.ListTools(callContext, &mcp.ListToolsParams{Cursor: cursor})
		if listErr != nil {
			chatMCP.evict(accountID, server.Name)
			return nil, listErr
		}
		tools = append(tools, page.Tools...)
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	return tools, nil
}

func chatMCPServer(name string) (config.ChatMCPConfig, bool) {
	for _, server := range config.ApplicationConfig.Chat.MCPServers {
		if server.Enabled && server.Name == name && chatCapabilityName.MatchString(server.Name) {
			return server, true
		}
	}
	return config.ChatMCPConfig{}, false
}

func chatToolSchema(value any) map[string]any {
	encoded, err := json.Marshal(value)
	if err != nil {
		return map[string]any{"type": "object"}
	}
	result := map[string]any{}
	if json.Unmarshal(encoded, &result) != nil {
		return map[string]any{"type": "object"}
	}
	return result
}

func (ChatService) Capabilities(ctx context.Context) (dtos.ChatCapabilities, error) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return dtos.ChatCapabilities{}, err
	}
	settings := config.ApplicationConfig.Chat
	result := dtos.ChatCapabilities{
		Policy: dtos.ChatRuntimePolicy{
			ContextWindowTokens: settings.ContextWindowTokens, CompactThreshold: settings.CompactThreshold,
			CompactKeepRecent: settings.CompactKeepRecent, MaxRetries: settings.MaxRetries,
			RetryBaseMillis: settings.RetryBaseMillis, RetryMaxMillis: settings.RetryMaxMillis, MaxToolRounds: 8,
		},
	}
	result.Skills, result.Issues = discoverChatSkills()
	seen := map[string]struct{}{}
	for _, server := range settings.MCPServers {
		if !server.Enabled {
			continue
		}
		capability := dtos.ChatMCPServerCapability{Name: server.Name, Status: "connected", Tools: []dtos.ChatMCPToolCapability{}}
		if !chatCapabilityName.MatchString(server.Name) {
			capability.Status, capability.Error = "failed", "invalid server name"
			result.MCPServers = append(result.MCPServers, capability)
			continue
		}
		if _, duplicate := seen[server.Name]; duplicate {
			capability.Status, capability.Error = "failed", "duplicate server name"
			result.MCPServers = append(result.MCPServers, capability)
			continue
		}
		seen[server.Name] = struct{}{}
		tools, err := listChatMCPTools(ctx, accountID, server)
		if err != nil {
			capability.Status, capability.Error = "failed", err.Error()
		} else {
			for _, tool := range tools {
				capability.Tools = append(capability.Tools, dtos.ChatMCPToolCapability{
					Name: tool.Name, Description: tool.Description, InputSchema: chatToolSchema(tool.InputSchema),
				})
			}
			sort.Slice(capability.Tools, func(i, j int) bool { return capability.Tools[i].Name < capability.Tools[j].Name })
		}
		result.MCPServers = append(result.MCPServers, capability)
	}
	sort.Slice(result.MCPServers, func(i, j int) bool { return result.MCPServers[i].Name < result.MCPServers[j].Name })
	return result, nil
}

func (ChatService) CallMCPTool(ctx context.Context, serverName, toolName string, arguments map[string]any) (dtos.ChatMCPToolResult, error) {
	accountID, err := chatAccountID(ctx)
	if err != nil {
		return dtos.ChatMCPToolResult{}, err
	}
	server, ok := chatMCPServer(serverName)
	if !ok {
		return dtos.ChatMCPToolResult{}, errors.New("MCP server is not published")
	}
	tools, err := listChatMCPTools(ctx, accountID, server)
	if err != nil {
		return dtos.ChatMCPToolResult{}, err
	}
	found := false
	for _, tool := range tools {
		if tool.Name == toolName {
			found = true
			break
		}
	}
	if !found {
		return dtos.ChatMCPToolResult{}, errors.New("MCP tool is not published")
	}
	session, err := chatMCP.session(ctx, accountID, server)
	if err != nil {
		return dtos.ChatMCPToolResult{}, err
	}
	callContext, cancel := context.WithTimeout(ctx, chatMCPTimeout(server))
	defer cancel()
	result, err := session.CallTool(callContext, &mcp.CallToolParams{Name: toolName, Arguments: arguments})
	if err != nil {
		chatMCP.evict(accountID, server.Name)
		return dtos.ChatMCPToolResult{}, err
	}
	parts := make([]string, 0, len(result.Content)+1)
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
			continue
		}
		encoded, marshalErr := json.Marshal(content)
		if marshalErr == nil {
			parts = append(parts, string(encoded))
		}
	}
	if result.StructuredContent != nil {
		if encoded, marshalErr := json.Marshal(result.StructuredContent); marshalErr == nil {
			parts = append(parts, string(encoded))
		}
	}
	return dtos.ChatMCPToolResult{Content: strings.Join(parts, "\n"), IsError: result.IsError}, nil
}
