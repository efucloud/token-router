package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/efucloud/common"
	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/migrations"
	"github.com/efucloud/token-router/pkg/models/daos"
	"github.com/efucloud/token-router/pkg/utils"
)

func main() {
	configPath := flag.String("config", "./config/config.yaml", "configuration file")
	modelName := flag.String("model", "", "published model name")
	upstreamModel := flag.String("upstream-model", "", "upstream model name")
	baseURL := flag.String("base-url", "", "OpenAI-compatible upstream base URL")
	apiKeyEnvironment := flag.String("api-key-env", "UPSTREAM_API_KEY", "environment variable containing the upstream API key")
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	if !*migrateOnly && (*modelName == "" || *upstreamModel == "" || *baseURL == "") {
		fmt.Fprintln(os.Stderr, "model, upstream-model and base-url are required")
		os.Exit(2)
	}
	apiKey := os.Getenv(*apiKeyEnvironment)
	if !*migrateOnly && apiKey == "" {
		fmt.Fprintf(os.Stderr, "%s is required\n", *apiKeyEnvironment)
		os.Exit(2)
	}

	common.LoadConfig(*configPath, config.ApplicationConfig)
	config.ApplicationConfig.Init()
	migrations.DatabaseMigrate()
	if *migrateOnly {
		fmt.Println("database migration complete")
		return
	}

	encryptedAPIKey, err := utils.EncryptGatewayCredential(apiKey, config.ApplicationConfig.Gateway.SecretKey)
	if err != nil {
		fatal(err)
	}
	db := config.DBConnect

	account := daos.Account{}
	err = db.Where("username = ?", "local-admin").First(&account).Error
	if err != nil {
		account = daos.Account{
			ID: utils.GenerateDatabaseId(), CreatorId: "bootstrap", UpdaterId: "bootstrap",
			Username: "local-admin", Nickname: "Local Admin", Role: "admin", Enable: true,
			Email: "local-admin@token-router.local", Phone: "local-admin", Language: "zh",
			TokenLimit: 5_000_000, RequestLimit: 50_000,
		}
		if err = db.Create(&account).Error; err != nil {
			fatal(err)
		}
	}

	provider := daos.Provider{}
	if err = db.Where("name = ?", "bailian").First(&provider).Error; err != nil {
		provider = daos.Provider{GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, Name: "bailian", Type: "openai-compatible", Status: "enabled", Config: "{}"}
		if err = db.Create(&provider).Error; err != nil {
			fatal(err)
		}
	}

	channel := daos.Channel{}
	if err = db.Where("provider_id = ? AND name = ?", provider.ID, "bailian-default").First(&channel).Error; err != nil {
		channel = daos.Channel{GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, ProviderID: provider.ID, Name: "bailian-default"}
	}
	channel.BaseURL = *baseURL
	channel.EncryptedAPIKey = encryptedAPIKey
	channel.Priority = 100
	channel.Weight = 1
	channel.Status = "enabled"
	channel.TimeoutSeconds = 180
	channel.HealthStatus = "unknown"
	if err = db.Save(&channel).Error; err != nil {
		fatal(err)
	}

	model := daos.AIModel{}
	if err = db.Where("name = ?", *modelName).First(&model).Error; err != nil {
		model = daos.AIModel{GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, Name: *modelName}
	}
	model.DisplayName = "Qwen3 Coder 480B"
	model.Modality = "chat"
	model.ContextWindow = 262144
	model.Capabilities = `["streaming","tools","reasoning"]`
	model.Status = "active"
	if err = db.Save(&model).Error; err != nil {
		fatal(err)
	}

	route := daos.ModelRoute{}
	if err = db.Where("model_id = ? AND channel_id = ? AND upstream_model = ?", model.ID, channel.ID, *upstreamModel).First(&route).Error; err != nil {
		route = daos.ModelRoute{GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, ModelID: model.ID, ChannelID: channel.ID, UpstreamModel: *upstreamModel}
	}
	route.Priority = 100
	route.Weight = 1
	route.Status = "enabled"
	if err = db.Save(&route).Error; err != nil {
		fatal(err)
	}

	var token daos.APIToken
	if err = db.Where("account_id = ? AND name = ?", account.ID, "local-smoke-test").First(&token).Error; err == nil {
		fmt.Printf("bootstrap complete; existing API key prefix: %s\n", token.KeyPrefix)
		return
	}
	plaintext, hash, prefix, err := utils.GenerateAPIToken()
	if err != nil {
		fatal(err)
	}
	token = daos.APIToken{
		GatewayRecord: daos.GatewayRecord{ID: utils.GenerateDatabaseId()}, AccountID: account.ID,
		Name: "local-smoke-test", KeyHash: hash, KeyPrefix: prefix, Status: "active",
		AllowedModels: fmt.Sprintf("[%q]", *modelName), TokenLimit: 2_000_000, RequestLimit: 10_000,
		IPAllowlist: "[]",
	}
	if err = db.Create(&token).Error; err != nil {
		fatal(err)
	}
	fmt.Printf("bootstrap complete; API key (shown once): %s\n", plaintext)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
