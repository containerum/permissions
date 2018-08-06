package migrations

import (
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		/*
			DEPRECATED
			_, err := db.Model(&model.Volume{}).Exec(
						`DELETE FROM "?TableName"
						WHERE "ns_id" IS NULL;
					ALTER TABLE "?TableName"
						DROP COLUMN IF EXISTS "replicas",
						DROP COLUMN IF EXISTS "active",
						ALTER COLUMN "ns_id" SET NOT NULL,
						ADD COLUMN IF NOT EXISTS "access_mode" TEXT NOT NULL`)
					return err
		*/
		return nil
	}, func(db migrations.DB) error {
		/*
			DEPRECATED
			_, err := db.Model(&model.Volume{}).Exec(
				`ALTER TABLE "?TableName"
					ADD COLUMN IF NOT EXISTS "replicas" INTEGER,
					ADD COLUMN IF NOT EXISTS "active" BOOLEAN,
					ALTER COLUMN "ns_id" DROP NOT NULL,
					DROP COLUMN IF EXISTS "access_mode"`)
			return err
		*/
		return nil
	})
}
