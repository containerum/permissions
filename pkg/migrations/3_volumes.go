package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
	"github.com/go-pg/pg/orm"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		if _, err := orm.CreateTable(db, &model.Volume{}, &orm.CreateTableOptions{IfNotExists: true, FKConstraints: true}); err != nil {
			return err
		}

		// at the moment "go-pg" can create foreign keys only for "has one" relation
		if _, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" ADD CONSTRAINT namespace_fk FOREIGN KEY (ns_id) REFERENCES namespaces (id)`); err != nil {
			return err
		}

		if _, err := db.Model(&model.Volume{}).Exec( /* language=sql*/
			`ALTER TABLE "?TableName" ADD CONSTRAINT storage_fk FOREIGN KEY (storage_id) REFERENCES storages (id)`); err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {
		if _, err := db.Model(&model.Volume{}).Exec( /* language=sql*/
			`ALTER TABLE "?TableName" DROP CONSTRAINT namespace_fk`); err != nil {
			return err
		}

		if _, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP CONSTRAINT storage_fk`); err != nil {
			return err
		}

		if _, err := orm.DropTable(db, &model.Volume{}, &orm.DropTableOptions{IfExists: true}); err != nil {
			return err
		}

		return nil
	})
}
