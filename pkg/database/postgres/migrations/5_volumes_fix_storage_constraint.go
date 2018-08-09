package migrations

import (
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		/*
			DEPRECATED
			_, err := db.Model(&model.Volume{}).Exec(
				`ALTER TABLE "?TableName"
						  DROP CONSTRAINT storage_fk,
						  ADD CONSTRAINT storage_fk FOREIGN KEY (storage_id)
							  REFERENCES storages (id)
							  ON UPDATE CASCADE
							  ON DELETE CASCADE
							  DEFERRABLE
							  INITIALLY DEFERRED`)
			return err
		*/
		return nil
	}, func(db migrations.DB) error {
		/*
			DEPRECATED
			_, err := db.Model(&model.Volume{}).Exec(
				`ALTER TABLE "?TableName"
						  DROP CONSTRAINT storage_fk,
						  ADD CONSTRAINT storage_fk FOREIGN KEY (storage_id)
							  REFERENCES storages (id)`)
			return err
		*/
		return nil
	})
}
