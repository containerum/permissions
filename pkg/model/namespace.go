package model

type Namespace struct {
	tableName struct{} `sql:"namespaces"`

	Resource

	RAM            int `sql:"ram,notnull"`
	CPU            int `sql:"cpu,notnull"`
	MaxExtServices int `sql:"max_ext_services,notnull"`
	MaxIntServices int `sql:"max_int_services,notnull"`
	MaxTraffic     int `sql:"max_traffic,notnull"`

	Volumes []*Volume `sql:"-"`
}
