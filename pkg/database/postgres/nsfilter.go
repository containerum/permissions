package postgres

import (
	"git.containerum.net/ch/permissions/pkg/database"
	kubeClientModel "github.com/containerum/kube-client/pkg/model"
	"github.com/go-pg/pg/orm"
)

type NamespaceFilter database.NamespaceFilter

func (f *NamespaceFilter) Filter(q *orm.Query) (*orm.Query, error) {
	if f.NotDeleted {
		q = q.Where("NOT ?TableAlias.deleted")
	}
	if f.Deleted {
		q = q.Where("?TableAlias.deleted")
	}
	if f.NotLimited {
		q = q.Where("permission.initial_access_level = permissions.current_access_level")
	}
	if f.Limited {
		q = q.Where("permission.initial_access_level != permissions.initial_access_level")
	}
	if f.Owned {
		q = q.Where("permission.initial_access_level = ?", kubeClientModel.Owner)
	}
	if f.NotOwned {
		q = q.Where("permission.initial_access_level != ?", kubeClientModel.Owner)
	}
	if f.Limit > 0 {
		q = q.Apply(f.Paginate)
	}

	return q, nil
}
