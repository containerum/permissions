package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg/orm"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		if _, err := orm.CreateTable(db, &model.Namespace{}, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true}); err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {
		if _, err := orm.DropTable(db, &model.Namespace{}, &orm.DropTableOptions{IfExists: true}); err != nil {
			return err
		}

		return nil
	})
}
