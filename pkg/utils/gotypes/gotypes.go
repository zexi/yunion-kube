package gotypes

import (
	"reflect"

	"yunion.io/x/pkg/gotypes"
)

func RegisterSerializable(objs ...gotypes.ISerializable) {
	for _, obj := range objs {
		gotypes.RegisterSerializable(reflect.TypeOf(obj), func() gotypes.ISerializable {
			return obj
		})
	}
}
