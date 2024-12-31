package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/welllog/golib/strz"
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

	list := doc.Find("div[data-testid=results-list]").Children().Nodes
	result.Projects = make([]*Project, 0, len(list))

	for _, v := range list {
		var proj Project

		item := goquery.NewDocumentFromNode(v)

		div1 := item.Find("div.search-title")
		href, _ := div1.Find("a").Attr("href")
		title := div1.Find("span").Text()
		proj.Name = strz.RemoveRunes(title, unicode.IsSpace)
		proj.Uri = fmt.Sprintf("https://github.com%s.git", href)

		div2 := item.Find("h3").Next()
		proj.Desc = trimMultiSpace(strings.TrimSpace(div2.Find("span.search-match").Text()))

		div3 := div2.Next()
		if div3.Is("div") {
			keyNodes := div3.Find("div a").Nodes
			for _, kn := range keyNodes {
				proj.Keys = append(proj.Keys, goquery.NewDocumentFromNode(kn).Text())
			}
		}

		lis := item.Find("ul li").Nodes
		if len(lis) >= 2 {
			base := 0
			if len(lis) == 3 {
				proj.Language = goquery.NewDocumentFromNode(lis[0]).Find("span").Text()
				base++
			}
			proj.Star = goquery.NewDocumentFromNode(lis[base]).Find("span").Text()
			updateAt, _ := goquery.NewDocumentFromNode(lis[base+1]).Find("div").Attr("title")
			upTime, _ := time.Parse("Jan 2, 2006, 3:04 PM UTC", updateAt)
			proj.UpdateAt = upTime.Format("2006-01-02 15:04:05")
		}

		result.Projects = append(result.Projects, &proj)
	}

	page := doc.Find("nav[aria-label=Pagination]")
	result.CurPage, _ = strconv.Atoi(page.Find("a[aria-current=page]").Text())
	result.CurPage, _ = strconv.Atoi(page.Find("a[rel=next]").Prev().Text())
	return result, nil
}

func trimMultiSpace(s string) string {
	buf.Reset()
	buf.Grow(len(s))

	begin := 0
	var spaceFind bool
	for i, v := range s {
		if unicode.IsSpace(v) {
			if !spaceFind {
				spaceFind = true
				buf.WriteString(s[begin:i])
				buf.WriteByte(' ')
			}
			begin = i + 1
			continue
		}

		spaceFind = false
	}
	buf.WriteString(s[begin:])
	return buf.String()
}
