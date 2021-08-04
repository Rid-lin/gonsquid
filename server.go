package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type responseWriter struct {
	http.ResponseWriter
	code int
}

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

func (t *Transport) configureRouter() {
	t.router.Use(t.logRequest)
	t.router.HandleFunc("/api/v1/updatedevices", t.handleUpdateDevices()).Methods("GET") // Update cashe of devices from Mikrotik
	t.router.HandleFunc("/api/v1/devices", t.handleGetDevices()).Methods("GET")          // Get all devices from lease or ARP
}

func (t *Transport) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var host string
		var hostArr []string
		var ok bool
		hostArr, ok = r.Header["X-Forwarded-For"]
		if !ok {
			hostPort := r.RemoteAddr
			hostArr = strings.Split(hostPort, ":")
		}
		if len(hostArr) > 0 {
			host = hostArr[0]
		}
		logger := logrus.WithFields(logrus.Fields{
			"remote_addr": host,
			// "request_id":  r.Context().Value(ctxKeyRequestID),
		})
		logger.Infof("started %s %s", r.Method, r.RequestURI)
		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		var level logrus.Level
		switch {
		case rw.code >= 500:
			level = logrus.ErrorLevel
		case rw.code >= 400:
			level = logrus.WarnLevel
		default:
			level = logrus.InfoLevel
		}
		logger.Logf(
			level,
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Since(start),
		)
	})
}
