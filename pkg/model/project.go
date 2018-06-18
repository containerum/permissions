package model

import (
	"git.containerum.net/ch/permissions/pkg/errors"
	"github.com/go-pg/pg/orm"
)

type Project struct {
	tableName struct{} `sql:"projects"`

	Resource

	Namespaces []Namespace `sql:"-" pg:"fk:project_id" json:"namespaces,omitempty"`
}

func (p *Project) BeforeInsert(db orm.DB) error {
	cnt, err := db.Model(p).
		Where("owner_user_id = ?owner_user_id").
		Where("label = ?label").
		Where("NOT deleted").
		Count()
	if err != nil {
		return err
	}

	if cnt > 0 {
		return errors.ErrResourceAlreadyExists().AddDetailF("project %s already exists", p.Label)
	}

	return nil
}

// ProjectCreateRequest contains parameters for creating project
//
// swagger:model
type ProjectCreateRequest struct {
	Label string `json:"label" binding:"required"`
}

// ProjectAddGroupRequest contains parameters for adding permissions for group
//
// swagger:model
type ProjectAddGroupRequest struct {
	GroupID string `json:"group" binding:"required"`
}
