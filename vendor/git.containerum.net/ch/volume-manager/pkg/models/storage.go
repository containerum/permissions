package model

import (
	"git.containerum.net/ch/volume-manager/pkg/errors"
	"github.com/go-pg/pg/orm"
)

// Storage describes volumes storage
//
// swagger:model
type Storage struct {
	tableName struct{} `sql:"storages"`

	Name string `sql:"name,pk,notnull" json:"name" binding:"required"`

	Size int `sql:"size,notnull" json:"size" binding:"gt=0"`

	Used int `sql:"used,notnull" json:"used" binding:"gt=0,ltecsfield=Size"`

	Volumes []*Volume `pg:"fk:storage_id" sql:"-" json:"volumes"`
}

func (s *Storage) BeforeInsert(db orm.DB) error {
	cnt, err := db.Model(s).Where("name = ?name").Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("storage %s already exists", s.Name)
	}
	return nil
}

func (s *Storage) BeforeUpdate(db orm.DB) error {
	if s.Size < s.Used {
		return errors.ErrQuotaExceeded().AddDetailF("storage quota exceeded (%d GiB)", s.Used-s.Size)
	}
	return nil
}

func (s *Storage) BeforeDelete(db orm.DB) error {
	cnt, err := db.Model(&Volume{StorageName: s.Name}).
		WherePK().
		Where("NOT deleted").
		Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return errors.ErrStorageDelete()
	}
	return nil
}

// UpdateStorageRequest represents request object for updating storage
//
// swagger:model
type UpdateStorageRequest struct {
	Name *string `json:"name,omitempty"`
	Size *int    `json:"size,omitempty" binding:"omitempty,gt=0,gtecsfield=Used"`
	Used *int    `json:"size,omitempty"`
}
