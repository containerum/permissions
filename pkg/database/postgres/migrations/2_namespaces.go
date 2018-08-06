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

		if _, err := orm.CreateTable(db, &model.Namespace{}, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true}); err != nil {
			return err
		}

		if _, err := db.Model(&model.Namespace{}).
			Exec( /* language=sql */ `CREATE UNIQUE INDEX unique_ns_owner_label ON "?TableName" ("owner_user_id", "label") WHERE NOT deleted`); err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {
		if _, err := db.Model(&model.Namespace{}).
			Exec( /* language=sql */ `DROP INDEX IF EXISTS unique_ns_owner_label`); err != nil {
			return err
		}

		if _, err := orm.DropTable(db, &model.Namespace{}, &orm.DropTableOptions{IfExists: true}); err != nil {
			return err
		}
		_, err := db.Exec( /* language=sql */ `DROP EXTENSION IF EXISTS "uuid-ossp"`)
		return err

		return nil
	})
}
