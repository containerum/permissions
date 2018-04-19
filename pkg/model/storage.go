package model

type Storage struct {
	tableName struct{} `sql:"storages"`

	ID       string   `sql:"id,pk,type:uuid,default:uuid_generate_v4()"`
	Name     string   `sql:"name,notnull,unique"`
	Size     int      `sql:"size,notnull"`
	Used     int      `sql:"used,notnull"`
	Replicas int      `sql:"replicas,notnull"`
	IPs      []string `sql:"ips,notnull,type:inet[],array"`

	Volumes []*Volume `pg:"fk:storage_id" sql:"-"`
}
