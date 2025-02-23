package dsDb

import (
	"context"

	commonDB "github.com/ottin4ttc/go_common/db"
)

func InitAwsPostgreSQL(ctx context.Context, cfg commonDB.PostgreSQLConfig) error {
	return commonDB.InitPostgreSQL(cfg)
}
