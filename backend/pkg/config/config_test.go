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

	"github.com/redis/go-redis/v9"
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
		Redis:       &RedisConfig{Enabled: true, Mode: "sentinel", Addresses: []string{"redis-1.example.com:26379", "redis-2.example.com:26379"}, Password: "redis-password", MasterName: "router-master"},
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
	if decoded.Redis == nil || !decoded.Redis.Enabled || decoded.Redis.Mode != "sentinel" || len(decoded.Redis.Addresses) != 2 || decoded.Redis.Password != "redis-password" || decoded.Redis.MasterName != "router-master" {
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
	if redisConfig.Mode != "standalone" || len(redisConfig.Addresses) != 1 || redisConfig.Addresses[0] != "127.0.0.1:6379" {
		t.Fatalf("unexpected Redis defaults: %#v", redisConfig)
	}
}

func TestRedisMasterModeAliasesStandalone(t *testing.T) {
	redisConfig := RedisConfig{Mode: "master", Addresses: []string{"redis-master:6379"}}
	redisConfig.Default()
	if redisConfig.Mode != "standalone" {
		t.Fatalf("master mode must normalize to standalone: %#v", redisConfig)
	}
}

func TestNewRedisClientModes(t *testing.T) {
	tests := []struct {
		name       string
		config     RedisConfig
		clientType string
	}{
		{name: "standalone", config: RedisConfig{Mode: "standalone", Addresses: []string{"redis:6379"}}, clientType: "client"},
		{name: "sentinel", config: RedisConfig{Mode: "sentinel", Addresses: []string{"sentinel-1:26379", "sentinel-2:26379"}, MasterName: "router-master"}, clientType: "client"},
		{name: "cluster", config: RedisConfig{Mode: "cluster", Addresses: []string{"redis-1:6379", "redis-2:6379"}}, clientType: "cluster"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := newRedisClient(&test.config)
			if err != nil {
				t.Fatalf("create %s Redis client: %v", test.name, err)
			}
			t.Cleanup(func() { _ = client.Close() })
			switch test.clientType {
			case "client":
				if _, ok := client.(*redis.Client); !ok {
					t.Fatalf("expected redis.Client, got %T", client)
				}
			case "cluster":
				if _, ok := client.(*redis.ClusterClient); !ok {
					t.Fatalf("expected redis.ClusterClient, got %T", client)
				}
			}
		})
	}
}

func TestNewRedisClientRejectsInvalidMode(t *testing.T) {
	if _, err := newRedisClient(&RedisConfig{Mode: "sentinel", Addresses: []string{"sentinel:26379"}}); err == nil {
		t.Fatal("sentinel mode must require masterName")
	}
	if _, err := newRedisClient(&RedisConfig{Mode: "unknown"}); err == nil {
		t.Fatal("unsupported Redis mode must fail validation")
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
	if chat.BuiltinTools.WorkspaceDirectory != "workspace" || chat.BuiltinTools.CommandTimeoutSeconds != 120 || chat.BuiltinTools.MaxOutputBytes != 1<<20 || chat.BuiltinTools.MaxUploadBytes != 32<<20 {
		t.Fatalf("unexpected builtin tool defaults: %#v", chat.BuiltinTools)
	}
}
