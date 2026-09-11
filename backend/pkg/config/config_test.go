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
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		AdminEmails: []string{"admin@example.com"},
		Gateway:     GatewayConfig{MaxAttempts: 2},
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
}

func TestChatConfigEnvironmentOverrides(t *testing.T) {
	firstSkillDirectory := filepath.Join(t.TempDir(), "skills-one")
	secondSkillDirectory := filepath.Join(t.TempDir(), "skills-two")
	mcpConfigPath := filepath.Join(t.TempDir(), "mcp.yaml")
	mcpConfig := `mcpServers:
  - name: workspace
    type: stdio
    enabled: true
    command: /usr/bin/node
    args: [/efucloud/mcp/workspace/index.js]
    workingDir: /efucloud/mcp/workspace
`
	if err := os.WriteFile(mcpConfigPath, []byte(mcpConfig), 0o600); err != nil {
		t.Fatalf("write MCP config: %v", err)
	}
	t.Setenv(chatSkillDirectoriesEnv, strings.Join([]string{firstSkillDirectory, secondSkillDirectory}, string(os.PathListSeparator)))
	t.Setenv(chatMCPConfigFileEnv, mcpConfigPath)

	chat := ChatConfig{
		SkillDirectories: []string{"from-main-config"},
		MCPServers:       []ChatMCPConfig{{Name: "from-main-config"}},
	}
	if err := chat.applyEnvironment(); err != nil {
		t.Fatalf("apply chat environment: %v", err)
	}
	if len(chat.SkillDirectories) != 2 || chat.SkillDirectories[0] != firstSkillDirectory || chat.SkillDirectories[1] != secondSkillDirectory {
		t.Fatalf("unexpected Skill directories: %#v", chat.SkillDirectories)
	}
	if len(chat.MCPServers) != 1 || chat.MCPServers[0].Name != "workspace" || chat.MCPServers[0].Command != "/usr/bin/node" {
		t.Fatalf("unexpected MCP servers: %#v", chat.MCPServers)
	}
}

func TestChatConfigRejectsInvalidMCPConfigFile(t *testing.T) {
	mcpConfigPath := filepath.Join(t.TempDir(), "mcp.yaml")
	if err := os.WriteFile(mcpConfigPath, []byte("mcpServerz: []\n"), 0o600); err != nil {
		t.Fatalf("write MCP config: %v", err)
	}
	t.Setenv(chatMCPConfigFileEnv, mcpConfigPath)

	var chat ChatConfig
	if err := chat.applyEnvironment(); err == nil || !strings.Contains(err.Error(), "field mcpServerz not found") {
		t.Fatalf("expected strict MCP config error, got %v", err)
	}
}
