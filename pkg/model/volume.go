package model

import (
	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/go-pg/pg/orm"
)

// Volume describes volume
//
// swagger:model
type Volume struct {
	tableName struct{} `sql:"volumes"`

	Resource

	Active *bool `sql:"active,notnull" json:"active,omitempty"`

	Capacity int `sql:"capacity,notnull" json:"capacity"`

	Replicas int `sql:"replicas,notnull" json:"replicas"`

	// swagger:strfmt uuid
	NamespaceID *string `sql:"ns_id,type:UUID" json:"namespace_id,omitempty"`

	GlusterName string `sql:"gluster_name,notnull" json:"gluster_name,omitempty"`

	// swagger:strfmt uuid
	StorageID string `sql:"storage_id,type:UUID,notnull" json:"storage_id,omitempty"`
}

func (v *Volume) BeforeInsert(db orm.DB) error {
	cnt, err := db.Model(v).WherePK().Where("NOT deleted").Count()
	if err != nil {
		return err
	}

	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("namespace %s already exists", v.Label)
	}

	return nil
}

func (v *Volume) AfterUpdate(db orm.DB) error {
	if err := v.Resource.AfterUpdate(db); err != nil {
		return err
	}

	var err error
	if v.Deleted {
		_, err = db.Model(&Storage{}).
			Relation("Volumes").
			Set("used = used - ?", v.Capacity).
			Update()
	} else {
		oldCapacityQuery := db.Model(v).Column("capacity").WherePK()
		_, err = db.Model(&Storage{}).
			Set("used = used - (?) + ?", oldCapacityQuery, v.Capacity).
			Update(v)
	}
	return err
}

func (v *Volume) AfterInsert(db orm.DB) error {
	return db.Insert(&Permission{
		ResourceID:         v.ID,
		UserID:             v.OwnerUserID,
		ResourceKind:       "Volume",
		InitialAccessLevel: AccessOwner,
		CurrentAccessLevel: AccessOwner,
	})
}

func (v *Volume) Mask() {
	v.Resource.Mask()
	v.Active = nil
	v.Replicas = 0
	v.NamespaceID = nil
	v.GlusterName = ""
	v.StorageID = ""
}
