package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Namespace{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" ADD COLUMN IF NOT EXISTS "kube_name" TEXT NOT NULL DEFAULT '';
			UPDATE "?TableName" SET kube_name = id::TEXT WHERE kube_name = '';
			ALTER TABLE "?TableName" ADD CONSTRAINT unique_kube_name UNIQUE (kube_name)`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Namespace{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" DROP COLUMN IF EXISTS "kube_name"`)
		return err
	})
}
