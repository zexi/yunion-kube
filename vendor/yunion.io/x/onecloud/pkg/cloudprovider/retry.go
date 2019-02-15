package cloudprovider

import (
	"strings"
	"time"
)

func IsError(err error, errs []string) bool {
	for i := range errs {
		if strings.Index(err.Error(), errs[i]) >= 0 {
			return true
		}
	}
	return false
}

func RetryOnError(tryFunc func() error, errs []string, maxTries int) error {
	tried := 0
	for tried < maxTries {
		err := tryFunc()
		if err == nil {
			return nil
		}
		if err != nil && !IsError(err, errs) {
			return err
		}
		tried += 1
		time.Sleep(10 * time.Duration(tried) * time.Second)
	}
	return ErrTimeout
}

func RetryUntil(tryFunc func() (bool, error), maxTries int) error {
	tried := 0
	for tried < maxTries {
		stop, err := tryFunc()
		if stop {
			return nil
		}
		if err != nil {
			return err
		}
		tried += 1
		time.Sleep(10 * time.Duration(tried) * time.Second)
	}
	return ErrTimeout
}
