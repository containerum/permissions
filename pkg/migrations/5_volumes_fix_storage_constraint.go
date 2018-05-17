package migrations

import (
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" 
				  	DROP CONSTRAINT storage_fk,
				  	ADD CONSTRAINT storage_fk FOREIGN KEY (storage_id)
				  		REFERENCES storages (id)
				  		ON UPDATE CASCADE
				  		ON DELETE CASCADE
				  		DEFERRABLE
				  		INITIALLY DEFERRED`)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Model(&model.Volume{}).Exec( /* language=sql */
			`ALTER TABLE "?TableName" 
				  	DROP CONSTRAINT storage_fk,
				  	ADD CONSTRAINT storage_fk FOREIGN KEY (storage_id)
				  		REFERENCES storages (id)`)
		return err
	})
}
