package models

import (
	"fmt"
)

type SFederatedManagedResourceBaseManager struct {
	federatedManager IFederatedModelManager
}

type SFederatedManagedResourceBase struct {
	FederatedResourceId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (m *SFederatedManagedResourceBaseManager) RegisterFederatedManager(fm IFederatedModelManager) {
	if m.federatedManager != nil {
		panic(fmt.Sprintf("federatedManager %s already registered", m.federatedManager.Keyword()))
	}
	m.federatedManager = fm
}

func (obj *SFederatedManagedResourceBase) SetFederatedResourceId(resId string) {
	obj.FederatedResourceId = resId
}
