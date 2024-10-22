package main

import (
	"bytes"
	"net"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		err error
	)

	cfg := newConfig()

	cache.cache = make(map[string]cacheRecord)

	data := NewTransport(cfg)
	/*Creating a channel to intercept the program end signal*/
	// exitChan := getExitSignalsChannel()

	data.HourlyQuota = uint64(cfg.DefaultQuotaHourly * cfg.SizeOneMegabyte)
	data.DailyQuota = uint64(cfg.DefaultQuotaDaily * cfg.SizeOneMegabyte)
	data.MonthlyQuota = uint64(cfg.DefaultQuotaMonthly * cfg.SizeOneMegabyte)

	go data.loopGetDataFromMT()

	http.HandleFunc("/", logreq(handleIndex))
	http.HandleFunc("/getmac", logreq(data.handlerGetMac()))
	http.HandleFunc("/setstatusdevices", logreq(data.handlerSetStatusDevices))
	http.HandleFunc("/getstatusdevices", logreq(data.handlerGetStatusDevices))

	log.Infof("gonsquid listens to:%v", cfg.BindAddr)

	go func() {
		err := http.ListenAndServe(cfg.BindAddr, nil)
		if err != nil {
			log.Error("http-server returned error:", err)
		}
	}()

	go data.Exit()

	/* Create output pipe */
	outputChannel := make(chan decodedRecord, 100)

	go data.pipeOutputToStdoutForSquid(outputChannel, cfg)

	/* Start listening on the specified port */
	log.Infof("Start listening to NetFlow stream on %v", cfg.FlowAddr)
	addr, err := net.ResolveUDPAddr("udp", cfg.FlowAddr)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	for {
		data.conn, err = net.ListenUDP("udp", addr)
		if err != nil {
			log.Errorln(err)
		} else {
			err = data.conn.SetReadBuffer(cfg.ReceiveBufferSizeBytes)
			if err != nil {
				log.Errorln(err)
			} else {
				/* Infinite-loop for reading packets */
				for {
					buf := make([]byte, 4096)
					rlen, remote, err := data.conn.ReadFromUDP(buf)

					if err != nil {
						log.Errorf("Error: %v\n", err)
					} else {

						stream := bytes.NewBuffer(buf[:rlen])

						go handlePacket(stream, remote, outputChannel, cfg)
					}
				}
			}
		}

	}
}
