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

	log "github.com/sirupsen/logrus"
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
	Location            *time.Location
	fileDestination     *os.File
	csvFiletDestination *os.File
	conn                *net.UDPConn
	timerUpdatedevice   *time.Timer
	renewOneMac         chan string
	exitChan            chan os.Signal
	Interval            string
	GomtcAddr           string
	QuotaType
	sync.RWMutex
}

func NewTransport(cfg *Config) *Transport {
	var err error

	fileDestination, err = os.OpenFile(cfg.NameFileToLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fileDestination.Close()
		log.Fatalf("Error, the '%v' file could not be created (there are not enough premissions or it is busy with another program): %v", cfg.NameFileToLog, err)
	}
	if cfg.CSV {
		csvFiletDestination, err = os.OpenFile(cfg.NameFileToLog+".csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fileDestination.Close()
			log.Fatalf("Error, the '%v' file could not be created (there are not enough premissions or it is busy with another program): %v", cfg.NameFileToLog, err)
		}
	}

	Location, err := time.LoadLocation(cfg.Loc)
	if err != nil {
		log.Errorf("Error loading Location(%v):%v", cfg.Loc, err)
		Location = time.UTC
	}

	return &Transport{
		ipToMac:             make(map[string]LineOfData),
		renewOneMac:         make(chan string, 100),
		Location:            Location,
		exitChan:            getExitSignalsChannel(),
		Interval:            cfg.Interval,
		fileDestination:     fileDestination,
		csvFiletDestination: csvFiletDestination,
		GomtcAddr:           cfg.GomtcAddr,
		QuotaType: QuotaType{
			HourlyQuota:  uint64(cfg.DefaultQuotaHourly * cfg.SizeOneMegabyte),
			DailyQuota:   uint64(cfg.DefaultQuotaDaily * cfg.SizeOneMegabyte),
			MonthlyQuota: uint64(cfg.DefaultQuotaMonthly * cfg.SizeOneMegabyte),
		},
	}
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

func (data *Transport) getDataOverApi() map[string]LineOfData {

	quotahourly := data.HourlyQuota
	quotadaily := data.DailyQuota
	quotamonthly := data.MonthlyQuota

	lineOfData := LineOfData{}
	ipToMac := map[string]LineOfData{}
	// arrDevices := []Device{}
	arrDevices, err := JSONClient(data.GomtcAddr, "/api/v1/devices")
	if err != nil {
		log.Error(err)
		return ipToMac
	}
	for _, value := range arrDevices {
		lineOfData.Device = value
		if value.HourlyQuota == 0 {
			value.HourlyQuota = quotahourly
		}
		if value.DailyQuota == 0 {
			value.DailyQuota = quotadaily
		}
		if value.MonthlyQuota == 0 {
			value.MonthlyQuota = quotamonthly
		}
		lineOfData.addressLists = strings.Split(lineOfData.AddressLists, ",")
		lineOfData.Timeout = time.Now()
		ipToMac[lineOfData.IP] = lineOfData
	}
	return ipToMac
}

func JSONClient(server, uri string) ([]Device, error) {
	url := server + uri

	spaceClient := http.Client{
		Timeout: time.Second * 10, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// req.Header.Set("User-Agent", "spacecount-tutorial")

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
	v := []Device{}
	jsonErr := json.Unmarshal(body, &v)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return v, nil
}

func (data *Transport) getDevices() map[string]LineOfData {
	return data.getDataOverApi()
}
