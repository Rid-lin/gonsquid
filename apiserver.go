package main

import "time"

func (t *Transport) runOnce(cfg *Config) {
	t.ipToMac = t.getDevices()
	t.setTimerUpdateDevice(cfg.Interval)

}

func (t *Transport) setTimerUpdateDevice(IntervalStr string) {
	t.Lock()
	interval, err := time.ParseDuration(IntervalStr)
	if err != nil {
		t.timerUpdatedevice = time.NewTimer(15 * time.Minute)
	} else {
		t.timerUpdatedevice = time.NewTimer(interval)
	}
	t.Unlock()
}
