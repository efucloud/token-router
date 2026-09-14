/*
Copyright 2022 The itcloudy.com Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigSerialization(t *testing.T) {
	config := &Config{

		LogConfig: &LogConfig{
			Level:      "debug",
			Filename:   "",
			MaxSize:    0,
			MaxAge:     0,
			MaxBackups: 0,
			LocalTime:  false,
			Compress:   false,
			Production: true,
		},
		OidcConfig: OidcConfig{
			Issuer:            "https://sso.example.com",
			ClientId:          "test-client",
			ClientSecret:      "test-secret",
			SkipClientIDCheck: true,
		},
		Mysql: &MysqlConfig{
			Host:                      "localhost:3306",
			User:                      "root",
			Password:                  "test-password",
			Dbname:                    "llm_route",
			Charset:                   "utf8mb4",
			Loc:                       "",
			DefaultStringSize:         0,
			DisableDatetimePrecision:  false,
			DontSupportRenameIndex:    false,
			DontSupportRenameColumn:   false,
			SkipInitializeWithVersion: false,
		},
		Redis:       &RedisConfig{Enabled: true, Address: "redis.example.com:6379", DB: 2, RouteTTLSeconds: 45},
		AdminEmails: []string{"admin@example.com"},
		Gateway:     GatewayConfig{MaxAttempts: 2},
		Chat: ChatConfig{
			SkillDirectories: []string{"/srv/token-router/skills"},
			MCPServers: []ChatMCPConfig{{
				Name:    "knowledge",
				Type:    "streamable-http",
				Enabled: true,
				URL:     "http://knowledge-mcp:3000/mcp",
			}},
			BuiltinTools: ChatBuiltinToolsConfig{
				Enabled:            true,
				WorkspaceDirectory: "/srv/token-router/workspaces",
				MaxUploadBytes:     2048,
			},
		},
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	var decoded Config
	if err = yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if decoded.Gateway.MaxAttempts != 2 || decoded.OidcConfig.ClientId != config.OidcConfig.ClientId || !decoded.OidcConfig.SkipClientIDCheck {
		t.Fatalf("unexpected round trip result: %#v", decoded)
	}
	if decoded.Redis == nil || !decoded.Redis.Enabled || decoded.Redis.Address != "redis.example.com:6379" || decoded.Redis.DB != 2 || decoded.Redis.RouteTTLSeconds != 45 {
		t.Fatalf("unexpected Redis config: %#v", decoded.Redis)
	}
	if len(decoded.Chat.SkillDirectories) != 1 || decoded.Chat.SkillDirectories[0] != "/srv/token-router/skills" {
		t.Fatalf("unexpected Skill directories: %#v", decoded.Chat.SkillDirectories)
	}
	if len(decoded.Chat.MCPServers) != 1 || decoded.Chat.MCPServers[0].Name != "knowledge" || decoded.Chat.MCPServers[0].URL != "http://knowledge-mcp:3000/mcp" {
		t.Fatalf("unexpected MCP servers: %#v", decoded.Chat.MCPServers)
	}
	if !decoded.Chat.BuiltinTools.Enabled || decoded.Chat.BuiltinTools.WorkspaceDirectory != "/srv/token-router/workspaces" || decoded.Chat.BuiltinTools.MaxUploadBytes != 2048 {
		t.Fatalf("unexpected builtin tools: %#v", decoded.Chat.BuiltinTools)
	}
}

func TestRedisConfigDefaults(t *testing.T) {
	var redisConfig RedisConfig
	redisConfig.Default()
	if redisConfig.Enabled {
		t.Fatal("Redis must remain disabled unless explicitly enabled")
	}
	if redisConfig.Address != "127.0.0.1:6379" || redisConfig.KeyPrefix != "token-router" || redisConfig.RouteTTLSeconds != 30 {
		t.Fatalf("unexpected Redis defaults: %#v", redisConfig)
	}
	if redisConfig.DialTimeoutMillis <= 0 || redisConfig.ReadTimeoutMillis <= 0 || redisConfig.WriteTimeoutMillis <= 0 {
		t.Fatalf("Redis timeouts must be bounded: %#v", redisConfig)
	}
}

func TestChatConfigDefaults(t *testing.T) {
	var chat ChatConfig
	chat.Default()
	if chat.ContextWindowTokens != 32768 || chat.CompactThreshold != 0.75 || chat.CompactKeepRecent != 8 {
		t.Fatalf("unexpected compaction defaults: %#v", chat)
	}
	if chat.MaxRetries != 3 || chat.RetryBaseMillis != 800 || chat.RetryMaxMillis != 8000 {
		t.Fatalf("unexpected retry defaults: %#v", chat)
	}
	if chat.BuiltinTools.WorkspaceDirectory != "workspace" || len(chat.BuiltinTools.AllowedRoles) != 1 || chat.BuiltinTools.AllowedRoles[0] != "admin" || chat.BuiltinTools.CommandTimeoutSeconds != 120 || chat.BuiltinTools.MaxOutputBytes != 1<<20 || chat.BuiltinTools.MaxUploadBytes != 32<<20 {
		t.Fatalf("unexpected builtin tool defaults: %#v", chat.BuiltinTools)
	}
}
