package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type (
	ListItem struct {
		Title template.HTML
		Image string
		Video string
		Link  string
		Time  time.Time
	}

	ListFields struct {
		NavItems    []NavItemGroup
		SearchQuery string
		ListItems   []ListItem
	}

	ListResponse struct {
		ContentElements []Article `json:"content_elements"`
	}
)

func handleList(pageType string, w http.ResponseWriter, r *http.Request) {
	var fields *ListFields
	var err error

	if pageType == "index" {
		fields, err = getIndex(r.URL.Path)
	} else if pageType == "realtime-top" {
		fields, err = getRealtimeTop(r.URL.Path)
	} else {
		fields, err = getList(r.URL.Path)
	}

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

func getIndex(path string) (fields *ListFields, err error) {
	res, err := http.Get(PREFIX + path)
	if err != nil {
		return
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

	contentType, listPath := "article", "/content-by-motherlode-id"
	if path == "/video/top" {
		contentType, listPath = "video", "/video-by-motherlode-id"
	}

	deployment := getDeployment(doc)

	t := math.Floor(float64(time.Now().Unix()) / 12e2)
	q := fmt.Sprintf(`{"contentType":"%s","mostHitHex":%0.f,"size":45,"website":"HK"}`, contentType, t)
	var mostHitResp struct {
		Response []struct {
			ID string `json:"_id"`
		} `json:"response"`
	}
	apiGet("/most-hit", q, deployment, &mostHitResp)

	ids := []string{}
	for _, r := range mostHitResp.Response {
		ids = append(ids, r.ID)
	}
	q = fmt.Sprintf(`{"id":"%s","size":45,"website_url":"hk-appledaily"}`, strings.Join(ids, "%20"))
	var listResp ListResponse
	apiGet(listPath, q, deployment, &listResp)

	items := []ListItem{}
	for _, item := range listResp.ContentElements {
		image, video := item.getImageAndVideo()
		items = append(items, ListItem{
			Title: template.HTML(item.Headlines.Basic),
			Video: video,
			Image: image,
			Link:  item.CanonicalUrl,
			Time:  item.DisplayDate,
		})
	}
	fields = &ListFields{
		NavItems:  getNavItems(doc),
		ListItems: items,
	}
	return
}

func getRealtimeTop(path string) (fields *ListFields, err error) {
	res, err := http.Get(PREFIX + path)
	if err != nil {
		return
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}
	scriptSrc := doc.Find("script#fusion-template-script").AttrOr("src", "")
	if scriptSrc == "" {
		return
	}
	resp, err := http.Get(PREFIX + scriptSrc)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	match := regexp.MustCompile("Fusion.tree\\s*=\\s*(.+?);").FindSubmatch(b)
	if len(match) < 2 {
		return
	}
	var tree struct {
		Children []struct {
			Children []struct {
				Props struct {
					CustomFields struct {
						QueryString string `json:"queryString"`
					} `json:"customFields"`
				} `json:"props"`
			} `json:"children"`
		} `json:"children"`
	}
	err = json.Unmarshal(match[1], &tree)
	if err != nil {
		return
	}
	ids := []string{}
	for _, a := range tree.Children {
		for _, b := range a.Children {
			if q := b.Props.CustomFields.QueryString; q != "" {
				ids = append(ids, q)
			}
		}
	}

	deployment := getDeployment(doc)

	items := []ListItem{}
	for _, id := range ids {
		q := fmt.Sprintf(`{"id":"%s","website":"hk-appledaily"}`, id)
		var listResp ListResponse
		apiGet("/collections", q, deployment, &listResp)
		for i, item := range listResp.ContentElements {
			if i > 1 {
				break
			}
			image, video := item.getImageAndVideo()
			items = append(items, ListItem{
				Title: template.HTML(item.Headlines.Basic),
				Video: video,
				Image: image,
				Link:  item.CanonicalUrl,
				Time:  item.DisplayDate,
			})
		}
	}
	fields = &ListFields{
		NavItems:  getNavItems(doc),
		ListItems: items,
	}
	return
}

func getList(path string) (fields *ListFields, err error) {
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

	q := fmt.Sprintf(`{"feedOffset":0,"feedQuery":"taxonomy.primary_section._id:\"%s\"+AND+type:story+AND+publish_date:[now-48h/h+TO+now]","feedSize":100,"sort":"display_date:desc"}`, config.ID)
	var listResp ListResponse
	err = apiGet("/query-feed", q, deployment, &listResp)
	if err != nil {
		return
	}

	items := []ListItem{}
	for _, item := range listResp.ContentElements {
		image, video := item.getImageAndVideo()
		items = append(items, ListItem{
			Title: template.HTML(item.Headlines.Basic),
			Video: video,
			Image: image,
			Link:  item.CanonicalUrl,
			Time:  item.DisplayDate,
		})
	}
	fields = &ListFields{
		NavItems:  getNavItems(doc),
		ListItems: items,
	}

	return
}

const ListTpl = `
{{ define "style" }}
.posts {
  display: flex;
  flex-wrap: wrap;
  max-width: 1600px;
  margin: auto;
}

.post {
  display: flex;
  flex-direction: column;
  width: 20%;
  padding: 5px;
  box-sizing: border-box;
}

@media (max-width: 1024px) {
  .post {
    width: 25%;
  }
}

@media (max-width: 450px) {
  .post {
    width: 50%;
  }
}

.media {
  height: 0;
  width: 100%;
  padding-bottom: 56.25%;
  position: relative;
}

video {
  height: 100%;
  width: 100%;
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
}

figure {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
  margin: 0;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

figure img {
  width: 100%;
}

.title {
  margin-top: 5px;
  font-size: 14px;
}
{{ end }}
{{ define "body" }}
<div class="posts">
  {{ range .ListItems }}
    {{ if .Video }}
      <div class="post">
        <div class="media">
          <video src="{{ .Video }}" preload="none" controls poster="{{ .Image }}"></video>
        </div>
        {{ if .Link }}
          <a class="title" href="{{ .Link }}">{{ .Title }}</a>
        {{ else }}
          <div class="title">{{ .Title }}</div>
        {{ end }}
      </div>
    {{ else }}
      <a class="post" href="{{ .Link }}">
        <div class="media">
          <figure><img src="{{ .Image }}"></figure>
        </div>
        <div class="title">{{ .Title }}</div>
      </a>
    {{ end }}
  {{ end }}
</div>
{{ end }}
`
