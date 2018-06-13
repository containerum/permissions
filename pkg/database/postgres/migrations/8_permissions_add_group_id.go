package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Permission{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" ADD COLUMN IF NOT EXISTS "group_id" UUID`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Permission{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP COLUMN IF EXISTS "group_id"`)
		return err
	})
}
