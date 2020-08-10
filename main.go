package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const (
	GITHUB_BASE_URL = "https://github.com/search?o=desc&q=%s&s=stars&type=Repositories"
	PAGE_SIZE = 10
	GO = "Go"
	JS = "JavaScript"
	HTML = "HTML"
	JAVA = "Java"
	PYTHON = "Python"
	CSS = "CSS"
	PHP = "PHP"
	TS = "TypeScript"
	CSHARP = "C#"
	Ruby = "Ruby"
	CPP = "C++"
	C = "C"
)

func main() {
	lang := flag.String("lang", "", "query language")
	query := flag.String("query", "go", "query words")
	limit := flag.Int("limit", 50, "clone projects limit")
	skip := flag.Int("skip", 0, "clone skip")
	dir := flag.String("dir", "", "dir")

	flag.Parse()

	fmt.Println("language: ", *lang)
	fmt.Println("query: ", *query)
	fmt.Println("limit: ", *limit)
	fmt.Println("skip: ", *skip)
	fmt.Println("dir: ", *dir)

	if *dir == "" || *dir == "." {
		FileBaseDir = ""
	} else {
		FileBaseDir = strings.TrimRight(*dir, "/") + "/"
	}

	page := 1
	if *skip > 0 {
		page += *skip / PAGE_SIZE
		*skip = *skip % PAGE_SIZE
	}

	pageContent, err := searchInGithub(*lang, *query, page)
	if err != nil {
		log.Printf("search github err: " + err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	end := make(chan struct{}, 1)

	var w sync.WaitGroup
	w.Add(1)
	go func() {
		defer w.Done()
		defer fmt.Println("^ -- ^ ######### complete all")
		defer func() {
			end <- struct{}{}
		}()

		var incr int
		var total int
		for {
			for _, v := range pageContent.Projects {
				total++
				if total <= *skip {
					continue
				}

				select {
				case <- ctx.Done():
					return
				default:
				}
				_ = Clone(ctx, v)
				incr++
				if incr >= *limit {
					return
				}
			}
			if pageContent.TotalPage <= pageContent.CurPage {
				return
			}
			pageContent,err = searchInGithub(*lang, *query, pageContent.CurPage + 1)
			if err != nil {
				log.Printf("search github err: " + err.Error())
				return
			}

		}

	}()


	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <- quit:
	case <- end:
	}
	cancel()
	w.Wait()
}

func searchInGithub(language, query string, page int) (*GitHubPage, error) {
	qry := url.QueryEscape(query)
	uri := fmt.Sprintf(GITHUB_BASE_URL, qry)
	if page > 1 {
		uri += "&p=" + strconv.Itoa(page)
	}
	if language != "" {
		uri += "&l=" + url.QueryEscape(language)
	}

	rsp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, errors.New("http code: " + strconv.Itoa(rsp.StatusCode))
	}

	return ParseHtml(rsp.Body)
}