package model

import (
	"errors"
	"time"

	"github.com/go-pg/pg/orm"
)

// Resource represents common resource information.
//
// swagger:ignore
type Resource struct {
	// swagger:strfmt uuid
	ID string `sql:"id,pk,type:uuid,default:uuid_generate_v4()" json:"id,omitempty"`

	CreateTime *time.Time `sql:"create_time,default:now(),notnull" json:"create_time,omitempty"`

	Deleted bool `sql:"deleted,notnull" json:"deleted,omitempty"`

	DeleteTime *time.Time `sql:"delete_time" json:"delete_time,omitempty"`

	// swagger:strfmt uuid
	TariffID *string `sql:"tariff_id,type:uuid" json:"tariff_id,omitempty"`

	// swagger:strfmt uuid
	OwnerUserID string `sql:"owner_user_id,type:uuid,notnull,unique:unique_owner_label" json:"owner_user_id,omitempty"`

	Label string `sql:"label,notnull,unique:unique_owner_label" json:"label"`

	Permissions []Permission `pg:"polymorphic:resource_" sql:"-" json:"users,omitempty"`
}

func (r *Resource) BeforeDelete(db orm.DB) error {
	// do not allow delete from app
	return errors.New("record delete not allowed, use update set deleted = true")
}

func (r *Resource) BeforeUpdate(db orm.DB) error {
	if r.Deleted {
		now := time.Now()
		r.DeleteTime = &now
	}
	return nil
}

func (r *Resource) Mask() {
	r.ID = ""
	r.CreateTime = nil
	r.DeleteTime = nil
	r.TariffID = nil
	r.OwnerUserID = ""
	r.Label = ""
}
