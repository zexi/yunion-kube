package client

import (
	"fmt"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/cmd/helm/installer"

	"yunion.io/x/log"
)

type InstallOption struct {
	// Name of the kubeconfig context to use
	KubeContext string `json:"kube_context"`

	// Namespace of Tiller
	Namespace string `json:"namespace"` // "kube-system"

	// Upgrade if Tiller is already installed
	Upgrade bool `json:"upgrade"`

	// Name of service account
	ServiceAccount string `json:"service_account"`

	// Use the canary Tiller image
	Canary bool `json:"canary_image"`

	// Override Tiller image
	ImageSpec string `json:"tiller_image"`

	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory int `json:"history_max"`
}

func PreInstall(cli kubernetes.Interface, opt *InstallOption) error {
	v1MetaData := metav1.ObjectMeta{
		Name:      opt.ServiceAccount, // "tiller"
		Namespace: opt.Namespace,
	}
	serviceAccount := &apiv1.ServiceAccount{
		ObjectMeta: v1MetaData,
	}
	var err error
	log.Infof("Create service account: %q, namespace: %q", v1MetaData.Name, v1MetaData.Namespace)
	for i := 0; i <= 5; i++ {
		_, err = cli.CoreV1().ServiceAccounts(opt.Namespace).Create(serviceAccount)
		if err != nil {
			log.Warningf("create service account failed: %v", err)
			if strings.Contains(err.Error(), "etcdserver: request timed out") {
				time.Sleep(time.Duration(40) * time.Second)
				continue
			}
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("create service account failed: %v", err)
			}
		}
		break
	}
	clusterRole := &v1.ClusterRole{
		ObjectMeta: v1MetaData,
		Rules: []v1.PolicyRule{{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"*",
			},
			Verbs: []string{
				"*",
			},
		},
			{
				NonResourceURLs: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			}},
	}

	log.Infof("Create cluster roles: %q, namespace: %q", v1MetaData.Name, v1MetaData.Namespace)
	clusterRoleName := opt.ServiceAccount
	for i := 0; i <= 5; i++ {
		_, err = cli.RbacV1().ClusterRoles().Create(clusterRole)
		if err != nil {
			if strings.Contains(err.Error(), "etcdserver: request timed out") {
				time.Sleep(time.Duration(10) * time.Second)
				continue
			} else if strings.Contains(err.Error(), "is forbidden") {
				_, errGet := cli.RbacV1().ClusterRoles().Get("cluster-admin", metav1.GetOptions{})
				if errGet != nil {
					return fmt.Errorf("clusterrole create error: %v cluster-admin not found: %v", err, errGet)
				}
				clusterRoleName = "cluster-admin"
				break
			}
			log.Warningf("create roles failed: %v", err)
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("crate roles failed: %s", err)
			}
		}
		break
	}

	log.Debugf("ClusterRole name: %s", clusterRoleName)
	log.Debugf("ServiceAccount name: %s", opt.ServiceAccount)
	clusterRoleBinding := &v1.ClusterRoleBinding{
		ObjectMeta: v1MetaData,
		RoleRef: v1.RoleRef{
			APIGroup: v1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      opt.ServiceAccount,
				Namespace: opt.Namespace,
			},
		},
	}
	log.Infof("Crate cluster role bindings: %q, namespace: %q, roleRef: %q", v1MetaData.Name, v1MetaData.Namespace, clusterRoleName)
	for i := 0; i <= 5; i++ {
		_, err = cli.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
		if err != nil {
			log.Warningf("create role bindings failed: %v", err)
			if strings.Contains(err.Error(), "etcdserver: request timed out") {
				time.Sleep(time.Duration(10) * time.Second)
				continue
			}
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("Create role bindings failed: %v", err)
			}
		}
		break
	}
	return nil
}

func Install(cli kubernetes.Interface, opt *InstallOption) error {
	log.Infof("Install helm tiller server")
	if err := PreInstall(cli, opt); err != nil {
		return err
	}

	opts := installer.Options{
		Namespace:      opt.Namespace,
		ServiceAccount: opt.ServiceAccount,
		UseCanary:      opt.Canary,
		ImageSpec:      opt.ImageSpec,
		MaxHistory:     opt.MaxHistory,
	}
	if err := installer.Install(cli, &opts); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
		if opt.Upgrade {
			if err := installer.Upgrade(cli, &opts); err != nil {
				return fmt.Errorf("error when upgrading: %v", err)
			}
			log.Infof("Tiller (the Helm server-side component) has been upgraded to the current version.")
		} else {
			msg := "Tiller is already installed in the cluster."
			return fmt.Errorf(msg)
		}
	} else {
		log.Infof("Tiller (the Helm server-side component) has been installed into your Kuberntes Cluster.")
	}
	log.Infof("Helm install finished")
	return nil
}
