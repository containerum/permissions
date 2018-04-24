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

// NamespaceAdminCreateRequest contains parameters for creating namespace without billing
//
// swagger:model
type NamespaceAdminCreateRequest struct {
	Label          string `json:"label" binding:"required"`
	CPU            int    `json:"cpu" binding:"required"`
	Memory         int    `json:"memory" binding:"required"`
	MaxExtServices int    `json:"max_ext_services" binding:"required"`
	MaxIntServices int    `json:"max_int_services" binding:"required"`
	MaxTraffic     int    `json:"max_traffic" binding:"required"`
}
