package model

import (
	"time"

	"git.containerum.net/ch/volume-manager/pkg/errors"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
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

	Label string `sql:"label,notnull" json:"label"`

	OwnerUserID string `sql:"owner_user_id,notnull,type:uuid" json:"owner_user_id,omitempty"`
}

func (r *Resource) BeforeDelete(db orm.DB) error {
	// do not allow delete from app
	logrus.Error("record delete not allowed, use update set deleted = true")
	return errors.ErrInternal()
}

func (r *Resource) Mask() {
	r.CreateTime = nil
	r.DeleteTime = nil
	r.OwnerUserID = ""
}
