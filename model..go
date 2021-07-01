package main

import (
	"time"
)

type QuotaType struct {
	HourlyQuota  uint64
	DailyQuota   uint64
	MonthlyQuota uint64
	Blocked      bool
}

type LineOfData struct {
	Comment,
	addressLists []string
	Device
}

//Device ...
type Device struct {
	// From MT
	// NotNeeded
	// activeClientId      string // 1:e8:d8:d1:47:55:93
	// allowDualStackQueue string
	// activeServer        string // dhcp_lan
	// address             string // pool_admin
	// clientId            string // 1:e8:d8:d1:47:55:93
	// dhcpOption          string //
	// dynamic             string // false
	// expiresAfter        string // 6m32s
	// lastSeen            string // 3m28s
	// radius              string // false
	// server              string // dhcp_lan
	// status              string // bound
	// insertQueueBefore   string
	// rateLimit           string
	// useSrcMac           string
	// agentCircuitId      string
	// blockAccess         string
	// leaseTime           string
	// agentRemoteId       string
	// dhcpOptionSet       string
	// srcMacAddress       string
	// alwaysBroadcast     string
	Id               string
	ActiveAddress    string // 192.168.65.85
	ActiveMacAddress string // E8:D8:D1:47:55:93
	AddressLists     string // inet
	MacAddress       string // E8:D8:D1:47:55:93
	Comment          string // nb=Vlad/com=UTTiST/col=Admin/quotahourly=500000000/quotadaily=50000000000
	HostName         string // root-hp
	// disabled         string // false
	//User Defined
	ID  int    `json:"id"`
	IP  string `json:"ip"`
	Mac string `json:"mac"`

	Manual          bool
	Blocked         bool
	Disabled        bool
	ShouldBeBlocked bool
	TypeD           string
	TimeoutBlock    string
	HourlyQuota     uint64
	DailyQuota      uint64
	MonthlyQuota    uint64
	Timeout         time.Time
	//UserType
	Name     string
	Position string
	Company  string
	IDUser   string
}
