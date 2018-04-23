package model

import "github.com/go-pg/pg/orm"

// swagger:ignore
type Namespace struct {
	tableName struct{} `sql:"namespaces"`

	Resource

	RAM            int `sql:"ram,notnull"`
	CPU            int `sql:"cpu,notnull"`
	MaxExtServices int `sql:"max_ext_services,notnull"`
	MaxIntServices int `sql:"max_int_services,notnull"`
	MaxTraffic     int `sql:"max_traffic,notnull"`

	Volumes []*Volume `pg:"fk:ns_id" sql:"-"`
}

func (ns *Namespace) AfterInsert(db orm.DB) error {
	return db.Insert(&Permission{
		ResourceID:         ns.ID,
		UserID:             ns.OwnerUserID,
		ResourceKind:       "Namespace",
		InitialAccessLevel: AccessOwner,
		CurrentAccessLevel: AccessOwner,
	})
}
