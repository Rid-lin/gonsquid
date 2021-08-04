package main

import (
	"bytes"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		err error
	)

	cfg := newConfig()

	// cache.cache = make(map[string]cacheRecord)

	t := NewTransport(cfg)
	/*Creating a channel to intercept the program end signal*/

	go t.Exit(cfg)

	// Endless file parsing loop
	go func(cfg *Config) {
		t.runOnce(cfg)
		for {
			<-t.timerUpdatedevice.C
			t.runOnce(cfg)
		}
	}(cfg)

	go func(cfg *Config, t *Transport) {
		if err := http.ListenAndServe(cfg.BindAddr, t.router); err != nil {
			logrus.Fatal(err)
		}
	}(cfg, t)

	/* Create output pipe */
	outputChannel := make(chan decodedRecord, 100)

	go t.pipeOutputToStdoutForSquid(outputChannel, cfg)

	/* Start listening on the specified port */
	logrus.Infof("Start listening to NetFlow stream on %v", cfg.FlowAddr)
	addr, err := net.ResolveUDPAddr("udp", cfg.FlowAddr)
	if err != nil {
		logrus.Fatalf("Error: %v\n", err)
	}

	for {
		t.conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			logrus.Errorln(err, "Sleeping 5 second")
			time.Sleep(5 * time.Second)
		} else {
			err = t.conn.SetReadBuffer(cfg.ReceiveBufferSizeBytes)
			if err != nil {
				logrus.Errorln(err, "Sleeping 2 second")
				time.Sleep(2 * time.Second)
			} else {
				/* Infinite-loop for reading packets */
				for {
					buf := make([]byte, 4096)
					rlen, remote, err := t.conn.ReadFromUDP(buf)

					if err != nil {
						logrus.Errorf("Error: %v\n", err)
					} else {

						stream := bytes.NewBuffer(buf[:rlen])

						go handlePacket(stream, remote, outputChannel, cfg)
					}
				}
			}
		}

	}
}
