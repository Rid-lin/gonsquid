package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-routeros/routeros"
	log "github.com/sirupsen/logrus"
)

type request struct {
	Time,
	IP string
	timeInt int64
}

type ResponseType struct {
	IP       string `JSON:"IP"`
	Mac      string `JSON:"Mac"`
	Hostname string `JSON:"Hostname"`
	Comment  string `JSON:"Comment"`
}

type Transport struct {
	ipToMac             map[string]LineOfData
	Location            *time.Location
	fileDestination     *os.File
	csvFiletDestination *os.File
	conn                *net.UDPConn
	clientROS           *routeros.Client
	renewOneMac         chan string
	exitChan            chan os.Signal
	QuotaType
	sync.RWMutex
}

func NewTransport(cfg *Config) *Transport {

	c, err := dial(cfg)
	if err != nil {
		log.Errorf("Error connect to %v:%v", cfg.MTAddr, err)
		c = tryingToReconnectToMokrotik(cfg)
	}
	// defer c.Close()

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
		clientROS:           c,
		fileDestination:     fileDestination,
		csvFiletDestination: csvFiletDestination,
	}
}

func dial(cfg *Config) (*routeros.Client, error) {
	if cfg.UseTLS {
		return routeros.DialTLS(cfg.MTAddr, cfg.MTUser, cfg.MTPass, nil)
	}
	return routeros.Dial(cfg.MTAddr, cfg.MTUser, cfg.MTPass)
}

func tryingToReconnectToMokrotik(cfg *Config) *routeros.Client {
	c, err := dial(cfg)
	if err != nil {
		if cfg.NumOfTryingConnectToMT == 0 {
			log.Fatalf("Error connect to %v:%v", cfg.MTAddr, err)
		} else if cfg.NumOfTryingConnectToMT < 0 {
			cfg.NumOfTryingConnectToMT = -1
		}
		log.Errorf("Error connect to %v:%v", cfg.MTAddr, err)
		time.Sleep(15 * time.Second)
		c = tryingToReconnectToMokrotik(cfg)
		cfg.NumOfTryingConnectToMT--
	}
	return c
}

func (data *Transport) GetInfo(request *request) ResponseType {
	var response ResponseType

	timeInt, err := strconv.ParseInt(request.Time, 10, 64)
	if err != nil {
		log.Errorf("Error parsing timeStamp(%v) from request:%v", timeInt, err)
		//With an incorrect time, removes 30 seconds from the current time to be able to identify the IP address
		timeInt = time.Now().Add(-30 * time.Second).Unix()
	}
	request.timeInt = timeInt
	data.RLock()
	ipStruct, ok := data.ipToMac[request.IP]
	data.RUnlock()
	// if ok && timeInt < ipStruct.timeoutInt {
	// 	log.Tracef("IP:%v to MAC:%v, hostname:%v, comment:%v", ipStruct.IP, ipStruct.Mac, ipStruct.HostName, ipStruct.Comment)
	// 	response.Mac = ipStruct.Mac
	// 	response.IP = ipStruct.IP
	// 	response.Hostname = ipStruct.HostName
	// 	response.Comment = ipStruct.Comment
	// } else
	if ok {
		log.Tracef("IP:%v to MAC:%v, hostname:%v, comment:%v", ipStruct.IP, ipStruct.Mac, ipStruct.HostName, ipStruct.Comment)
		response.Mac = ipStruct.Mac
		response.IP = ipStruct.IP
		response.Hostname = ipStruct.HostName
		response.Comment = ipStruct.Comment
		// } else if !ok {
		// 	// TODO Make information about the mac-address loaded from the router
		// 	log.Tracef("IP:'%v' not find in table lease of router:'%v'", ipStruct.IP, cfg.MTAddr)
		// 	response.Mac = request.IP
		// 	response.IP = request.IP
	} else {
		device := data.getInfoFromMTAboutIP(request.IP, &cfg)
		data.updateInfoAboutIP(device)
		log.Tracef("IP:'%v' not find in table lease of router:'%v'", ipStruct.IP, cfg.MTAddr)
		response.Mac = device.Mac
		response.IP = request.IP
	}
	if response.Mac == "" {
		response.Mac = request.IP
	}

	return response
}

func (data *Transport) getInfoFromMTAboutIP(ip string, cfg *Config) DeviceType {
	device := DeviceType{}

	reply2, err2 := data.clientROS.Run("/ip/dhcp-server/lease/print", "?active-address="+ip)
	if err2 != nil {
		log.Error(err2)
	}
	for _, re := range reply2.Re {
		device.Id = re.Map[".id"]
		device.IP = re.Map["active-address"]
		device.Mac = re.Map["mac-address"]
		device.HostName = re.Map["host-name"]
		device.Groups = re.Map["address-lists"]

	}
	return device

}

func (data *Transport) updateInfoAboutIP(device DeviceType) {
	data.RLock()
	quotahourly := data.HourlyQuota
	quotadaily := data.DailyQuota
	quotamonthly := data.MonthlyQuota
	data.RUnlock()

	lineOfData := LineOfData{}
	lineOfData.DeviceType = device
	if lineOfData.HourlyQuota == 0 {
		lineOfData.HourlyQuota = quotahourly
	}
	if lineOfData.DailyQuota == 0 {
		lineOfData.DailyQuota = quotadaily
	}
	if lineOfData.MonthlyQuota == 0 {
		lineOfData.MonthlyQuota = quotamonthly
	}

	data.Lock()
	data.ipToMac[device.IP] = lineOfData
	data.Unlock()
}

/*
Jun 22 21:39:13 192.168.65.1 dhcp,info dhcp_lan deassigned 192.168.65.149 from 04:D3:B5:FC:E8:09
Jun 22 21:40:16 192.168.65.1 dhcp,info dhcp_lan assigned 192.168.65.202 to E8:6F:38:88:92:29
*/

func (data *Transport) loopGetDataFromMT() {
	// defer func() {
	// 	if e := recover(); e != nil {
	// 		log.Errorf("Error while trying to get data from the router:%v", e)
	// 	}
	// }()
	for {

		data.Lock()
		data.ipToMac = data.getDataFromMT()
		data.Unlock()

		// ipToMac := data.getDataFromMT()
		// data.Lock()
		// data.ipToMac = ipToMac
		// data.Unlock()
		// ipToMac = map[string]LineOfData{}

		interval, err := time.ParseDuration(cfg.Interval)
		if err != nil {
			interval = 10 * time.Minute
		}
		time.Sleep(interval)

	}
}

func (data *Transport) getDataFromMT() map[string]LineOfData {

	quotahourly := data.HourlyQuota
	quotadaily := data.DailyQuota
	quotamonthly := data.MonthlyQuota

	lineOfData := LineOfData{}
	ipToMac := map[string]LineOfData{}
	reply, err := data.clientROS.Run("/ip/arp/print")
	if err != nil {
		log.Error(err)
	}
	for _, re := range reply.Re {
		lineOfData.IP = re.Map["address"]
		lineOfData.Mac = re.Map["mac-address"]
		if lineOfData.HourlyQuota == 0 {
			lineOfData.HourlyQuota = quotahourly
		}
		if lineOfData.DailyQuota == 0 {
			lineOfData.DailyQuota = quotadaily
		}
		if lineOfData.MonthlyQuota == 0 {
			lineOfData.MonthlyQuota = quotamonthly
		}
		lineOfData.timeoutInt = time.Now().Add(1 * time.Minute).Unix()
		ipToMac[lineOfData.IP] = lineOfData
	}
	reply2, err2 := data.clientROS.Run("/ip/dhcp-server/lease/print") //, "?status=bound") //, "?disabled=false")
	if err2 != nil {
		log.Error(err2)
	}
	for _, re := range reply2.Re {
		lineOfData.Id = re.Map[".id"]
		lineOfData.IP = re.Map["active-address"]
		lineOfData.Mac = re.Map["active-mac-address"]
		lineOfData.timeout = re.Map["expires-after"]
		lineOfData.HostName = re.Map["host-name"]
		lineOfData.Comment = re.Map["comment"]
		lineOfData.HourlyQuota, lineOfData.DailyQuota, lineOfData.MonthlyQuota, lineOfData.Name, lineOfData.Position, lineOfData.Company, lineOfData.TypeD = parseComments(lineOfData.Comment)
		if lineOfData.HourlyQuota == 0 {
			lineOfData.HourlyQuota = quotahourly
		}
		if lineOfData.DailyQuota == 0 {
			lineOfData.DailyQuota = quotadaily
		}
		if lineOfData.MonthlyQuota == 0 {
			lineOfData.MonthlyQuota = quotamonthly
		}
		lineOfData.disable = re.Map["disabled"]
		addressLists := re.Map["address-lists"]
		lineOfData.addressLists = strings.Split(addressLists, ",")

		//Calculating the time when the lease of the address ends
		timeStr, err := time.ParseDuration(lineOfData.timeout)
		if err != nil {
			timeStr = 10 * time.Second
		}
		// Writes to a variable for further quick comparison
		lineOfData.timeoutInt = time.Now().Add(timeStr).Unix()

		ipToMac[lineOfData.IP] = lineOfData

	}
	return ipToMac
}

func parseComments(comment string) (
	quotahourly, quotadaily, quotamonthly uint64,
	name, position, company, typeD string) {
	commentArray := strings.Split(comment, "/")
	for _, value := range commentArray {
		switch {
		case strings.Contains(value, "tel"):
			typeD = "tel"
			name = parseParamertToStr(value)
		case strings.Contains(value, "nb"):
			typeD = "nb"
			name = parseParamertToStr(value)
		case strings.Contains(value, "ws"):
			typeD = "ws"
			name = parseParamertToStr(value)
		case strings.Contains(value, "srv"):
			typeD = "srv"
			name = parseParamertToStr(value)
		case strings.Contains(value, "prn"):
			typeD = "prn"
			name = parseParamertToStr(value)
		case strings.Contains(value, "name"):
			typeD = "other"
			name = parseParamertToStr(value)
		case strings.Contains(value, "col"):
			position = parseParamertToStr(value)
		case strings.Contains(value, "com"):
			company = parseParamertToStr(value)
		case strings.Contains(value, "quotahourly"):
			quotahourly = parseParamertToUint(value)
		case strings.Contains(value, "quotadaily"):
			quotadaily = parseParamertToUint(value)
		case strings.Contains(value, "quotamonthly"):
			quotamonthly = parseParamertToUint(value)
		}
	}
	return

}

func parseParamertToStr(inpuStr string) string {
	Arr := strings.Split(inpuStr, "=")
	if len(Arr) > 1 {
		return Arr[1]
	} else {
		log.Errorf("Parameter error. The parameter is specified incorrectly or not specified at all.(%v)", inpuStr)
	}
	return ""
}

func parseParamertToUint(inputValue string) (quota uint64) {
	var err error
	Arr := strings.Split(inputValue, "=")
	if len(Arr) > 1 {
		quotaStr := Arr[1]
		quota, err = strconv.ParseUint(quotaStr, 10, 64)
		if err != nil {
			log.Errorf("Error parse quota from:(%v) with:(%v)", quotaStr, err)
		}
		return
	} else {
		log.Errorf("Parameter error. The parameter is specified incorrectly or not specified at all.(%v)", inputValue)
	}
	return
}

func (transport *Transport) syncStatusDevices(inputSync map[string]bool) {
	result := map[string]bool{}
	ipToMac := transport.getDataFromMT()
	transport.Lock()
	transport.ipToMac = ipToMac
	transport.Unlock()

	for _, value := range ipToMac {
		for keySync := range inputSync {
			if keySync == value.IP || keySync == value.Mac || keySync == value.HostName {
				result[value.Id] = inputSync[keySync]
			}
		}
	}

	for key := range result {
		err := transport.setStatusDevice(key, result[key])
		if err != nil {
			log.Error(err)
		}
	}
}

func (data *Transport) setStatusDevice(number string, status bool) error {

	var statusMtT string
	if status {
		statusMtT = "yes"
	} else {
		statusMtT = "no"
	}

	reply, err := data.clientROS.Run("/ip/dhcp-server/lease/set", "=disabled="+statusMtT, "=numbers="+number)
	if err != nil {
		return err
	} else if reply.Done.Word != "!done" {
		return fmt.Errorf("%v", reply.Done.Word)
	}
	log.Tracef("Response from Mikrotik(numbers):%v(%v)", reply, number)
	return nil
}
