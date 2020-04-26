package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func apiGet(path, query, deployment string, target interface{}) (err error) {
	req, err := http.NewRequest("GET", "https://hk.appledaily.com/pf/api/v3/content/fetch"+path, nil)
	if err != nil {
		return
	}
	q := req.URL.Query()
	q.Set("query", query)
	q.Set("d", deployment)
	q.Set("_website", "hk-appledaily")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&target)
	return
}

func getDeployment(doc *goquery.Document) string {
	var deployment string
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		script := s.Text()
		if !strings.Contains(script, "Fusion.deployment") {
			return
		}
		match := regexp.MustCompile("Fusion.deployment=\"(\\d+)\"").FindStringSubmatch(s.Text())
		if len(match) != 2 {
			return
		}
		deployment = match[1]
	})
	return deployment
}
