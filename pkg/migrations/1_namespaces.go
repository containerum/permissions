package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg/orm"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		if _, err := db.Exec( /* language=sql */ `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`); err != nil {
			return err
		}

		_, err := db.Model(&model.Namespace{}).CreateTable(&orm.CreateTableOptions{IfNotExists: true})
		return err
	}, func(db migrations.DB) error {
		if _, err := db.Model(&model.Namespace{}).DropTable(&orm.DropTableOptions{IfExists: true}); err != nil {
			return err
		}

		_, err := db.Exec( /* language=sql */ `DROP EXTENSION IF EXISTS "uuid-ossp"`)
		return err
	})
}
