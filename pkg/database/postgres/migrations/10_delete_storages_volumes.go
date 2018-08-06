package migrations

import (
	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		if _, err := db.Exec(`DROP TABLE IF EXISTS volumes`); err != nil {
			return err
		}
		if _, err := db.Exec(`DROP TABLE IF EXISTS storages`); err != nil {
			return err
		}
		return nil
	}, func(db migrations.DB) error {
		return nil
	})
}
