package postgres

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

func (pgdb *PgDB) CreateProject(ctx context.Context, project *model.Project) error {
	pgdb.log.Debugf("create project %+v", project)

	_, err := pgdb.db.Model(project).
		Returning("*").
		Insert()
	if err != nil {
		err = pgdb.handleError(err)
	}

	return err
}

func (pgdb *PgDB) ProjectByLabel(ctx context.Context, project string) (p model.Project, err error) {
	pgdb.log.WithField("project", project).Debugf("get project")

	p.Label = project
	err = pgdb.db.Model(&p).
		Column("projects.*", "Namespaces").
		Relation("Namespaces", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("NOT namespaces.deleted"), nil
		}).
		Where("label = ?label").
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("project %s not exists", project)
	default:
		err = pgdb.handleError(err)
	}

	return
}
