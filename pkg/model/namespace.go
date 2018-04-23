package model

import "github.com/go-pg/pg/orm"

// Namespace describes namespace
//
// swagger:model
type Namespace struct {
	tableName struct{} `sql:"namespaces"`

	Resource

	RAM            int `sql:"ram,notnull" json:"ram"`
	CPU            int `sql:"cpu,notnull" json:"cpu"`
	MaxExtServices int `sql:"max_ext_services,notnull" json:"max_external_services"`
	MaxIntServices int `sql:"max_int_services,notnull" json:"max_internal_services"`
	MaxTraffic     int `sql:"max_traffic,notnull" json:"max_traffic"`

	Volumes []*Volume `pg:"fk:ns_id" sql:"-" json:"volumes,omitempty"`
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
