package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type standing struct {
	Date time.Time
	Home string
	Away string
}

type competition struct {
	ID   string
	Name string
	URL  string
}

func MackolikFener() ([]standing, error) {
	res, err := http.Get("https://www.mackolik.com/takim/fenerbah%C3%A7e/ma%C3%A7lar/8lroq0cbhdxj8124qtxwrhvmm")
	if err != nil {
		return nil, fmt.Errorf("mackolik get: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("mackolik get: status=%v, body=%s", res.StatusCode, body)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("goquery: %w", err)
	}

	var competitionDatas []competition
	allLeagues := make(map[string]string)
	doc.Find(".page-team__dropdown--competition").Find(".component-dropdown__custom-item").Each(func(i int, s *goquery.Selection) {
		id, ok := s.Attr("data-value")
		if !ok {
			return
		}
		allLeagues[id] = s.Text()
	})

	settingsstr := html.UnescapeString(doc.Find(".page-team").AttrOr("data-settings", ""))
	var competetionSetting map[string]interface{}
	err = json.Unmarshal([]byte(settingsstr), &competetionSetting)
	if err != nil {
		return nil, fmt.Errorf("unmarshall competetion setting: %w", err)
	}

	competetiondata := competetionSetting["competitionPickerData"].(map[string]interface{})
	for id, url := range competetiondata {
		uu := url.(string)
		if !strings.Contains(uu, "/2021-") {
			continue
		}
		competitionDatas = append(
			competitionDatas,
			competition{
				ID:   id,
				Name: allLeagues[id],
				URL:  url.(string),
			},
		)
	}

	var allStandings []standing
	for _, comp := range competitionDatas {
		standings, err := fetchLeaugeStandings(comp)
		if err != nil {
			return nil, fmt.Errorf("fetch league standings: %w", err)
		}
		allStandings = append(allStandings, standings...)
	}

	sort.Slice(allStandings, func(i int, j int) bool {
		return allStandings[i].Date.Before(allStandings[j].Date)
	})

	return allStandings, nil

}
func fetchLeaugeStandings(c competition) ([]standing, error) {
	resp, err := http.Get(c.URL)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", c.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("%v: status=%v, body=%s", c.Name, resp.StatusCode, body)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%v: goquery: %w", c.Name, err)
	}

	var standings []standing
	doc.Find(".p0c-team-matches__teams-container").Each(func(i int, s *goquery.Selection) {
		timestamp, ok := s.Find(".p0c-team-matches__button--start-time").Attr("data-start-timestamp")
		if !ok {
			return
		}

		ts, err := strconv.Atoi(timestamp)
		if err != nil {
			log.Printf("%v: unable to convert %q to integer", c.Name, timestamp)
			return
		}

		startTime := time.Unix(int64(ts), 0).UTC()
		var homeTeam, awayTeam string
		s.Find(".p0c-team-matches__teams").Each(func(i int, s *goquery.Selection) {
			s.Find(".p0c-team-matches__team-full-name").Each(func(i int, s *goquery.Selection) {
				p := s.ParentsFiltered(".p0c-team-matches__team").First()
				if p.HasClass("p0c-team-matches__team--home") {
					homeTeam = s.Text()
					return
				}
				awayTeam = s.Text()
			})
		})

		standings = append(
			standings,
			standing{
				Date: startTime,
				Home: strings.TrimSpace(homeTeam),
				Away: strings.TrimSpace(awayTeam),
			})
	})

	return standings, nil
}
