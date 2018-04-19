package model

type Volume struct {
	tableName struct{} `sql:"volumes"`

	Resource

	Active      bool   `sql:"active,notnull"`
	Capacity    int    `sql:"capacity,notnull"`
	Replicas    int    `sql:"replicas,notnull"`
	NamespaceID int    `sql:"ns_id,type:UUID,notnull"`
	GlusterName string `sql:"gluster_name,notnull"`
	StorageID   string `sql:"storage_id,type:UUID,notnull"`
}
