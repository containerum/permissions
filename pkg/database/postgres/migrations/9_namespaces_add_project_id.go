package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Namespace{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id)`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Namespace{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP COLUMN IF EXISTS project_id`)
		return err
	})
}
