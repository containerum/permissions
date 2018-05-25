package dao

import (
	"context"

	"git.containerum.net/ch/permissions/pkg/errors"
	"git.containerum.net/ch/permissions/pkg/model"
	"github.com/go-pg/pg"
)

func (dao *DAO) CreateStorage(ctx context.Context, storage *model.Storage) error {
	dao.log.Debugf("create storage %+v", storage)

	storage.ID = ""
	storage.Used = 0
	_, err := dao.db.Model(storage).
		Returning("*").
		Insert()
	return dao.handleError(err)
}

func (dao *DAO) StorageByName(ctx context.Context, name string) (ret model.Storage, err error) {
	dao.log.WithField("name", name).Debugf("get storage by name")

	err = dao.db.Model(&ret).Where("name = ?", name).Select()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrResourceNotExists().AddDetailF("storage %s not exists", name)
	default:
		err = dao.handleError(err)
	}

	return
}

func (dao *DAO) AllStorages(ctx context.Context) (ret []model.Storage, err error) {
	dao.log.Debugf("get storage list")

	err = dao.db.Model(&ret).Select()
	err = dao.handleError(err)
	return
}

func (dao *DAO) UpdateStorage(ctx context.Context, name string, storage model.Storage) error {
	dao.log.WithField("name", name).Debugf("update storage to %+v", storage)

	if storage.Name != name {
		cnt, err := dao.db.Model(&storage).
			WherePK().
			WhereOr("name = ?", name).
			Count()
		if err != nil {
			return dao.handleError(err)
		}
		if cnt > 0 {
			return errors.ErrResourceAlreadyExists().AddDetailF("storage %s already exists", storage.Name)
		}
	}

	result, err := dao.db.Model(&storage).
		WherePK().
		WhereOr("name = ?", name).
		Set("name = ?name").
		Set("size = ?size").
		Set("replicas = ?replicas").
		Set("ips = ?ips").
		Update()
	if err != nil {
		return dao.handleError(err)
	}
	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("storage %s not exists", storage.Name)
	}
	return nil
}

func (dao *DAO) DeleteStorage(ctx context.Context, storage model.Storage) error {
	dao.log.WithField("name", storage.Name).Debugf("delete storage")

	result, err := dao.db.Model(&storage).WherePK().Delete()
	if err != nil {
		return dao.handleError(err)
	}
	if result.RowsAffected() <= 0 {
		return errors.ErrResourceNotExists().AddDetailF("storage %s not exists", storage.Name)
	}
	return nil
}

func (dao *DAO) LeastUsedStorage(ctx context.Context, minFree int) (ret model.Storage, err error) {
	dao.log.WithField("min_free", minFree).Debugf("get least used storage with constraint")

	err = dao.db.Model(&ret).
		Where("size - used >= ?", minFree).
		OrderExpr("used ASC").
		First()
	switch err {
	case pg.ErrNoRows:
		err = errors.ErrNoFreeStorages()
	default:
		err = dao.handleError(err)
	}

	return
}
