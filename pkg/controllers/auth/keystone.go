package auth

import (
	"fmt"
	"sync"
	"time"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient/auth"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/util/wait"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"

	o "yunion.io/yunion-kube/pkg/options"
)

const (
	defaultTimeout             = 600
	roleAssignmentsResetPeriod = 1 * time.Hour
)

type KeystoneAuthenticator struct {
	reconciler          *Reconciler
	roleAssignmentsLock *sync.Mutex
	roleAssignments     RoleAssignments
}

// NewKeystoneAuthenticator returns a password authenticator that validates credentials using keystone
func NewKeystoneAuthenticator(k8sCli kubernetes.Interface) (*KeystoneAuthenticator, error) {
	k := &KeystoneAuthenticator{
		roleAssignmentsLock: new(sync.Mutex),
	}
	err := k.ResetRoleAssignments()
	if err != nil {
		return k, fmt.Errorf("New keystone auth: %v", err)
	}

	reconciler := NewReconciler(k, k8sCli)
	k.reconciler = reconciler

	go wait.Forever(func() {
		err := k.ResetRoleAssignments()
		if err != nil {
			log.Errorf("ResetRoleAssignments error: %v", err)
			return
		}
		log.Infof("ResetRoleAssignments success.")
	}, roleAssignmentsResetPeriod)
	return k, nil
}

func (k *KeystoneAuthenticator) GetRoleAssignments() RoleAssignments {
	k.roleAssignmentsLock.Lock()
	defer k.roleAssignmentsLock.Unlock()

	return k.roleAssignments
}

func (k *KeystoneAuthenticator) ResetRoleAssignments() error {
	k.roleAssignmentsLock.Lock()
	defer k.roleAssignmentsLock.Unlock()

	ras, err := getRoleAssignments()
	if err != nil {
		log.Errorf("ResetRoleAssignments error: %v", err)
		return err
	}
	k.roleAssignments = ras
	return nil
}

func getRoleAssignments() (ret RoleAssignments, err error) {
	s := auth.AdminSession(o.Options.Region, "", "", "")
	if s == nil {
		err = fmt.Errorf("Can not get auth adminSession")
		return
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONNull, "include_names")
	data, err := modules.RoleAssignments.List(s, query)
	if err != nil {
		err = fmt.Errorf("List RoleAssignments error: %v", err)
		return
	}
	ret, err = NewRoleAssignmentsByJSON(jsonutils.NewArray(data.Data...))
	return
}

// AuthenticateToken checks the token via Keystone call
func (k *KeystoneAuthenticator) AuthenticateToken(token string) (user.Info, bool, error) {
	cred, err := auth.Verify(token)
	if err != nil {
		err = fmt.Errorf("Failed to verify token %q: %v", token, err)
		log.Errorf("%v", err)
		return nil, false, err
	}

	userName := cred.GetUserName()
	userId := cred.GetUserId()

	ras, err := k.reconciler.ReconcileRBAC(userName, userId)
	if err != nil {
		log.Errorf("ReconcileRBAC for user: %s, id: %s, error: %v", userName, userId, err)
		return nil, false, err
	}

	projects := ras.Projects().List()

	extra := map[string][]string{
		"alpha.kubernetes.io/identity/roles":        cred.GetRoles(),
		"alpha.kubernetes.io/identity/project/id":   ras.ProjectIDs(),
		"alpha.kubernetes.io/identity/project/name": projects,
	}
	authenticatedUser := &user.DefaultInfo{
		Name:   userName,
		UID:    userId,
		Groups: []string{cred.GetProjectName()},
		Extra:  extra,
	}
	return authenticatedUser, true, nil
}
