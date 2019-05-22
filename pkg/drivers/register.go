package drivers

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type DriverManager struct {
	*sync.Map
	keySep string
}

func NewDriverManager(keySep string) *DriverManager {
	if len(keySep) == 0 {
		keySep = "->"
	}
	man := &DriverManager{
		Map:    new(sync.Map),
		keySep: keySep,
	}
	return man
}

func (m *DriverManager) getIndexKey(keys ...string) string {
	return strings.Join(keys, m.keySep)
}

func (m *DriverManager) Register(drv interface{}, keys ...string) error {
	key := m.getIndexKey(keys...)
	_, ok := m.Load(key)
	if ok {
		return errors.New(fmt.Sprintf("Driver %s already register", key))
	}
	m.Store(key, drv)
	return nil
}

func (m *DriverManager) Get(keys ...string) (interface{}, error) {
	key := m.getIndexKey(keys...)
	drv, ok := m.Load(key)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Not found driver by key: %v", key))
	}
	return drv, nil
}
