package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type (
	Article struct {
		Headlines struct {
			Basic string `json:"basic"`
		} `json:"headlines"`
		CanonicalUrl string    `json:"canonical_url"`
		DisplayDate  time.Time `json:"display_date"`
		PromoItems   struct {
			Basic struct {
				PromoImage struct {
					Url string `json:"url"`
				} `json:"promo_image"`
				Streams []struct {
					Url string `json:"url"`
				} `json:"streams"`
				Type string `json:"type"`
				Url  string `json:"url"`
			} `json:"basic"`
		} `json:"promo_items"`

		ArticleContentElements []ArticleContentElement `json:"content_elements"`

		Streams []struct {
			Url string `json:"url"`
		} `json:"streams"`
	}

	ArticleContentElement struct {
		Type string `json:"type"`

		// when type is text
		Text string `json:"content"`

		// when type is raw_html
		HTML template.HTML

		// when type is image
		Url     string `json:"url"`
		Caption string `json:"caption"`
	}

	ArticleFields struct {
		NavItems    []NavItemGroup
		SearchQuery string
		Title       string
		Image       string
		Video       string
		Elements    []ArticleContentElement
	}
)

func (article Article) getImageAndVideo() (string, string) {
	image := article.PromoItems.Basic.Url
	if image == "" {
		image = article.PromoItems.Basic.PromoImage.Url
	}
	video := ""
	if len(article.PromoItems.Basic.Streams) > 0 {
		video = article.PromoItems.Basic.Streams[0].Url
	}
	if len(article.Streams) > 0 {
		video = article.Streams[0].Url
	}
	return image, video
}

func handleArticle(w http.ResponseWriter, r *http.Request) {
	fields, err := getArticle(r.URL.Path)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	t := template.Must(template.New("layout").Parse(Layout))
	t = template.Must(t.New("article").Parse(ArticleTpl))
	err = t.ExecuteTemplate(w, "layout", fields)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
}

func getArticle(path string) (fields *ArticleFields, err error) {
	res, err := http.Get(PREFIX + path)
	if err != nil {
		return
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
	}

	var resp Article
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		script := s.Text()
		if !strings.Contains(script, "Fusion.globalContent") {
			return
		}
		match := regexp.MustCompile("Fusion.globalContent\\s*=\\s*(.+?);(Fusion|$)").FindStringSubmatch(script)
		if len(match) < 2 {
			return
		}
		json.Unmarshal([]byte(match[1]), &resp)
	})

	for i := range resp.ArticleContentElements {
		if resp.ArticleContentElements[i].Type == "raw_html" {
			resp.ArticleContentElements[i].HTML = template.HTML(resp.ArticleContentElements[i].Text)
		}
	}

	image, video := resp.getImageAndVideo()
	fields = &ArticleFields{
		NavItems: getNavItems(doc),
		Title:    resp.Headlines.Basic,
		Image:    image,
		Video:    video,
		Elements: resp.ArticleContentElements,
	}

	return
}

const ArticleTpl = `
{{ define "style" }}
.post {
  padding: 10px;
  max-width: 600px;
}

.line {
  margin-bottom: 20px;
}

video, img {
  width: 100%;
  height: auto;
}
{{ end }}
{{ define "body" }}
<div class="post">
  <h1>{{ .Title }}</h1>
  {{ if .Video }}
    <div class="line">
      <video src="{{ .Video }}" preload="none" controls poster="{{ .Image }}"></video>
    </div>
  {{ else if .Image }}
    <div class="line"><img src="{{ .Image }}"></div>
  {{ end }}
  {{ range .Elements }}
    {{ if eq .Type "text" }}
      <div class="line">{{ .Text }}</div>
    {{ else if eq .Type "raw_html"}}
      <div class="line">{{ .HTML }}</div>
    {{ else if eq .Type "image"}}
      <div class="line"><img src="{{ .Url }}"><br>{{ .Caption }}</div>
    {{ end }}
  {{ end }}
</div>
{{ end }}
`
