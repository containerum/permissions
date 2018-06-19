package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Permission{}).Exec( /* language=sql */
			`CREATE TYPE ACCESS_LEVEL_ AS ENUM ('none', 'guest', 'member', 'master', 'admin');
			ALTER TABLE "?TableName" ADD COLUMN IF NOT EXISTS initial_access_level_ ACCESS_LEVEL_,
									 ADD COLUMN IF NOT EXISTS current_access_level_ ACCESS_LEVEL_;
			UPDATE "?TableName" SET initial_access_level_ = CASE initial_access_level WHEN 'none' THEN 'none'::ACCESS_LEVEL_
																					  WHEN 'read' THEN 'guest'::ACCESS_LEVEL_
																					  WHEN 'readdelete' THEN 'member'::ACCESS_LEVEL_
																					  WHEN 'write' THEN 'master'::ACCESS_LEVEL_
																				  	  WHEN 'owner' THEN 'admin'::ACCESS_LEVEL_
														    END,
									current_access_level_ = CASE current_access_level WHEN 'none' THEN 'none'::ACCESS_LEVEL_
																				  	  WHEN 'read' THEN 'guest'::ACCESS_LEVEL_
																				  	  WHEN 'readdelete' THEN 'member'::ACCESS_LEVEL_
																				  	  WHEN 'write' THEN 'master'::ACCESS_LEVEL_
																				  	  WHEN 'owner' THEN 'admin'::ACCESS_LEVEL_
														    END;
			ALTER TABLE "?TableName" DROP COLUMN IF EXISTS current_access_level,
									 DROP COLUMN IF EXISTS initial_access_level;
			ALTER TABLE "?TableName" RENAME COLUMN current_access_level_ TO current_access_level;
			ALTER TABLE "?TableName" RENAME COLUMN initial_access_level_ TO initial_access_level;
			ALTER TABLE "?TableName" ALTER COLUMN current_access_level SET NOT NULL,
									 ALTER COLUMN initial_access_level SET NOT NULL;
			DROP TYPE IF EXISTS ACCESS_LEVEL;
			ALTER TYPE ACCESS_LEVEL_ RENAME TO ACCESS_LEVEL;`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Permission{}).Exec( /* language=sql */
			`CREATE TYPE ACCESS_LEVEL_ AS ENUM ('none', 'read', 'readdelete', 'write', 'owner');
			ALTER TABLE "?TableName" ADD COLUMN IF NOT EXISTS initial_access_level_ ACCESS_LEVEL_,
									 ADD COLUMN IF NOT EXISTS current_access_level_ ACCESS_LEVEL_;
			UPDATE "?TableName" SET initial_access_level_ = CASE initial_access_level WHEN 'none' THEN 'none'::ACCESS_LEVEL_
																					  WHEN 'guest' THEN 'read'::ACCESS_LEVEL_
																					  WHEN 'member' THEN 'readdelete'::ACCESS_LEVEL_
																					  WHEN 'master' THEN 'write'::ACCESS_LEVEL_
																				  	  WHEN 'admin' THEN 'owner'::ACCESS_LEVEL_
														    END,
									current_access_level_ = CASE current_access_level WHEN 'none' THEN 'none'::ACCESS_LEVEL_
																					  WHEN 'guest' THEN 'read'::ACCESS_LEVEL_
																					  WHEN 'member' THEN 'readdelete'::ACCESS_LEVEL_
																					  WHEN 'master' THEN 'write'::ACCESS_LEVEL_
																				  	  WHEN 'admin' THEN 'owner'::ACCESS_LEVEL_
														    END;
			ALTER TABLE "?TableName" DROP COLUMN IF EXISTS current_access_level,
									 DROP COLUMN IF EXISTS initial_access_level;
			ALTER TABLE "?TableName" RENAME COLUMN current_access_level_ TO current_access_level;
			ALTER TABLE "?TableName" RENAME COLUMN initial_access_level_ TO initial_access_level;
			ALTER TABLE "?TableName" ALTER COLUMN current_access_level SET NOT NULL,
									 ALTER COLUMN initial_access_level SET NOT NULL;
			DROP TYPE IF EXISTS ACCESS_LEVEL;
			ALTER TYPE ACCESS_LEVEL_ RENAME TO ACCESS_LEVEL;`)
		return err
	})
}
