package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type QuotaType struct {
	HourlyQuota  uint64
	DailyQuota   uint64
	MonthlyQuota uint64
	Blocked      bool
}

type PersonType struct {
	Name,
	Position,
	Company string
}

type LineOfData struct {
	// timeoutStr,
	Comment,
	disable string
	addressLists []string
	// timeoutInt   int64
	DeviceType
	QuotaType
	PersonType
}

type DeviceType struct {
	Id       string
	IP       string
	TypeD    string
	Mac      string
	HostName string
	Groups   string
	timeout  time.Time
}

func logreq(f func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("path: %s", r.URL.Path)

		f(w, r)
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w,
		`<html>
			<head>
			<title>go-macfrommikrotik</title>
			</head>
			<body>
			Более подробно на https://github.com/Rid-lin/gonsquid
			</body>
			</html>
			`)
}

func (data *Transport) handlerGetMac() http.HandlerFunc {
	var (
		request  request
		Response ResponseType
	)

	return func(w http.ResponseWriter, r *http.Request) {
		request.Time = r.URL.Query().Get("time")
		request.IP = r.URL.Query().Get("ip")
		Response = data.GetInfo(&request)
		log.Debugf(" | Request:'%v','%v' response:'%v'", request.Time, request.IP, Response.Mac)
		responseJSON, err := json.Marshal(Response)
		if err != nil {
			log.Errorf("Error Marshaling mac'%v'to JSON:'%v'", Response.Mac, err)
		}
		_, err2 := w.Write(responseJSON)
		if err2 != nil {
			log.Errorf("Error send response:%v", err2)
		}
	}
}

func (data *Transport) handlerSetStatusDevices(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		errorResponse(w, "Content Type is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	result := map[string]bool{}
	var unmarshalErr *json.UnmarshalTypeError

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&result)
	if err != nil {
		if errors.As(err, &unmarshalErr) {
			errorResponse(w, "Bad Request. Wrong Type provided for field "+unmarshalErr.Field, http.StatusBadRequest)
		} else {
			errorResponse(w, "Bad Request "+err.Error(), http.StatusBadRequest)
		}
		return
	}
	data.syncStatusDevices(result)

	errorResponse(w, "Recived", http.StatusOK)
	log.Println(result)
}

func (data *Transport) handlerGetStatusDevices(w http.ResponseWriter, r *http.Request) {
	defaultLine := LineOfData{}

	data.RLock()
	dataToDSend := data.ipToMac

	defaultLine.HourlyQuota = data.HourlyQuota
	defaultLine.DailyQuota = data.DailyQuota
	defaultLine.MonthlyQuota = data.MonthlyQuota
	data.RUnlock()

	dataToDSend["default"] = defaultLine

	json_data, err := json.Marshal(dataToDSend)
	if err != nil {
		log.Errorf("Error witn Marshaling to JSON status of all devices:(%v)", err)
	}
	fmt.Fprint(w, string(json_data))
}

func errorResponse(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Error(err)
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Error(err)
	}
}
