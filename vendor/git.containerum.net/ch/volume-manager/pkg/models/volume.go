package model

import (
	"time"

	"git.containerum.net/ch/volume-manager/pkg/errors"
	"github.com/containerum/kube-client/pkg/model"
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

	StorageName string `sql:"storage_name,notnull" json:"storage_name,omitempty"`

	AccessMode model.PersistentVolumeAccessMode `sql:"access_mode,notnull" json:"access_mode,omitempty"`
}

func (v *Volume) BeforeInsert(db orm.DB) error {
	cnt, err := db.Model(v).
		Where("ns_id = ?ns_id").
		Where("label = ?label").
		Where("NOT deleted").
		Count()
	if err != nil {
		return err
	}

	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("volume %s already exists", v.Label)
	}

	_, err = db.Model(&Storage{Name: v.StorageName}).
		WherePK().
		Set("used = used + (?)", v.Capacity).
		Update()

	return err
}

func (v *Volume) AfterUpdate(db orm.DB) error {
	var err error
	if v.Deleted {
		_, err = db.Model(&Storage{Name: v.StorageName}).
			WherePK().
			Set("used = used - ?", v.Capacity).
			Update()
	} else {
		oldCapacityQuery := db.Model(v).Column("capacity").WherePK()
		_, err = db.Model(&Storage{Name: v.StorageName}).
			WherePK().
			Set("used = used - (?) + ?", oldCapacityQuery, v.Capacity).
			Update(v)
	}
	return err
}

func (v *Volume) ToKube() model.Volume {
	vol := model.Volume{
		Name: v.Label,
		CreatedAt: func() *string {
			t := v.CreateTime.Format(time.RFC3339)
			return &t
		}(),
		TariffID: func() string {
			if v.TariffID == nil {
				return ""
			}
			return *v.TariffID
		}(),
		Capacity:    uint(v.Capacity),
		StorageName: v.StorageName,
		AccessMode:  v.AccessMode,
	}

	return vol
}

func (v *Volume) Mask() {
	v.Resource.Mask()
	v.StorageName = ""
	v.AccessMode = ""
}

// VolumeCreateRequest is a request object for creating volume
//
// swagger:model
type VolumeCreateRequest = model.CreateVolume

// DirectVolumeCreateRequest is a request object for creating volume as admin (without billing)
//
// swagger:model
type DirectVolumeCreateRequest struct {
	Label    string `json:"label" binding:"required"`
	Capacity int    `json:"capacity" binding:"gt=0"`
}

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

// AdminVolumeResizeRequest contains parameters for changing volume size as admin
//
// swagger:model
type AdminVolumeResizeRequest struct {
	Capacity int `json:"capacity" binding:"gt=0"`
}
