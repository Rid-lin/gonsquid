package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NetFlow v5 implementation

type header struct {
	Version          uint16
	FlowRecords      uint16
	Uptime           uint32
	UnixSec          uint32
	UnixNsec         uint32
	FlowSeqNum       uint32
	EngineType       uint8
	EngineID         uint8
	SamplingInterval uint16
}

type binaryRecord struct {
	Ipv4SrcAddrInt uint32
	Ipv4DstAddrInt uint32
	Ipv4NextHopInt uint32
	InputSnmp      uint16
	OutputSnmp     uint16
	InPkts         uint32
	InBytes        uint32
	FirstInt       uint32
	LastInt        uint32
	L4SrcPort      uint16
	L4DstPort      uint16
	_              uint8
	TCPFlags       uint8
	Protocol       uint8
	SrcTos         uint8
	SrcAs          uint16
	DstAs          uint16
	SrcMask        uint8
	DstMask        uint8
	_              uint16
}

type decodedRecord struct {
	header
	binaryRecord

	Host              string
	SamplingAlgorithm uint8
	Ipv4SrcAddr       string
	Ipv4DstAddr       string
	Ipv4NextHop       string
	SrcHostName       string
	DstHostName       string
	Duration          uint16
}

func intToIPv4Addr(intAddr uint32) net.IP {

	return net.IPv4(
		byte(intAddr>>24),
		byte(intAddr>>16),
		byte(intAddr>>8),
		byte(intAddr))
}

func decodeRecord(header *header, binRecord *binaryRecord, remoteAddr *net.UDPAddr, cfg *Config) decodedRecord {

	decodedRecord := decodedRecord{

		Host: remoteAddr.IP.String(),

		header: *header,

		binaryRecord: *binRecord,

		Ipv4SrcAddr: intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(),
		Ipv4DstAddr: intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(),
		Ipv4NextHop: intToIPv4Addr(binRecord.Ipv4NextHopInt).String(),
		Duration:    uint16((binRecord.LastInt - binRecord.FirstInt) / 1000),
	}

	// decode sampling info
	decodedRecord.SamplingAlgorithm = uint8(0x3 & (decodedRecord.SamplingInterval >> 14))
	decodedRecord.SamplingInterval = 0x3fff & decodedRecord.SamplingInterval

	return decodedRecord
}

func (t *Transport) decodeRecordToSquid(record *decodedRecord, cfg *Config) (string, string) {
	binRecord := record.binaryRecord
	header := record.header
	remoteAddr := record.Host
	srcmacB := make([]byte, 8)
	dstmacB := make([]byte, 8)
	binary.BigEndian.PutUint16(srcmacB, binRecord.SrcAs)
	binary.BigEndian.PutUint16(dstmacB, binRecord.DstAs)
	// srcmac = srcmac[2:8]
	// dstmac = dstmac[2:8]

	var protocol, message, message2 string

	switch fmt.Sprintf("%v", binRecord.Protocol) {
	case "6":
		protocol = "TCP_PACKET"
	case "17":
		protocol = "UDP_PACKET"
	case "1":
		protocol = "ICMP_PACKET"

	default:
		protocol = "OTHER_PACKET"
	}

	tm := time.Unix(int64(header.UnixSec), 0).In(cfg.Location)
	year := tm.Year()
	month := tm.Month()
	day := tm.Day()
	hour := tm.Hour()
	minute := tm.Minute()
	second := tm.Second()
	tz := tm.Format("-0700")

	ok := cfg.CheckEntryInSubNet(intToIPv4Addr(binRecord.Ipv4DstAddrInt))
	ok2 := cfg.CheckEntryInSubNet(intToIPv4Addr(binRecord.Ipv4SrcAddrInt))

	if ok && !ok2 {
		ipDst := intToIPv4Addr(binRecord.Ipv4DstAddrInt).String()
		if inIgnor(ipDst, cfg.IgnorList) {
			return "", ""
		}
		response := t.GetInfo(&request{
			IP:   ipDst,
			Time: fmt.Sprint(header.UnixSec)})
		message = fmt.Sprintf("%v.000 %6v %v %v/- %v HEAD %v:%v %v FIRSTUP_PARENT/%v packet_netflow/:%v %v %v",
			header.UnixSec,                       // time
			binRecord.LastInt-binRecord.FirstInt, //delay
			ipDst,                                // dst ip
			protocol,                             // protocol
			binRecord.InBytes,                    // size
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), //src ip
			binRecord.L4SrcPort, // src port
			response.Mac,        // dstmac
			remoteAddr,          // routerIP
			// net.HardwareAddr(srcmacB).String(), // srcmac
			binRecord.L4DstPort, // dstport
			response.HostName,
			response.Comments,
		)
		// 1628047627|2021|Aug|04|08|27|07|+0500|44459|9193|192.168.65.195|192.168.65.195|53062|c8:58:c0:38:68:a5|94.100.180.59|443|portal.mail.ru:443|TCP_TUNNEL|200|CONNECT|C8:58:C0:38:68:A5|HIER_DIRECT/94.100.180.59|-

		message2 = fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v:%v|%v|%v|%v|%v|%v/%v|%v",
			header.UnixSec,                       // unix timestamp 1628047627
			year,                                 // year - 2021
			month.String(),                       //month Aug
			day,                                  // day 04
			hour,                                 //hour 08
			minute,                               // minute 27
			second,                               // second 07
			tz,                                   //timezone +0500
			binRecord.LastInt-binRecord.FirstInt, // delay 44459
			binRecord.InBytes,                    // size 9193
			ipDst,                                // dst ip 192.168.65.195
			response.HostName,                    // dst ip 192.168.65.195
			binRecord.L4DstPort,                  // dst port 53062
			response.Mac,                         // dstmac c8:58:c0:38:68:a5
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), // src ip 94.100.180.59
			binRecord.L4SrcPort, // src port 443
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), binRecord.L4SrcPort, // src ip portal.mail.ru:443
			"NF_PACKET", // protocol TCP_TUNNEL
			"200",       // 200
			protocol,    // CONNECT
			response.Mac,
			"DATA_FROM",
			remoteAddr, // routerIP 94.100.180.59
			"-",
		)

		// message2 = fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v,non_inverse,%v",
		// 	header.UnixSec,                       // time
		// 	binRecord.LastInt-binRecord.FirstInt, // delay
		// 	binRecord.InBytes,                    // size
		// 	protocol,                             // protocol
		// 	remoteAddr,                           // routerIP
		// 	intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip
		// 	binRecord.L4SrcPort, // src port
		// 	response.Mac,        // dstmac
		// 	response.HostName,
		// 	intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), // src ip
		// 	binRecord.L4DstPort, // dstport
		// 	response.Comments,
		// )

	} else if !ok && ok2 {
		ipDst := intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String()
		if inIgnor(ipDst, cfg.IgnorList) {
			return "", ""
		}
		response := t.GetInfo(&request{
			IP:   ipDst,
			Time: fmt.Sprint(header.UnixSec)})
		message = fmt.Sprintf("%v.000 %6v %v %v/- %v HEAD %v:%v %v FIRSTUP_PARENT/%v packet_netflow_inverse/:%v %v %v",
			header.UnixSec,                       // time
			binRecord.LastInt-binRecord.FirstInt, //delay
			ipDst,                                //src ip - Local
			protocol,                             // protocol
			binRecord.InBytes,                    // size
			intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip - Inet
			binRecord.L4DstPort, // dstport
			response.Mac,        // dstmac
			remoteAddr,          // routerIP
			// net.HardwareAddr(srcmacB).String(), // srcmac
			binRecord.L4SrcPort, // src port
			response.HostName,
			response.Comments,
		)
		message2 = fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v|%v:%v|%v|%v|%v|%v|%v/%v|%v",
			header.UnixSec,                       // unix timestamp 1628047627
			year,                                 // year - 2021
			month.String(),                       //month Aug
			day,                                  // day 04
			hour,                                 //hour 08
			minute,                               // minute 27
			second,                               // second 07
			tz,                                   //timezone +0500
			binRecord.LastInt-binRecord.FirstInt, // delay 44459
			binRecord.InBytes,                    // size 9193
			ipDst,                                // dst ip 192.168.65.195
			response.HostName,                    // dst ip 192.168.65.195
			binRecord.L4SrcPort,                  // dst port 53062
			response.Mac,                         // dstmac c8:58:c0:38:68:a5
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), // src ip 94.100.180.59
			binRecord.L4DstPort, // src port 443
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), binRecord.L4DstPort, // src ip portal.mail.ru:443
			"NF_I_PACKET", // protocol TCP_TUNNEL
			"200",         // 200
			protocol,      // CONNECT
			response.Mac,
			"DATA_FROM",
			remoteAddr, // routerIP 94.100.180.59
			"-",
		)

		// message2 = fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,inverse,%v",
		// 	header.UnixSec,                       // time
		// 	binRecord.LastInt-binRecord.FirstInt, //delay
		// 	binRecord.InBytes,                    // size
		// 	protocol,                             // protocol
		// 	remoteAddr,                           // routerIP
		// 	ipDst, 									//src ip - Local (reverses dst ip)
		// 	binRecord.L4SrcPort, 					// src port (reverses dst port)
		// 	response.Mac,        					// dstmac (reverses src mac)
		// 	response.HostName,
		// 	intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip - Inet  (reverses src ip)
		// 	binRecord.L4DstPort, // dstport  (reverses src port)
		// 	response.Comments,
		// )

	}
	return message, message2
}

func (cfg *Config) CheckEntryInSubNet(ipv4addr net.IP) bool {
	for _, subNet := range cfg.SubNets {
		ok, err := checkIP(subNet, ipv4addr)
		if err != nil { // если ошибка, то следующая строка
			logrus.Error("Error while determining the IP subnet address:", err)
			return false

		}
		if ok {
			return true
		}
	}

	return false
}

func checkIP(subnet string, ipv4addr net.IP) (bool, error) {
	_, netA, err := net.ParseCIDR(subnet)
	if err != nil {
		return false, err
	}

	return netA.Contains(ipv4addr), nil
}

func (t *Transport) pipeOutputToStdoutForSquid(outputChannel chan decodedRecord, cfg *Config) {
	for record := range outputChannel {
		logrus.Tracef("Get from outputChannel:%v", record)
		message, csvMessage := t.decodeRecordToSquid(&record, cfg)
		logrus.Tracef("Decoded record (%v) to message (%v)", record, message)
		message = filtredMessage(message, cfg.IgnorList)
		if message == "" {
			continue
		}
		if _, err := t.fileDestination.WriteString(message + "\n"); err != nil {
			logrus.Errorf("Error writing data buffer:%v", err)
		} else {
			logrus.Tracef("Added to log:%v", message)
		}
		if cfg.CSV {
			if _, err := t.csvFiletDestination.WriteString(csvMessage + "\n"); err != nil {
				logrus.Errorf("Error writing data buffer:%v", err)
			} else {
				logrus.Tracef("Added to CSV:%v", message)
			}
		}
	}
}

func filtredMessage(message string, IgnorList []string) string {
	for _, ignorStr := range IgnorList {
		if strings.Contains(message, ignorStr) {
			logrus.Tracef("Line of log :%v contains ignorstr:%v, skipping...", message, ignorStr)
			return ""
		}
	}
	return message
}

func inIgnor(message string, IgnorList []string) bool {
	for _, ignorStr := range IgnorList {
		if strings.Contains(message, ignorStr) {
			logrus.Tracef("Line of log :%v contains ignorstr:%v, skipping...", message, ignorStr)
			return true
		}
	}
	return false
}

// type cacheRecord struct {
// 	Hostname string
// 	// timeout  time.Time
// }

// type Cache struct {
// 	cache map[string]cacheRecord
// 	sync.RWMutex
// }

var (
	// cache Cache
	// cache      = map[string]cacheRecord{}
	// cacheMutex = sync.RWMutex{}
	// writer           *bufio.Writer
	fileDestination     *os.File
	csvFiletDestination *os.File
)

func handlePacket(buf *bytes.Buffer, remoteAddr *net.UDPAddr, outputChannel chan decodedRecord, cfg *Config) {
	header := header{}
	err := binary.Read(buf, binary.BigEndian, &header)
	if err != nil {
		logrus.Printf("Error: %v\n", err)
	} else {

		for i := 0; i < int(header.FlowRecords); i++ {
			record := binaryRecord{}
			err := binary.Read(buf, binary.BigEndian, &record)
			if err != nil {
				logrus.Printf("binary.Read failed: %v\n", err)
				break
			}

			decodedRecord := decodeRecord(&header, &record, remoteAddr, cfg)
			logrus.Tracef("Send to outputChannel:%v", decodedRecord)
			outputChannel <- decodedRecord
		}
	}
}
