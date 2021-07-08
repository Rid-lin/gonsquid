package main

import "time"

func (data *Transport) runOnce(cfg *Config) {
	data.Lock()
	data.ipToMac = data.getDevices()
	data.setTimerUpdateDevice(cfg.Interval)
	data.Unlock()
}

func (t *Transport) setTimerUpdateDevice(IntervalStr string) {
	interval, err := time.ParseDuration(IntervalStr)
	if err != nil {
		t.timerUpdatedevice = time.NewTimer(15 * time.Minute)
	} else {
		t.timerUpdatedevice = time.NewTimer(interval)
	}
}
