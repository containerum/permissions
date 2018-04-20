package model

import (
	"time"

	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/go-pg/pg/orm"
)

// swagger:ignore
type ResourceKind string // enum

const (
	ResourceNamespace ResourceKind = "namespace"
	ResourceVolume    ResourceKind = "volume"
)

// swagger:ignore
type AccessLevel string // enum

const (
	AccessNone       AccessLevel = "none"
	AccessRead       AccessLevel = "read"
	AccessReadDelete AccessLevel = "readdelete"
	AccessWrite      AccessLevel = "write"
	AccessOwner      AccessLevel = "owner"
)

// swagger:ignore
type Permission struct {
	tableName struct{} `sql:"permissions"`

	ID                    string       `sql:"id,pk,type:uuid,default:uuid_generate_v4()"`
	ResourceKind          ResourceKind `sql:"resource_kind,type:RESOURCE_KIND,notnull"` // WARN: custom type here, do not forget create it
	ResourceID            string       `sql:"resource_id,type:UUID,notnull"`
	CreateTime            time.Time    `sql:"create_time,default:now(),notnull"`
	UserID                string       `sql:"user_id,type:uuid,notnull"`
	InitialAccessLevel    AccessLevel  `sql:"initial_access_level,type:ACCESS_LEVEL,notnull"` // WARN: custom type here, do not forget create it
	CurrentAccessLevel    AccessLevel  `sql:"current_access_level,type:ACCESS_LEVEL,notnull"` // WARN: custom type here, do not forget create it
	AccessLevelChangeTime time.Time    `sql:"access_level_change_time,default:now(),notnull"`
}

func (p *Permission) BeforeInsert(db orm.DB) error {
	if p.InitialAccessLevel == AccessOwner {
		cnt, err := db.Model((*Permission)(nil)).
			Column("id").
			Where("resource_id = ?", p.ResourceID).
			Where("resource_kind = ?", p.ResourceKind).
			Where("initial_access_level = ?", AccessOwner).
			SelectAndCount()
		if err != nil {
			return err
		}
		if cnt > 0 {
			return errors.ErrOwnerAlreadyExists()
		}
	}
	if p.CurrentAccessLevel > p.InitialAccessLevel {
		// that`s our error if we will here
		return errors.ErrInternal().AddDetails("initial access level must be greater than current access level")
	}
	return nil
}

func (p *Permission) BeforeUpdate(db orm.DB) error {
	if p.CurrentAccessLevel > p.InitialAccessLevel {
		// that`s our error if we will here
		return errors.ErrInternal().AddDetails("initial access level must be greater than current access level")
	}
	return nil
}

// SetUserAccessRequest is a request object for setting user accesses
//
// swagger:model SetResourcesAccessesRequest
type SetUserAccessesRequest struct {
	Access AccessLevel `json:"access"`
}
