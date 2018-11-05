package lxcfs

import (
	"encoding/json"
	"time"

	admissionregistration "k8s.io/api/admissionregistration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/options"
)

const (
	defaultAnnotation      = "initializer.kubernetes.io/lxcfs"
	defaultConfigName      = "lxcfs.initializer"
	defaultInitializerName = "lxcfs.initializer.kubernetes.io"
	lxcfsLabel             = "lxcfs"
	notUseLxcfs            = "false"
)

type LxcfsInitializeController struct {
	client     *kubernetes.Clientset
	stopCh     chan struct{}
	controller cache.Controller
}

func NewLxcfsInitializeController(clientset *kubernetes.Clientset, stopCh chan struct{}) *LxcfsInitializeController {
	c := &LxcfsInitializeController{
		client: clientset,
		stopCh: stopCh,
	}

	// Watch uninitialized Pods in all namespaces.
	restClient := c.client.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", corev1.NamespaceAll, fields.Everything())

	// Wrap the returned watchlist to workaround the inability to include
	// the `IncludeUninitialized` list option when setting up watch clients.
	includeUninitializedWatchlist := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.IncludeUninitialized = true
			return watchlist.List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.IncludeUninitialized = true
			return watchlist.Watch(options)
		},
	}

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(includeUninitializedWatchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				err := initializePod(obj.(*corev1.Pod), getLxcfsConfig(), c.client)
				if err != nil {
					log.Errorf("Initialize pod error: %v", err)
				}
			},
		},
	)

	c.controller = controller
	return c
}

func (c *LxcfsInitializeController) Run() {
	err := ensureCreateInitializerConfig(c.client)
	if err != nil {
		log.Errorf("Create %s InitializerConfiguration: %v", defaultConfigName, err)
		return
	}
	if c.controller != nil {
		go c.controller.Run(c.stopCh)
	}
	<-c.stopCh
}

type config struct {
	volumes      []corev1.Volume
	volumeMounts []corev1.VolumeMount
}

func getLxcfsConfig() *config {
	// -v /var/lib/lxcfs/proc/cpuinfo:/proc/cpuinfo:rw
	// -v /var/lib/lxcfs/proc/diskstats:/proc/diskstats:rw
	// -v /var/lib/lxcfs/proc/meminfo:/proc/meminfo:rw
	// -v /var/lib/lxcfs/proc/stat:/proc/stat:rw
	// -v /var/lib/lxcfs/proc/swaps:/proc/swaps:rw
	// -v /var/lib/lxcfs/proc/uptime:/proc/uptime:rw
	c := &config{
		volumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      "lxcfs-proc-cpuinfo",
				MountPath: "/proc/cpuinfo",
			},
			corev1.VolumeMount{
				Name:      "lxcfs-proc-meminfo",
				MountPath: "/proc/meminfo",
			},
			corev1.VolumeMount{
				Name:      "lxcfs-proc-diskstats",
				MountPath: "/proc/diskstats",
			},
			corev1.VolumeMount{
				Name:      "lxcfs-proc-stat",
				MountPath: "/proc/stat",
			},
			corev1.VolumeMount{
				Name:      "lxcfs-proc-swaps",
				MountPath: "/proc/swaps",
			},
			corev1.VolumeMount{
				Name:      "lxcfs-proc-uptime",
				MountPath: "/proc/uptime",
			},
		},
		volumes: []corev1.Volume{
			corev1.Volume{
				Name: "lxcfs-proc-cpuinfo",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/cpuinfo",
					},
				},
			},
			corev1.Volume{
				Name: "lxcfs-proc-diskstats",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/diskstats",
					},
				},
			},
			corev1.Volume{
				Name: "lxcfs-proc-meminfo",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/meminfo",
					},
				},
			},
			corev1.Volume{
				Name: "lxcfs-proc-stat",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/stat",
					},
				},
			},
			corev1.Volume{
				Name: "lxcfs-proc-swaps",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/swaps",
					},
				},
			},
			corev1.Volume{
				Name: "lxcfs-proc-uptime",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/lib/lxcfs/proc/uptime",
					},
				},
			},
		},
	}
	return c
}

func initializePod(pod *corev1.Pod, c *config, clientset *kubernetes.Clientset) error {
	if pod.ObjectMeta.GetInitializers() != nil {
		pendingInitializers := pod.ObjectMeta.GetInitializers().Pending

		if defaultInitializerName == pendingInitializers[0].Name {
			log.Debugf("Initializing pod: %s", pod.Name)

			initializedPod := pod.DeepCopy()

			// Remove self from the list of pending Initializers while preserving ordering.
			if len(pendingInitializers) == 1 {
				initializedPod.ObjectMeta.Initializers = nil
			} else {
				initializedPod.ObjectMeta.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
			}

			if options.Options.LxcfsRequireAnnotation {
				a := pod.ObjectMeta.GetAnnotations()
				_, ok := a[defaultAnnotation]
				if !ok {
					log.Infof("Required %q annotation missing; pod %q skipping lxcfs injection", defaultConfigName, pod.Name)
					_, err := clientset.CoreV1().Pods(pod.Namespace).Update(initializedPod)
					if err != nil {
						return err
					}
					return nil
				}
			}

			labels := pod.ObjectMeta.GetLabels()
			if val, ok := labels[lxcfsLabel]; ok && val == notUseLxcfs {
				log.Printf("Pod %q set lxcfs=false label; skipping lxcfs injection", pod.Name)
				_, err := clientset.CoreV1().Pods(pod.Namespace).Update(initializedPod)
				if err != nil {
					return err
				}
				return nil
			}

			containers := initializedPod.Spec.Containers

			// Modify the Pod to include the Envoy container
			// and configuration volume. Then patch the original pod.
			for i := range containers {
				containers[i].VolumeMounts = append(containers[i].VolumeMounts, c.volumeMounts...)
			}

			initializedPod.Spec.Volumes = append(pod.Spec.Volumes, c.volumes...)

			oldData, err := json.Marshal(pod)
			if err != nil {
				return err
			}

			newData, err := json.Marshal(initializedPod)
			if err != nil {
				return err
			}

			patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, corev1.Pod{})
			if err != nil {
				return err
			}

			_, err = clientset.CoreV1().Pods(pod.Namespace).Patch(pod.Name, types.StrategicMergePatchType, patchBytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetInitializerConfig() *admissionregistration.InitializerConfiguration {
	conf := admissionregistration.InitializerConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultConfigName,
		},
		Initializers: []admissionregistration.Initializer{
			admissionregistration.Initializer{
				Name: defaultInitializerName,
				Rules: []admissionregistration.Rule{
					admissionregistration.Rule{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						Resources:   []string{"pods"},
					},
				},
			},
		},
	}
	return &conf
}

func ensureCreateInitializerConfig(client *kubernetes.Clientset) error {
	_, err := client.AdmissionregistrationV1alpha1().InitializerConfigurations().Get(defaultConfigName, metav1.GetOptions{})
	if err == nil {
		// already exists
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}
	_, err = client.AdmissionregistrationV1alpha1().InitializerConfigurations().Create(GetInitializerConfig())
	return err
}
