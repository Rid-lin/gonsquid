package main

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/sirupsen/logrus"
)

type request struct {
	Time,
	IP string
	// timeInt int64
}

type ResponseType struct {
	IP       string `JSON:"IP"`
	Mac      string `JSON:"Mac"`
	HostName string `JSON:"Hostname"`
	Comments string `JSON:"Comment"`
}

type Transport struct {
	ipToMac             map[string]LineOfData
	router              *mux.Router
	Location            *time.Location
	fileDestination     *os.File
	csvFiletDestination *os.File
	conn                *net.UDPConn
	timerUpdatedevice   *time.Timer
	renewOneMac         chan string
	exitChan            chan os.Signal
	Interval            string
	GomtcAddr           string
	// QuotaType
	sync.RWMutex
}

func NewTransport(cfg *Config) *Transport {
	var err error

	fileDestination, err = os.OpenFile(cfg.NameFileToLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fileDestination.Close()
		logrus.Fatalf("Error, the '%v' file could not be created (there are not enough premissions or it is busy with another program): %v", cfg.NameFileToLog, err)
	}
	if cfg.CSV {
		csvFiletDestination, err = os.OpenFile(cfg.NameFileToLog+".csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fileDestination.Close()
			logrus.Fatalf("Error, the '%v' file could not be created (there are not enough premissions or it is busy with another program): %v", cfg.NameFileToLog, err)
		}
	}

	Location, err := time.LoadLocation(cfg.Loc)
	if err != nil {
		logrus.Errorf("Error loading Location(%v):%v", cfg.Loc, err)
		Location = time.UTC
	}

	t := &Transport{
		ipToMac:             make(map[string]LineOfData),
		renewOneMac:         make(chan string, 100),
		router:              mux.NewRouter(),
		Location:            Location,
		exitChan:            getExitSignalsChannel(),
		Interval:            cfg.Interval,
		fileDestination:     fileDestination,
		csvFiletDestination: csvFiletDestination,
		GomtcAddr:           cfg.GomtcAddr,
		// QuotaType: QuotaType{
		// 	HourlyQuota:  uint64(cfg.DefaultQuotaHourly * cfg.SizeOneMegabyte),
		// 	DailyQuota:   uint64(cfg.DefaultQuotaDaily * cfg.SizeOneMegabyte),
		// 	MonthlyQuota: uint64(cfg.DefaultQuotaMonthly * cfg.SizeOneMegabyte),
		// },
	}
	t.configureRouter()
	return t
}

func (data *Transport) GetInfo(request *request) ResponseType {
	var response ResponseType
	data.RLock()
	ipStruct, ok := data.ipToMac[request.IP]
	data.RUnlock()
	if ok {
		response.Mac = ipStruct.Mac
		if response.Mac == "" {
			response.Mac = ipStruct.ActiveMacAddress
		}
	} else {
		data.timerUpdatedevice.Stop()
		data.setTimerUpdateDevice(data.Interval)
	}
	if response.Mac == "" {
		response.Mac = request.IP
	}

	return response
}

func getDataOverApi(
	// qh, qd, qm uint64,
	addr string) map[string]LineOfData {
	lineOfData := LineOfData{}
	ipToMac := map[string]LineOfData{}
	// arrDevices := []Device{}
	arrDevices, err := JSONClient(addr, "/api/v1/devices")
	if err != nil {
		logrus.Error(err)
		return ipToMac
	}
	for _, value := range arrDevices {
		lineOfData.Device = value
		// if value.HourlyQuota == 0 {
		// 	value.HourlyQuota = qh
		// }
		// if value.DailyQuota == 0 {
		// 	value.DailyQuota = qd
		// }
		// if value.MonthlyQuota == 0 {
		// 	value.MonthlyQuota = qm
		// }
		lineOfData.addressLists = strings.Split(lineOfData.AddressLists, ",")
		lineOfData.Timeout = time.Now()
		ipToMac[lineOfData.IP] = lineOfData
	}
	return ipToMac
}

func JSONClient(server, uri string) ([]Device, error) {
	url := server + uri

	spaceClient := http.Client{
		Timeout: time.Second * 10, // Timeout after 10 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		return nil, getErr
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, readErr
	}
	d := []Device{}
	jsonErr := json.Unmarshal(body, &d)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return d, nil
}

func (t *Transport) getDevices() map[string]LineOfData {
	t.RLock()
	// qh := t.HourlyQuota
	// qd := t.DailyQuota
	// qm := t.MonthlyQuota
	addr := t.GomtcAddr
	t.RUnlock()
	return getDataOverApi(
		// qh, qd, qm,
		addr)
}
