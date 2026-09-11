package migrations

import (
	"context"
	"fmt"
	"strings"

	"github.com/efucloud/token-router/pkg/config"
	"github.com/efucloud/token-router/pkg/models/daos"
)

type table interface {
	Indexes() (results map[string][]string)
	UniqueIndexes() (results map[string][]string)
	TableName() string
}

func DatabaseMigrate() {
	var migs []interface{}
	migs = []interface{}{
		&daos.Account{},
		&daos.AIModel{},
		&daos.Provider{},
		&daos.Channel{},
		&daos.ModelRoute{},
		&daos.APIToken{},
		&daos.UsageLog{},
		&daos.RouteAttempt{},
		&daos.APIRateLimitBucket{},
		&daos.ConcurrencyLease{},
		&daos.AuditLog{},
		&daos.ChatConversation{},
		&daos.ChatMessage{},
	}
	err := config.DBConnect.Migrator().AutoMigrate(migs...)
	if err != nil {
		config.Logger.Fatalf("migrate database failed, err: %s", err.Error())
	} else {
		config.Logger.Info("migrate database success")
	}
	ctx := context.Background()
	for _, item := range migs {
		t := item.(table)
		for name, idx := range t.Indexes() {
			if config.ApplicationConfig.Mysql != nil {
				if !mysqlHasIndex(ctx, t.TableName(), name) {
					sql := fmt.Sprintf("ALTER TABLE %s ADD INDEX %s (%s);", t.TableName(), name, strings.Join(idx, ","))
					err := config.DBConnect.Exec(sql).Error
					if err != nil {
						config.Logger.Fatalf("sql: [%s] exec failed, err:%s  ", sql, err.Error())
					}
				}
			}
		}
		for name, idx := range t.UniqueIndexes() {
			if config.ApplicationConfig.Mysql != nil {
				if !mysqlHasIndex(ctx, t.TableName(), name) {
					sql := fmt.Sprintf("ALTER TABLE %s ADD UNIQUE INDEX %s (%s);", t.TableName(), name, strings.Join(idx, ","))
					err := config.DBConnect.Exec(sql).Error
					if err != nil {
						config.Logger.Fatalf("sql: [%s] exec failed, err:%s  ", sql, err.Error())
					}
				}

			}
		}
	}
}

func hasColumn(ctx context.Context, tableName, column string) bool {
	var count int64
	_ = config.DBConnect.WithContext(ctx).Raw(
		"SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?;",
		config.DBConnect.Migrator().CurrentDatabase(), tableName, column,
	).Row().Scan(&count)
	return count > 0
}
func mysqlHasIndex(ctx context.Context, tableName, index string) bool {
	var count int64
	err := config.DBConnect.WithContext(ctx).Raw(
		"SELECT count(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE table_schema = ? AND table_name = ? AND INDEX_NAME = ?;",
		config.DBConnect.Migrator().CurrentDatabase(), tableName, index,
	).Row().Scan(&count)
	if err != nil {
		config.Logger.Fatal(err)
	}
	return count > 0
}
func postgresHasIndex(ctx context.Context, tableName, index string) bool {
	var count int64
	err := config.DBConnect.WithContext(ctx).Raw(
		"SELECT EXISTS ( SELECT 1 FROM pg_indexes WHERE tablename = ? AND indexname = ?);",
		config.DBConnect.Migrator().CurrentDatabase(), tableName, index,
	).Row().Scan(&count)
	if err != nil {
		config.Logger.Fatal(err)
	}
	return count > 0
}
