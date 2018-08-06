package migrations

import (
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		/*
			DEPRECATED
			if _, err := db.Exec( `CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`); err != nil {
				return err
			}

			if _, err := orm.CreateTable(db, &model.Storage{}, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true}); err != nil {
				return err
			}
		*/
		return nil
	}, func(db migrations.DB) error {
		/*
			DEPRECATED
			if _, err := orm.DropTable(db, &model.Storage{}, &orm.DropTableOptions{IfExists: true}); err != nil {
				return err
			}

			_, err := db.Exec(`DROP EXTENSION IF EXISTS "uuid-ossp"`)
			return err
		*/
		return nil
	})
}
