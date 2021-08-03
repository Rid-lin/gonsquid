package main

import (
	"bytes"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		err error
	)

	cfg := newConfig()

	// cache.cache = make(map[string]cacheRecord)

	data := NewTransport(cfg)
	/*Creating a channel to intercept the program end signal*/

	// Endless file parsing loop
	go func(cfg *Config) {
		data.runOnce(cfg)
		for {
			<-data.timerUpdatedevice.C
			data.runOnce(cfg)
		}
	}(cfg)

	go data.Exit(cfg)

	/* Create output pipe */
	outputChannel := make(chan decodedRecord, 100)

	go data.pipeOutputToStdoutForSquid(outputChannel, cfg)

	/* Start listening on the specified port */
	logrus.Infof("Start listening to NetFlow stream on %v", cfg.FlowAddr)
	addr, err := net.ResolveUDPAddr("udp", cfg.FlowAddr)
	if err != nil {
		logrus.Fatalf("Error: %v\n", err)
	}

	for {
		data.conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			logrus.Errorln(err, "Sleeping 5 second")
			time.Sleep(5 * time.Second)
		} else {
			err = data.conn.SetReadBuffer(cfg.ReceiveBufferSizeBytes)
			if err != nil {
				logrus.Errorln(err, "Sleeping 2 second")
				time.Sleep(2 * time.Second)
			} else {
				/* Infinite-loop for reading packets */
				for {
					buf := make([]byte, 4096)
					rlen, remote, err := data.conn.ReadFromUDP(buf)

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
