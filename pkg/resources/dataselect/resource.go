package dataselect

import (
	"fmt"
	"reflect"

	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/proto/hapi/release"

	"yunion.io/x/yunion-kube/pkg/helm/data/cache"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type IList interface {
	Append(obj interface{})
	SetMeta(meta api.ListMeta)
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

type convertF func(item interface{}) (DataCell, error)

func ToCells(data interface{}, cf convertF) ([]DataCell, error) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Can't traverse non-slice value, kind: %v", v.Kind())
	}
	cells := make([]DataCell, 0)
	for i := 0; i < v.Len(); i++ {
		cell, err := cf(v.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		cells = append(cells, cell)
	}
	return cells, nil
}

func FromCells(cells []DataCell, list IList) {
	for _, cell := range cells {
		list.Append(cell.GetObject())
	}
}

func ToResourceList(list IList, data interface{}, cellConvertF convertF, dsQuery *DataSelectQuery) error {

	cells, err := ToCells(data, cellConvertF)
	if err != nil {
		return err
	}
	selector := GenericDataSelector(cells, dsQuery)
	FromCells(selector.Data(), list)
	list.SetMeta(selector.ListMeta())
	return nil
}

func getObjectMeta(obj interface{}) (metaV1.ObjectMeta, error) {
	v := reflect.ValueOf(obj)
	f := v.FieldByName("ObjectMeta")
	if !f.IsValid() {
		return metaV1.ObjectMeta{}, fmt.Errorf("Object %#v not have ObjectMeta field", obj)
	}
	meta := f.Interface().(metaV1.ObjectMeta)
	return meta, nil
}

func getObjectPodStatus(obj interface{}) (v1.PodStatus, error) {
	v := reflect.ValueOf(obj)
	f := v.FieldByName("Status")
	if !f.IsValid() {
		return v1.PodStatus{}, fmt.Errorf("Object %#v not have Status field", obj)
	}
	status := f.Interface().(v1.PodStatus)
	return status, nil
}

func NewNamespaceDataCell(obj interface{}) (DataCell, error) {
	meta, err := getObjectMeta(obj)
	if err != nil {
		return NamespaceDataCell{}, err
	}
	return NamespaceDataCell{ObjectMeta: meta, Object: obj}, nil
}

type NamespaceDataCell struct {
	ObjectMeta metaV1.ObjectMeta
	Object     interface{}
}

func (cell NamespaceDataCell) GetObject() interface{} {
	return cell.Object
}

func (cell NamespaceDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NameProperty:
		return StdComparableString(cell.ObjectMeta.Name)
	case CreationTimestampProperty:
		return StdComparableTime(cell.ObjectMeta.CreationTimestamp.Time)
	case NamespaceProperty:
		return StdComparableString(cell.ObjectMeta.Namespace)
	default:
		return nil
	}
}

func NewNamespacePodStatusDataCell(obj interface{}) (DataCell, error) {
	meta, err := getObjectMeta(obj)
	if err != nil {
		return NamespacePodStatusDataCell{}, err
	}
	status, err := getObjectPodStatus(obj)
	if err != nil {
		return NamespacePodStatusDataCell{}, err
	}
	return NamespacePodStatusDataCell{
		NamespaceDataCell: NamespaceDataCell{ObjectMeta: meta, Object: obj},
		Status:            status,
	}, nil
}

type NamespacePodStatusDataCell struct {
	NamespaceDataCell
	Status v1.PodStatus
}

func (cell NamespacePodStatusDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case StatusProperty:
		return StdComparableString(cell.Status.Phase)
	default:
		return cell.NamespaceDataCell.GetProperty(name)
	}
}

type HelmReleaseDataCell struct {
	Release *release.Release
}

func NewHelmReleaseDataCell(obj interface{}) (DataCell, error) {
	rls, ok := obj.(*release.Release)
	if !ok {
		return nil, fmt.Errorf("Object %#v not *release.Release", obj)
	}
	return HelmReleaseDataCell{Release: rls}, nil
}

func (cell HelmReleaseDataCell) GetObject() interface{} {
	return cell.Release
}

func (cell HelmReleaseDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NameProperty:
		return StdComparableString(cell.Release.Name)
	//case CreationTimestampProperty:
	//return StdComparableTime(cell.ObjectMeta.CreationTimestamp.Time)
	case NamespaceProperty:
		return StdComparableString(cell.Release.Namespace)
	default:
		return nil
	}
}

type ChartDataCell struct {
	Chart *cache.ChartResult
}

func NewChartDataCell(obj interface{}) (DataCell, error) {
	chart, ok := obj.(*cache.ChartResult)
	if !ok {
		return nil, fmt.Errorf("Object %#v not *cache.ChartResult", obj)
	}
	return ChartDataCell{Chart: chart}, nil
}

func (cell ChartDataCell) GetObject() interface{} {
	return cell.Chart
}

func (cell ChartDataCell) GetProperty(name PropertyName) ComparableValue {
	switch name {
	case NameProperty:
		return StdComparableString(cell.Chart.Chart.Name)
	default:
		return nil
	}
}
