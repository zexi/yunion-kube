package dataselect

import (
	"fmt"
	"reflect"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type IList interface {
	Append(obj interface{})
	SetMeta(meta api.ListMeta)
	ToCell(obj interface{}) DataCell
}

type ListMeta struct {
	api.ListMeta
}

func NewListMeta() *ListMeta {
	return &ListMeta{}
}

func (l *ListMeta) SetMeta(meta api.ListMeta) {
	l.ListMeta = meta
}

type convertF func(item interface{}) DataCell

func ToCells(data interface{}, cf convertF) ([]DataCell, error) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Can't traverse non-slice value, kind: %v", v.Kind())
	}
	cells := make([]DataCell, 0)
	for i := 0; i < v.Len(); i++ {
		cells = append(cells, cf(v.Index(i).Interface()))
	}
	return cells, nil
}

func FromCells(cells []DataCell, list IList) {
	for _, cell := range cells {
		list.Append(cell)
	}
}

func ToResourceList(list IList, data interface{}, dsQuery *DataSelectQuery) error {

	cells, err := ToCells(data, list.ToCell)
	if err != nil {
		return err
	}
	selector := GenericDataSelector(cells, dsQuery)
	FromCells(selector.Data(), list)
	list.SetMeta(selector.ListMeta())
	return nil
}
