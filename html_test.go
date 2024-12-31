package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/welllog/golib/strz"
)

func TestGetHtml(t *testing.T) {
	f, err := os.Open("search.html")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	seg := doc.Find("div[data-testid=results-list]")
	for _, node := range seg.Children().Nodes {
		d := goquery.NewDocumentFromNode(node)

		div1 := d.Find("div.search-title")
		href, _ := div1.Find("a").Attr("href")
		title := div1.Find("span").Text()
		title = strz.RemoveRunes(title, unicode.IsSpace)

		div2 := d.Find("h3").Next()
		desc := trimMultiSpace(strings.TrimSpace(div2.Find("span.search-match").Text()))
		div3 := div2.Next()
		if div3.Is("div") {
			keyNodes := div3.Find("div a").Nodes
			for _, kn := range keyNodes {
				fmt.Println(goquery.NewDocumentFromNode(kn).Text())
			}
		}

		var language, star, updateAt string
		lis := d.Find("ul li").Nodes
		if len(lis) >= 2 {
			base := 0
			if len(lis) == 3 {
				language = goquery.NewDocumentFromNode(lis[0]).Find("span").Text()
				base++
			}
			star = goquery.NewDocumentFromNode(lis[base]).Find("span").Text()
			updateAt, _ = goquery.NewDocumentFromNode(lis[base+1]).Find("div").Attr("title")
		}

		up, _ := time.Parse("Jan 2, 2006, 3:04 PM UTC", updateAt)
		fmt.Printf("href: %s, title: %s, desc: %s, language: %s, star: %s, update: %s@%s\n",
			href, title, desc, language, star, updateAt, up.Format("2006-01-02 15:04:05"))
	}

	page := doc.Find("nav[aria-label=Pagination]")
	fmt.Println(page.Find("a[aria-current=page]").Text())
	fmt.Println(page.Find("a[rel=next]").Prev().Text())
}
