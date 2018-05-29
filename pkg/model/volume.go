package model

import (
	"time"

	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/containerum/kube-client/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

// Volume describes volume
//
// swagger:model
type Volume struct {
	tableName struct{} `sql:"volumes"`

	Resource

	Capacity int `sql:"capacity,notnull" json:"capacity"`

	// swagger:strfmt uuid
	NamespaceID string `sql:"ns_id,type:uuid" json:"namespace_id,omitempty"`

	GlusterName string `sql:"gluster_name,notnull" json:"gluster_name,omitempty"`

	// swagger:strfmt uuid
	StorageID string `sql:"storage_id,type:uuid,notnull" json:"storage_id,omitempty"`

	StorageName string `sql:"-" json:"storage_name,omitempty"`

	AccessMode model.PersistentVolumeAccessMode `sql:"access_mode,notnull" json:"access_mode,omitempty"`
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

	_, err = db.Model(&Storage{ID: v.StorageID}).
		WherePK().
		Set("used = used + (?)", v.Capacity).
		Update()

	return err
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
	return db.Insert(&Permission{
		ResourceID:         v.ID,
		UserID:             v.OwnerUserID,
		ResourceType:       ResourceVolume,
		InitialAccessLevel: model.Owner,
		CurrentAccessLevel: model.Owner,
	})
}

func (v *Volume) AfterSelect(db orm.DB) error {
	return db.Model(&Storage{ID: v.StorageID}).
		Column("name").
		WherePK().
		Select(pg.Scan(&v.StorageName))
}

func (v *Volume) Mask() {
	v.Resource.Mask()
	v.NamespaceID = ""
	v.GlusterName = ""
	v.StorageID = ""
	v.AccessMode = ""
}

// VolumeWithPermissions is a response object for get requests
//
// swagger:model VolumeWithPermissions
type VolumeWithPermissions struct {
	Volume `pg:",override"`

	Permission Permission `pg:"fk:resource_id" sql:"-" json:",inline"`

	Permissions []Permission `pg:"polymorphic:resource_" sql:"-" json:"users"`
}

func (vp *VolumeWithPermissions) ToKube() model.Volume {
	vol := model.Volume{
		ID:         vp.ID,
		CreatedAt:  new(string),
		Owner:      vp.OwnerUserID,
		OwnerLogin: vp.OwnerUserLogin,
		Label:      vp.Label,
		Access:     vp.Permission.CurrentAccessLevel,
		Capacity:   uint(vp.Capacity),
		Users:      make([]model.UserAccess, len(vp.Permissions)),
	}
	*vol.CreatedAt = vp.CreateTime.Format(time.RFC3339)
	for i, v := range vp.Permissions {
		vol.Users[i] = model.UserAccess{
			Username:    v.UserLogin,
			AccessLevel: v.CurrentAccessLevel,
		}
	}
	return vol
}

func (vp *VolumeWithPermissions) Mask() {
	vp.Volume.Mask()
	vp.Permission.Mask()
	if vp.OwnerUserID != vp.Permission.UserID {
		vp.Permissions = nil
	}
	for i := range vp.Permissions {
		vp.Permissions[i].Mask()
	}
}

// VolumeCreateRequest is a request object for creating volume
//
// swagger:model
type VolumeCreateRequest = model.CreateVolume

// VolumeRenameRequest is a request object for renaming volume
//
// swagger:model
type VolumeRenameRequest = model.ResourceUpdateName

// VolumeResizeRequest contains parameters for changing volume size
//
// swagger:model
type VolumeResizeRequest struct {
	// swagger:strfmt uuid
	TariffID string `json:"tariff_id" binding:"required,uuid"`
}
