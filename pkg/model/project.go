package model

type Project struct {
	tableName struct{} `sql:"projects"`

	Resource
}
