package models

type ISyncableManager interface {
	IClusterModelManager
	GetSubManagers() []ISyncableManager
	// SyncResources(ctx context.Context, userCred mcclient.TokenCredential, cls *SCluster) error
}

type SSyncableManager struct {
	subManagers []ISyncableManager
}

func newSyncableManager() *SSyncableManager {
	return &SSyncableManager{
		subManagers: make([]ISyncableManager, 0),
	}
}

func (m *SSyncableManager) AddSubManager(mans ...ISyncableManager) *SSyncableManager {
	m.subManagers = append(m.subManagers, mans...)
	return m
}

func (m *SSyncableManager) GetSubManagers() []ISyncableManager {
	return m.subManagers
}
