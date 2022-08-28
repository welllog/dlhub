package main

import (
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type GitHubPage struct {
	Projects  []*Project
	TotalPage int
	CurPage   int
}

type Project struct {
	Uri      string
	Name     string
	Desc     string
	Keys     []string
	Star     string
	Language string
	UpdateAt string
}

func ParseHtml(r io.Reader) (*GitHubPage, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	result := new(GitHubPage)

	nodes := doc.Find(".repo-list-item").Nodes
	result.Projects = make([]*Project, 0, len(nodes))

	for _, v := range nodes {
		var proj Project

		item := goquery.NewDocumentFromNode(v)

		atag := item.Find("a").First()
		href, _ := atag.Attr("href")
		proj.Uri = "https://github.com" + href + ".git"
		proj.Name = strings.TrimSpace(atag.Text())
		proj.Desc = strings.TrimSpace(item.Find("p.mb-1").First().Text())

		keys := item.Find("a[data-ga-click]").Nodes
		for _, v := range keys {
			for _, attr := range v.Attr {
				if attr.Key == "title" {
					proj.Keys = append(proj.Keys, strings.TrimPrefix(attr.Val, "Topic: "))
				}
			}
		}

		attaches := item.Find("div.mr-3")
		proj.Star = strings.TrimSpace(attaches.First().Find("a").First().Text())
		proj.Language = strings.TrimSpace(attaches.Eq(1).Find("span[itemprop=programmingLanguage]").
			First().Text())
		proj.UpdateAt, _ = attaches.Find("relative-time").First().Attr("datetime")

		result.Projects = append(result.Projects, &proj)
	}

	page := doc.Find(".paginate-container").First().Find(".current").First()
	total, _ := page.Attr("data-total-pages")
	result.TotalPage, _ = strconv.Atoi(total)
	result.CurPage, _ = strconv.Atoi(strings.TrimSpace(page.Text()))
	return result, nil
}
