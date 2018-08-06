package cloudprovider

import "time"

func WaitStatus(res ICloudResource, expect string, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			return err
		}
		if res.GetStatus() == expect {
			return nil
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}

func WaitDeleted(res ICloudResource, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			if err == ErrNotFound {
				return nil
			} else {
				return err
			}
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}