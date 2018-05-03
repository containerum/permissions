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
	cnt, err := db.Model(v).
		Where("owner_user_id = ?owner_user_id").
		Where("label = ?label").
		Where("NOT deleted").
		Count()
	if err != nil {
		return err
	}

	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("volume %s already exists", v.Label)
	}

	return nil
}

func (v *Volume) AfterUpdate(db orm.DB) error {
	if err := v.Resource.AfterUpdate(db); err != nil {
		return err
	}

	var err error
	if v.Deleted {
		_, err = db.Model(&Storage{ID: v.StorageID}).
			WherePK().
			Set("used = used - ?", v.Capacity).
			Update()
	} else {
		oldCapacityQuery := db.Model(v).Column("capacity").WherePK()
		_, err = db.Model(&Storage{ID: v.StorageID}).
			WherePK().
			Set("used = used - (?) + ?", oldCapacityQuery, v.Capacity).
			Update(v)
	}
	return err
}

func (v *Volume) AfterInsert(db orm.DB) error {
	err := db.Insert(&Permission{
		ResourceID:         v.ID,
		UserID:             v.OwnerUserID,
		ResourceKind:       "Volume",
		InitialAccessLevel: AccessOwner,
		CurrentAccessLevel: AccessOwner,
	})
	if err != nil {
		return err
	}

	_, err = db.Model((*Storage)(nil)).
		Where("id", v.StorageID).
		Set("used = used + (?)", v.Capacity).
		Update()

	return err
}

func (v *Volume) Mask() {
	v.Resource.Mask()
	v.Active = nil
	v.Replicas = 0
	v.NamespaceID = nil
	v.GlusterName = ""
	v.StorageID = ""
}

// VolumeWithPermissions is a response object for get requests
//
// swagger:model
type VolumeWithPermissions struct {
	Volume `pg:",override"`

	Permission

	Permissions []Permission `pg:"polymorphic:resource_" sql:"-" json:"users,omitempty"`
}

func (vp *VolumeWithPermissions) Mask() {
	vp.Volume.Mask()
	vp.Permission.Mask()
	if vp.OwnerUserID != vp.Permission.UserID {
		vp.Permissions = nil
	}
}
