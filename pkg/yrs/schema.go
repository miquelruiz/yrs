package yrs

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/sqlite"
)

// Can't use ".." in an embed pattern, so migrations need to be in this same
// directory

//go:generate cp -r ../../db ./
//go:embed db/migrations/*.sql
var migrationsFS embed.FS

type nullWriter struct{}

func (*nullWriter) Write(_ []byte) (int, error) { return 0, nil }

func ManageSchema(driver, dsn string) (*sql.DB, error) {
	u, err := url.Parse(fmt.Sprintf("%s:%s", driver, strings.TrimPrefix(dsn, "file:")))
	if err != nil {
		return nil, fmt.Errorf("error managing database schema: %v", err)
	}

	dbm := dbmate.New(u)
	dbm.FS = migrationsFS
	dbm.AutoDumpSchema = false
	dbm.Log = &nullWriter{}

	err = dbm.CreateAndMigrate()
	if err != nil {
		return nil, err
	}

	return sql.Open(driver, dsn)
}
