package main

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type (
	NavItemGroup struct {
		Title string
		Items []NavItem
	}

	NavItem struct {
		Title string
		Link  string
	}
)

func getNavItems(doc *goquery.Document) (navItems []NavItemGroup) {
	var nav struct {
		Service struct {
			Navigation struct {
				Data struct {
					Children []struct {
						Name     string `json:"name"`
						Children []struct {
							Name string `json:"name"`
							Url  string `json:"_id"`
						} `json:"children"`
					} `json:"children"`
				} `json: "data"`
			} `json:"hk-navigation"`
		} `json:"site-service-v3"`
	}
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		script := s.Text()
		if !strings.Contains(script, "Fusion.contentCache") {
			return
		}
		match := regexp.MustCompile("Fusion.contentCache\\s*=\\s*(.+?);").FindStringSubmatch(script)
		if len(match) != 2 {
			return
		}
		str := strings.Replace(match[1], `{\"hierarchy\":\"hk-navigation\"}`, "hk-navigation", 1)
		json.Unmarshal([]byte(str), &nav)
	})
	for _, section := range nav.Service.Navigation.Data.Children {
		items := []NavItem{}
		for _, item := range section.Children {
			if item.Url == "/daily/catalog" {
				continue
			}
			items = append(items, NavItem{
				Title: item.Name,
				Link:  item.Url,
			})
		}
		if len(items) == 0 {
			continue
		}
		if len(navItems) == 0 {
			items = append([]NavItem{
				NavItem{
					Title: "蘋果新聞",
					Link:  "/",
				},
			}, items...)
		}
		navItems = append(navItems, NavItemGroup{
			Title: section.Name,
			Items: items,
		})
	}
	return
}

const Layout = `{{ define "layout" }}
<html>
<head>
<meta charset="utf-8">
<meta name="referrer" content="never">
<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
<meta name="referrer" content="never">
<meta name="referrer" content="no-referrer">
<title>hk.appledaily.com</title>
<link rel="icon" href="data:;base64,iVBORw0KGgo=">
<style>
body {
  padding: 0;
  margin: 0;
  font-family: "San Francisco",Roboto,"Segoe UI","Helvetica Neue","Lucida Grande",sans-serif;
}

a {
  color: #1565c0;
  text-decoration: none;
}

.nav {
  display: flex;
  flex-wrap: wrap;
  padding: 10px 10px 0;
}

.nav-item {
  display: block;
  padding: 1px 3px;
  margin: 0 3px;
  font-size: 14px;
  background: #2196F3;
  color: #fff;
}
{{ template "style" . }}
</style>
</head>
<body>
  {{ range .NavItems }}
    <div class="nav">
    <strong>{{ .Title }}</strong>
    {{ range .Items }}
      <a class="nav-item" href="{{ .Link }}">{{ .Title }}</a>
    {{ end }}
    </div>
  {{ end }}
{{ template "body" . }}
</body>
</html>
{{ end }}
`
