package model

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/onecloud/pkg/cloudcommon/object"

	"yunion.io/x/yunion-kube/pkg/apis"
)

type SK8SObjectFactory struct {
	structType reflect.Type
}

func (f *SK8SObjectFactory) DataType() reflect.Type {
	return f.structType
}

func NewK8SObjectFactory(model interface{}) *SK8SObjectFactory {
	val := reflect.Indirect(reflect.ValueOf(model))
	st := val.Type()
	if st.Kind() != reflect.Struct {
		panic("expect struct kind")
	}
	factory := &SK8SObjectFactory{
		structType: st,
	}
	return factory
}

type SK8SModelBase struct {
	object.SObject

	K8SObject runtime.Object `json:"rawObject"`

	manager IK8SModelManager
	cluster ICluster
}

func (m SK8SModelBase) GetId() string {
	return ""
}

func (m SK8SModelBase) GetName() string {
	return ""
}

func (m SK8SModelBase) Keyword() string {
	return m.GetModelManager().Keyword()
}

func (m SK8SModelBase) KeywordPlural() string {
	return m.GetModelManager().KeywordPlural()
}

func (m *SK8SModelBase) SetModelManager(man IK8SModelManager, virtual IK8SModel) IK8SModel {
	m.manager = man
	m.SetVirtualObject(virtual)
	return m
}

func (m SK8SModelBase) GetModelManager() IK8SModelManager {
	return m.manager
}

func (m *SK8SModelBase) SetK8SObject(obj runtime.Object) IK8SModel {
	m.K8SObject = obj
	return m
}

func (m *SK8SModelBase) GetK8SObject() runtime.Object {
	return m.K8SObject
}

func (m *SK8SModelBase) GetMetaObject() metav1.Object {
	return m.GetK8SObject().(metav1.Object)
}

func (m *SK8SModelBase) SetCluster(cluster ICluster) IK8SModel {
	m.cluster = cluster
	return m
}

func (m *SK8SModelBase) GetCluster() ICluster {
	return m.cluster
}

func (m *SK8SModelBase) GetNamespace() string {
	return ""
}

func (m *SK8SModelBase) GetObjectMeta() apis.ObjectMeta {
	kObj := m.GetK8SObject()
	v := reflect.ValueOf(kObj)
	f := reflect.Indirect(v).FieldByName("ObjectMeta")
	if !f.IsValid() {
		panic(fmt.Sprintf("get invalid object meta %#v", kObj))
	}
	meta := f.Interface().(metav1.ObjectMeta)
	return apis.ObjectMeta{
		ObjectMeta:  meta,
		ClusterMeta: apis.NewClusterMeta(m.GetCluster()),
	}
}

func (m *SK8SModelBase) GetTypeMeta() apis.TypeMeta {
	kObj := m.GetK8SObject()
	v := reflect.ValueOf(kObj)
	f := reflect.Indirect(v).FieldByName("TypeMeta")
	if !f.IsValid() {
		panic(fmt.Sprintf("get invalid object meta %#v", kObj))
	}
	meta := f.Interface().(metav1.TypeMeta)
	return apis.TypeMeta{
		TypeMeta: meta,
	}
}

type SK8SModelBaseManager struct {
	object.SObject

	factory     *SK8SObjectFactory
	orderFields OrderFields

	keyword       string
	keywordPlural string
}

func NewK8SModelBaseManager(model interface{}, keyword, keywordPlural string) SK8SModelBaseManager {
	factory := NewK8SObjectFactory(model)
	modelMan := SK8SModelBaseManager{
		factory:       factory,
		orderFields:   make(map[string]IOrderField),
		keyword:       keyword,
		keywordPlural: keywordPlural,
	}
	return modelMan
}

func (m *SK8SModelBaseManager) GetIModelManager() IK8SModelManager {
	virt := m.GetVirtualObject()
	if virt == nil {
		panic(fmt.Sprintf("Forgot to call SetVirtualObject?"))
	}
	r, ok := virt.(IK8SModelManager)
	if !ok {
		panic(fmt.Sprintf("Cannot convert virtual object to IK8SModelManager"))
	}
	return r
}

func (m *SK8SModelBaseManager) Factory() *SK8SObjectFactory {
	return m.factory
}

func (m *SK8SModelBaseManager) Keyword() string {
	return m.keyword
}

func (m *SK8SModelBaseManager) KeywordPlural() string {
	return m.keywordPlural
}

func (m *SK8SModelBaseManager) GetContextManagers() [][]IK8SModelManager {
	return nil
}

func (m *SK8SModelBaseManager) ValidateName(name string) error {
	return nil
}

func (m *SK8SModelBaseManager) GetQuery(cluster ICluster) IQuery {
	return NewK8SResourceQuery(cluster, m.GetIModelManager())
}

func (m *SK8SModelBaseManager) GetOrderFields() OrderFields {
	return m.orderFields
}

func (m *SK8SModelBaseManager) RegisterOrderFields(fields ...IOrderField) {
	m.orderFields.Set(fields...)
}

func (m *SK8SModelBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query apis.ListInputK8SBase) (IQuery, error) {
	return q, nil
}
