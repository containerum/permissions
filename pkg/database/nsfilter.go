package database

import (
	"reflect"

	"github.com/go-pg/pg/orm"
)

type NamespaceFilter struct {
	orm.Pager
	NotDeleted bool `filter:"not_deleted"`
	Deleted    bool `filter:"deleted"`
	NotLimited bool `filter:"not_limited"`
	Limited    bool `filter:"limited"`
	Owned      bool `filter:"owner"`
	NotOwned   bool `filter:"not_owner"`
}

var nsFilterCache = make(map[string]int)

func init() {
	t := reflect.TypeOf(NamespaceFilter{})
	for i := 0; i < t.NumField(); i++ {
		tag, ok := t.Field(i).Tag.Lookup("filter")
		if !ok {
			continue
		}
		nsFilterCache[tag] = i
	}
}

func ParseNamespaceFilter(filters ...string) NamespaceFilter {
	var ret NamespaceFilter
	v := reflect.ValueOf(&ret).Elem()
	for _, filter := range filters {
		if field, ok := nsFilterCache[filter]; ok {
			v.Field(field).SetBool(true)
		}
	}
	return ret
}
