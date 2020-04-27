package client

import (
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client/api"
)

type ResourceHandler interface {
	Create(kind string, namespace string, object *runtime.Unknown) (*runtime.Unknown, error)
	CreateV2(kind string, namespace string, object runtime.Object) (runtime.Object, error)
	Update(kind string, namespace string, name string, object *runtime.Unknown) (*runtime.Unknown, error)
	UpdateV2(kind string, object runtime.Object) (runtime.Object, error)
	Get(kind string, namespace string, name string) (runtime.Object, error)
	List(kind string, namespace string, labelSelector string) ([]runtime.Object, error)
	Delete(kind string, namespace string, name string, options *metav1.DeleteOptions) error
	GetIndexer() *CacheFactory
	GetClientset() *kubernetes.Clientset
	Close()
}

type resourceHandler struct {
	client       *kubernetes.Clientset
	cacheFactory *CacheFactory
}

func NewResourceHandler(kubeClient *kubernetes.Clientset, cacheFactory *CacheFactory) ResourceHandler {
	return &resourceHandler{
		client:       kubeClient,
		cacheFactory: cacheFactory,
	}
}

func (h *resourceHandler) GetClientset() *kubernetes.Clientset {
	return h.client
}

func (h *resourceHandler) GetIndexer() *CacheFactory {
	return h.cacheFactory
}

func (h *resourceHandler) Close() {
	close(h.cacheFactory.stopChan)
}

func (h *resourceHandler) Create(kind string, namespace string, object *runtime.Unknown) (*runtime.Unknown, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return nil, fmt.Errorf("Resource kind (%s) not support yet . ", kind)
	}
	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Post().
		Resource(kind).
		SetHeader("Content-Type", "application/json").
		Body([]byte(object.Raw))
	if resource.Namespaced {
		req.Namespace(namespace)
	}
	var result runtime.Unknown
	err := req.Do().Into(&result)

	return &result, err
}

func (h *resourceHandler) CreateV2(kind string, namespace string, object runtime.Object) (runtime.Object, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return nil, fmt.Errorf("Resource kind (%s) not support yet . ", kind)
	}
	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Post().Resource(kind)
	if resource.Namespaced {
		req.Namespace(namespace)
	}
	return req.VersionedParams(&metav1.CreateOptions{}, metav1.ParameterCodec).
		Body(object).
		Do().
		Get()
}

func (h *resourceHandler) Update(kind string, namespace string, name string, object *runtime.Unknown) (*runtime.Unknown, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return nil, fmt.Errorf("Resource kind (%s) not support yet.", kind)
	}

	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Put().
		Resource(kind).
		Name(name).
		SetHeader("Content-Type", "application/json").
		Body([]byte(object.Raw))
	if resource.Namespaced {
		req.Namespace(namespace)
	}

	var result runtime.Unknown
	err := req.Do().Into(&result)

	return &result, err
}

func (h *resourceHandler) UpdateV2(kind string, object runtime.Object) (runtime.Object, error) {
	metaObj, ok := object.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("object %#v not metav1.Object", object)
	}
	putSpec := runtime.Unknown{}
	objStr, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(strings.NewReader(string(objStr))).Decode(&putSpec); err != nil {
		return nil, err
	}

	// todo: fix convert unknown to runtime.object
	updateObj, err := h.Update(kind, metaObj.GetNamespace(), metaObj.GetName(), &putSpec)
	if err != nil {
		return nil, err
	}
	jBytes, err := updateObj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(strings.NewReader(string(jBytes))).Decode(object); err != nil {
		return nil, err
	}
	return object, err
}

func (h *resourceHandler) Delete(kind string, namespace string, name string, options *metav1.DeleteOptions) error {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return fmt.Errorf("Resource kind (%s) not support yet.", kind)
	}
	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Delete().
		Resource(kind).
		Name(name).
		Body(options)
	if resource.Namespaced {
		req.Namespace(namespace)
	}

	return req.Do().Error()
}

// Get object from cache
func (h *resourceHandler) Get(kind string, namespace string, name string) (runtime.Object, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return nil, fmt.Errorf("Resource kind (%s) not support yet.", kind)
	}
	genericInformer, err := h.cacheFactory.sharedInformerFactory.ForResource(resource.GroupVersionResourceKind.GroupVersionResource)
	if err != nil {
		return nil, err
	}
	lister := genericInformer.Lister()
	var result runtime.Object
	if resource.Namespaced {
		result, err = lister.ByNamespace(namespace).Get(name)
		if err != nil {
			return nil, err
		}
	} else {
		result, err = lister.Get(name)
		if err != nil {
			return nil, err
		}
	}
	result.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   resource.GroupVersionResourceKind.Group,
		Version: resource.GroupVersionResourceKind.Version,
		Kind:    resource.GroupVersionResourceKind.Kind,
	})

	return result, nil
}

// Get object from cache
func (h *resourceHandler) List(kind string, namespace string, labelSelector string) ([]runtime.Object, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return nil, fmt.Errorf("Resource kind (%s) not support yet.", kind)
	}
	genericInformer, err := h.cacheFactory.sharedInformerFactory.ForResource(resource.GroupVersionResourceKind.GroupVersionResource)
	if err != nil {
		return nil, err
	}
	selectors, err := labels.Parse(labelSelector)
	if err != nil {
		log.Errorf("Build label selector error: %v.", err)
		return nil, err
	}

	lister := genericInformer.Lister()
	var objs []runtime.Object
	if resource.Namespaced {
		objs, err = lister.ByNamespace(namespace).List(selectors)
		if err != nil {
			return nil, err
		}
	} else {
		objs, err = lister.List(selectors)
		if err != nil {
			return nil, err
		}
	}

	for i := range objs {
		objs[i].GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   resource.GroupVersionResourceKind.Group,
			Version: resource.GroupVersionResourceKind.Version,
			Kind:    resource.GroupVersionResourceKind.Kind,
		})
	}

	return objs, nil
}
