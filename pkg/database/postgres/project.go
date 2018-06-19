package postgres

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/sirupsen/logrus"
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

func (pgdb *PgDB) ProjectByID(ctx context.Context, project string) (p model.Project, err error) {
	pgdb.log.WithField("project", project).Debugf("get project")

	p.ID = project
	err = pgdb.db.Model(&p).
		ColumnExpr("?TableAlias.*").
		Column("Namespaces").
		Relation("Namespaces", func(q *orm.Query) (*orm.Query, error) {
			return q.Where("NOT namespaces.deleted"), nil
		}).
		WherePK().
		Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("project %s not exists", project)
	default:
		err = pgdb.handleError(err)
	}

	return
}

func (pgdb *PgDB) DeleteGroupFromProject(ctx context.Context, projectID, groupID string) (deletedPerms []model.Permission, err error) {
	pgdb.log.WithFields(logrus.Fields{
		"project_id": projectID,
		"group_id":   groupID,
	}).Debugf("delete group from project")

	_, err = pgdb.db.Model(&deletedPerms).
		Where("group_id = ?", groupID).
		Where("resource_type = ?", model.ResourceNamespace).
		Where("resource_id IN (?)", pgdb.db.Model(&model.Namespace{ProjectID: &projectID}).
			Where("project_id = ?project_id")).
		Returning("*").
		Delete()

	switch err {
	case pg.ErrNoRows:
		err = nil
	default:
		err = pgdb.handleError(err)
	}

	return
}
