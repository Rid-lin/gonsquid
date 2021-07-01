package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
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

	ok := cfg.CheckEntryInSubNet(intToIPv4Addr(binRecord.Ipv4DstAddrInt))
	ok2 := cfg.CheckEntryInSubNet(intToIPv4Addr(binRecord.Ipv4SrcAddrInt))

	if ok && !ok2 {
		response := t.GetInfo(&request{
			IP:   intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(),
			Time: fmt.Sprint(header.UnixSec)})
		message = fmt.Sprintf("%v.000 %6v %v %v/- %v HEAD %v:%v %v FIRSTUP_PARENT/%v packet_netflow/:%v %v %v",
			header.UnixSec,                                   // time
			binRecord.LastInt-binRecord.FirstInt,             //delay
			intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip
			protocol,          // protocol
			binRecord.InBytes, // size
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), //src ip
			binRecord.L4DstPort, // dstport
			response.Mac,        // dstmac
			remoteAddr,          // routerIP
			// net.HardwareAddr(srcmacB).String(), // srcmac
			binRecord.L4SrcPort, // src port
			response.HostName,
			response.Comments,
		)
		message2 = fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,non_inverse,%v",
			header.UnixSec,                       // time
			binRecord.LastInt-binRecord.FirstInt, // delay
			binRecord.InBytes,                    // size
			protocol,                             // protocol
			remoteAddr,                           // routerIP
			intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip
			binRecord.L4DstPort, // dstport
			response.Mac,        // dstmac
			response.HostName,
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), // src ip
			binRecord.L4SrcPort, // src port
			response.Comments,
		)

	} else if !ok && ok2 {
		response := t.GetInfo(&request{
			IP:   intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(),
			Time: fmt.Sprint(header.UnixSec)})
		message = fmt.Sprintf("%v.000 %6v %v %v/- %v HEAD %v:%v %v FIRSTUP_PARENT/%v packet_netflow_inverse/:%v %v %v",
			header.UnixSec,                                   // time
			binRecord.LastInt-binRecord.FirstInt,             //delay
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), //src ip - Local
			protocol,          // protocol
			binRecord.InBytes, // size
			intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip - Inet
			binRecord.L4DstPort, // dstport
			response.Mac,        // dstmac
			remoteAddr,          // routerIP
			// net.HardwareAddr(srcmacB).String(), // srcmac
			binRecord.L4SrcPort, // src port
			response.HostName,
			response.Comments,
		)
		message2 = fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,inverse,%v",
			header.UnixSec,                       // time
			binRecord.LastInt-binRecord.FirstInt, //delay
			binRecord.InBytes,                    // size
			protocol,                             // protocol
			remoteAddr,                           // routerIP
			intToIPv4Addr(binRecord.Ipv4SrcAddrInt).String(), //src ip - Local (reverses dst ip)
			binRecord.L4SrcPort, // src port (reverses dst port)
			response.Mac,        // dstmac (reverses src mac)
			response.HostName,
			intToIPv4Addr(binRecord.Ipv4DstAddrInt).String(), // dst ip - Inet  (reverses src ip)
			binRecord.L4DstPort, // dstport  (reverses src port)
			response.Comments,
		)

	}
	return message, message2
}

func (cfg *Config) CheckEntryInSubNet(ipv4addr net.IP) bool {
	for _, subNet := range cfg.SubNets {
		ok, err := checkIP(subNet, ipv4addr)
		if err != nil { // если ошибка, то следующая строка
			log.Error("Error while determining the IP subnet address:", err)
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

func (data *Transport) pipeOutputToStdoutForSquid(outputChannel chan decodedRecord, cfg *Config) {
	for record := range outputChannel {
		log.Tracef("Get from outputChannel:%v", record)
		message, csvMessage := data.decodeRecordToSquid(&record, cfg)
		log.Tracef("Decoded record (%v) to message (%v)", record, message)
		message = filtredMessage(message, cfg.IgnorList)
		if message == "" {
			continue
		}
		if _, err := data.fileDestination.WriteString(message + "\n"); err != nil {
			log.Errorf("Error writing data buffer:%v", err)
		} else {
			log.Tracef("Added to log:%v", message)
		}
		if cfg.CSV {
			if _, err := data.csvFiletDestination.WriteString(csvMessage + "\n"); err != nil {
				log.Errorf("Error writing data buffer:%v", err)
			} else {
				log.Tracef("Added to CSV:%v", message)
			}
		}
	}
}

func filtredMessage(message string, IgnorList []string) string {
	for _, ignorStr := range IgnorList {
		if strings.Contains(message, ignorStr) {
			log.Tracef("Line of log :%v contains ignorstr:%v, skipping...", message, ignorStr)
			return ""
		}
	}
	return message
}

type cacheRecord struct {
	Hostname string
	// timeout  time.Time
}

type Cache struct {
	cache map[string]cacheRecord
	sync.RWMutex
}

var (
	cache Cache
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
		log.Printf("Error: %v\n", err)
	} else {

		for i := 0; i < int(header.FlowRecords); i++ {
			record := binaryRecord{}
			err := binary.Read(buf, binary.BigEndian, &record)
			if err != nil {
				log.Printf("binary.Read failed: %v\n", err)
				break
			}

			decodedRecord := decodeRecord(&header, &record, remoteAddr, cfg)
			log.Tracef("Send to outputChannel:%v", decodedRecord)
			outputChannel <- decodedRecord
		}
	}
}
