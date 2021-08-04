package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func (t *Transport) handleUpdateDevices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.timerUpdatedevice.Stop()
		t.timerUpdatedevice.Reset(1 * time.Second)
	}
}

func (t *Transport) handleGetDevices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		arr, err := t.GetDevices()
		if err != nil {
			return
		}
		renderJSON(w, arr)
	}
}

// renderJSON преобразует 'v' в формат JSON и записывает результат, в виде ответа, в w.
func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
}

func (t *Transport) GetDevices() ([]LineOfData, error) {
	devices := []LineOfData{}
	t.RLock()
	for _, value := range t.ipToMac {
		devices = append(devices, value)
	}
	t.RUnlock()
	return devices, nil
}
