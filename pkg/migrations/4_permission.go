package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg/orm"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		if _, err := db.Exec( /* language=sql */ "CREATE TYPE RESOURCE_KIND AS ENUM ('namespace', 'volume')"); err != nil {
			return err
		}

		if _, err := db.Exec( /* language=sql */ "CREATE TYPE ACCESS_LEVEL AS ENUM ('none', 'read', 'readdelete', 'write', 'owner')"); err != nil {
			return err
		}

		if _, err := orm.CreateTable(db, &model.Permission{}, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true}); err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {
		if _, err := orm.DropTable(db, &model.Permission{}, &orm.DropTableOptions{IfExists: true}); err != nil {
			return err
		}

		if _, err := db.Exec( /* language=sql */ "DROP TYPE IF EXISTS ACCESS_LEVEL"); err != nil {
			return err
		}

		if _, err := db.Exec( /* language=sql */ "DROP TYPE IF EXISTS RESOURCE_KIND"); err != nil {
			return err
		}

		return nil
	})
}
