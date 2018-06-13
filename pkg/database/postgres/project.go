package postgres

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/model"
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
