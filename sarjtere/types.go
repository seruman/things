package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var Istanbul = must(time.LoadLocation("Europe/Istanbul"))

// YesNo is EPDK's "EVET"/"HAYIR" boolean.
type YesNo string

const (
	Yes YesNo = "EVET"
	No  YesNo = "HAYIR"
)

func (y YesNo) Bool() bool { return y == Yes }

// SocketStatus is a socket's state in an availability window.
type SocketStatus string

const (
	StatusFree    SocketStatus = "FREE"
	StatusInUse   SocketStatus = "IN_USE"
	StatusFault   SocketStatus = "FAULT"
	StatusUnknown SocketStatus = "UNKNOWN"
)

// StationSummary is one entry of GET /stations.
type StationSummary struct {
	ID        int64       `json:"id"`
	Title     string      `json:"title"`
	Brand     string      `json:"brand"`
	Lat       float64     `json:"lat"`
	Lng       float64     `json:"lng"`
	Green     YesNo       `json:"green"`
	Sockets   []SocketRef `json:"sockets"`
	Available bool        `json:"available"`
}

type SocketRef struct {
	ID int64 `json:"id"`
}

// Station is the full detail from GET /stations/id/{id}/{timestamp}.
type Station struct {
	ID             int64         `json:"id"`
	Title          string        `json:"title"`
	Address        string        `json:"address"`
	Lat            float64       `json:"lat"`
	Lng            float64       `json:"lng"`
	Phone          string        `json:"phone"`
	ReportURL      string        `json:"reportUrl"`
	ReservationURL string        `json:"reservationUrl"`
	OperatorID     string        `json:"operatorid"`
	OperatorTitle  string        `json:"operatortitle"`
	LicenceActive  bool          `json:"licenceActive"`
	LicenceStatus  int           `json:"licenceStatus"`
	StationActive  bool          `json:"stationActive"`
	ServiceType    string        `json:"serviceType"`
	Green          YesNo         `json:"green"`
	Brand          string        `json:"brand"`
	CityID         string        `json:"cityid"`
	DistrictID     string        `json:"districtid"`
	Smart          YesNo         `json:"smart"`
	PaymentTypes   []PaymentType `json:"paymentTypes"`
	Sockets        []Socket      `json:"sockets"`
}

type PaymentType struct {
	Name string `json:"name"`
}

type Socket struct {
	ID           int64                `json:"id"`
	Type         string               `json:"type"`
	SubType      string               `json:"subType"`
	SocketNumber string               `json:"socketNumber"`
	Price        float64              `json:"price"`
	Power        float64              `json:"power"`
	Prices       []PriceWindow        `json:"prices"`
	Availability []AvailabilityWindow `json:"availability"`
}

type PriceWindow struct {
	Active    int       `json:"active"`
	Price     float64   `json:"price"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

type AvailabilityWindow struct {
	ID            int64        `json:"id"`
	Active        int          `json:"active"`
	Status        SocketStatus `json:"status"`
	StartTime     time.Time    `json:"startTime"`
	EndTime       time.Time    `json:"endTime"`
	ReservationID int64        `json:"reservationid"`
}

func (w AvailabilityWindow) Contains(t time.Time) bool {
	return !t.Before(w.StartTime) && t.Before(w.EndTime)
}

// StatusAt returns the availability window covering t, if any.
func (s Socket) StatusAt(t time.Time) (AvailabilityWindow, bool) {
	for _, w := range s.Availability {
		if w.Contains(t) {
			return w, true
		}
	}
	return AvailabilityWindow{Status: StatusUnknown}, false
}

type City struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	CityCode int    `json:"cityCode"`
}

// Registry is the API gateway's response envelope.
type Registry struct {
	StatusCode int               `json:"statusCode"`
	NumRows    int               `json:"numRows"`
	Data       []RegistryStation `json:"data"`
}

// RegistryStation is a station as filed with EPDK. No live data.
type RegistryStation struct {
	No                 string           `json:"sarjIstasyonuNo"`
	Name               string           `json:"sarjIstasyonuAdi"`
	Operator           string           `json:"sarjIstasyonuIsletmecisi"`
	NetworkLicenceNo   string           `json:"sarjAgiIsletmecisiLisansNo"`
	NetworkOperator    string           `json:"sarjAgiIsletmecisiUnvan"`
	Address            string           `json:"adres"`
	ServiceType        string           `json:"hizmetSekli"`
	Brand              string           `json:"marka"`
	DistributorLicence string           `json:"olumluGorusVerenDagitimSirketiLisansNo"`
	Distributor        string           `json:"olumluGorusVerenDagitimSirketiLisansUnvani"`
	Lat                float64          `json:"enlem"`
	Lng                float64          `json:"boylam"`
	Sockets            []RegistrySocket `json:"soketler"`
	Green              YesNo            `json:"yesilSarjIstasyonuMu"`
	ApprovalDocumentNo string           `json:"dagitimSirketiOlumluGorusBelgeNumarasi"`
}

type RegistrySocket struct {
	No      string  `json:"soketNo"`
	Type    string  `json:"soketTipi"`
	SubType string  `json:"soketTuru"`
	Power   float64 `json:"soketGucu,string"`
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// parseLatLng parses "lat,lng".
func parseLatLng(s string) (float64, float64, error) {
	a, b, ok := strings.Cut(s, ",")
	if !ok {
		return 0, 0, fmt.Errorf("expected lat,lng, got %q", s)
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(a), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("lat: %w", err)
	}

	lng, err := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("lng: %w", err)
	}

	return lat, lng, nil
}

// Mirror is https://alperkonan.github.io/prizy-data/stations.json.
type Mirror struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Source      string          `json:"source"`
	Stations    []MirrorStation `json:"stations"`
}

type MirrorStation struct {
	No       string         `json:"id"`
	Name     string         `json:"name"`
	Brand    string         `json:"brand"`
	Operator string         `json:"operator"`
	Address  string         `json:"address"`
	Lat      float64        `json:"lat"`
	Lng      float64        `json:"lng"`
	Green    bool           `json:"green"`
	Sockets  []MirrorSocket `json:"sockets"`
	ACCount  int            `json:"acCount"`
	ACMaxKW  *float64       `json:"acMaxKw"`
	DCCount  int            `json:"dcCount"`
	DCMaxKW  *float64       `json:"dcMaxKw"`
}

type MirrorSocket struct {
	Type     string   `json:"type"`
	Standard string   `json:"standard"`
	KW       *float64 `json:"kw"`
}
