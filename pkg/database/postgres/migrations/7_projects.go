package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg/orm"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := orm.CreateTable(db, &model.Project{}, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true})
		return err
	}, func(db migrations.DB) error {
		_, err := orm.DropTable(db, &model.Project{}, &orm.DropTableOptions{IfExists: true})
		return err
	})
}
