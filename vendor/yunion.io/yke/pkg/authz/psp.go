package authz

import (
	"context"

	"yunion.io/yke/pkg/k8s"
	"yunion.io/yke/pkg/templates"
	"yunion.io/yunioncloud/pkg/log"
)

func ApplyDefaultPodSecurityPolicy(ctx context.Context, kubeConfigPath string, k8sWrapTransport k8s.WrapTransport) error {
	log.Infof("[authz] Applying default PodSecurityPolicy")
	k8sClient, err := k8s.NewClient(kubeConfigPath, k8sWrapTransport)
	if err != nil {
		return err
	}
	if err := k8s.UpdatePodSecurityPolicyFromYaml(k8sClient, templates.DefaultPodSecurityPolicy); err != nil {
		return err
	}
	log.Infof("[authz] Default PodSecurityPolicy applied successfully")
	return nil
}

func ApplyDefaultPodSecurityPolicyRole(ctx context.Context, kubeConfigPath string, k8sWrapTransport k8s.WrapTransport) error {
	log.Infof("[authz] Applying default PodSecurityPolicy Role and RoleBinding")
	k8sClient, err := k8s.NewClient(kubeConfigPath, k8sWrapTransport)
	if err != nil {
		return err
	}
	if err := k8s.UpdateRoleFromYaml(k8sClient, templates.DefaultPodSecurityRole); err != nil {
		return err
	}
	if err := k8s.UpdateRoleBindingFromYaml(k8sClient, templates.DefaultPodSecurityRoleBinding); err != nil {
		return err
	}
	log.Infof("[authz] Default PodSecurityPolicy Role and RoleBinding applied successfully")
	return nil
}
