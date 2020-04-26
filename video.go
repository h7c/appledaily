package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type (
	VideoListResponse struct {
		PlaylistItems []Article `json:"playlistItems"`
	}
)

func handleVideo(w http.ResponseWriter, r *http.Request) {
	fields, err := getVideo(r.URL.Path)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	t := template.Must(template.New("layout").Parse(Layout))
	t = template.Must(t.New("list").Parse(ListTpl))
	err = t.ExecuteTemplate(w, "layout", fields)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
}

func getVideo(path string) (fields *ListFields, err error) {
	res, err := http.Get(PREFIX + path)
	if err != nil {
		return
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

	var config struct {
		ID string `json:"_id"`
	}
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		script := s.Text()
		if !strings.Contains(script, "Fusion.globalContent") {
			return
		}
		match := regexp.MustCompile("Fusion.globalContent\\s*=\\s*(.+?);").FindStringSubmatch(script)
		if len(match) < 2 {
			return
		}
		err = json.Unmarshal([]byte(match[1]), &config)
		if err != nil {
			return
		}
	})
	if config.ID == "" {
		return
	}

	deployment := getDeployment(doc)

	q := fmt.Sprintf(`{"size":81,"tag":"hk_%s_videowall"}`, config.ID[7:])
	var listResp VideoListResponse
	err = apiGet("/playlists", q, deployment, &listResp)
	if err != nil {
		return
	}

	items := []ListItem{}
	for _, item := range listResp.PlaylistItems {
		image, video := item.getImageAndVideo()
		items = append(items, ListItem{
			Title: template.HTML(item.Headlines.Basic),
			Video: video,
			Image: image,
			Time:  item.DisplayDate,
		})
	}
	fields = &ListFields{
		NavItems:  getNavItems(doc),
		ListItems: items,
	}

	return
}
