package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/go-pg/pg/orm"
)

type ResourceKind string // enum

const (
	ResourceNamespace ResourceKind = "namespace"
	ResourceVolume    ResourceKind = "volume"
)

type AccessLevel int // enum
//go:generate stringer -type=AccessLevel -trimprefix=Access
const (
	AccessNone AccessLevel = iota
	AccessRead
	AccessReadDelete
	AccessWrite
	AccessOwner
)

func (i AccessLevel) Value() (driver.Value, error) {
	return strings.ToLower(i.String()), nil
}

func (i *AccessLevel) Scan(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid enum value %[1]v (%[1]T)", v)
	}
	str = strings.ToLower(str)
	for idx := 0; idx < len(_AccessLevel_index)-1; idx++ {
		if strings.ToLower(_AccessLevel_name[_AccessLevel_index[idx]:_AccessLevel_index[idx+1]]) == str {
			*i = AccessLevel(idx)
			return nil
		}
	}
	return fmt.Errorf("invalid enum value %v", v)
}

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
