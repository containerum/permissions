package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`UPDATE "?TableName" 
			SET "ns_id" = '00000000-0000-0000-0000-000000000000'
			WHERE "ns_id" IS NULL;
		ALTER TABLE "?TableName" 
			DROP COLUMN IF EXISTS "replicas",
			DROP COLUMN IF EXISTS "active",
			ALTER COLUMN "ns_id" SET NOT NULL,
			ADD COLUMN IF NOT EXISTS "access_mode" TEXT NOT NULL`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName"
				ADD COLUMN IF NOT EXISTS "replicas" INTEGER,
				ADD COLUMN IF NOT EXISTS "active" BOOLEAN,
				ALTER COLUMN "ns_id" DROP NOT NULL,
				DROP COLUMN IF EXISTS "access_mode";
			UPDATE "?TableName"
				SET "ns_id" = NULL
				WHERE "ns_id" = '00000000-0000-0000-0000-000000000000'`)
		return err
	})
}
