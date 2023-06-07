package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const (
	GITHUB_BASE_URL = "https://github.com/search?o=desc&q=%s&s=stars&type=Repositories"
	PAGE_SIZE       = 10
	GO              = "Go"
	JS              = "JavaScript"
	HTML            = "HTML"
	JAVA            = "Java"
	PYTHON          = "Python"
	CSS             = "CSS"
	PHP             = "PHP"
	TS              = "TypeScript"
	CSHARP          = "C#"
	Ruby            = "Ruby"
	CPP             = "C++"
	C               = "C"
)

var (
	lang  = flag.String("lang", "", "query language")
	query = flag.String("query", "go", "query words")
	limit = flag.Int("limit", 50, "clone projects limit")
	skip  = flag.Int("skip", 0, "clone skip")
	dir   = flag.String("dir", "", "dir")
	do    = flag.String("do", "pull", "do something like clone or pull")
)

func main() {
	flag.Parse()

	fmt.Println("language: ", *lang)
	fmt.Println("query: ", *query)
	fmt.Println("limit: ", *limit)
	fmt.Println("skip: ", *skip)
	fmt.Println("dir: ", *dir)
	fmt.Println("do: ", *do)

	if *dir == "" || *dir == "." {
		FileBaseDir = ""
	} else {
		FileBaseDir = strings.TrimRight(*dir, "/") + "/"
	}

	ctx, cancel := context.WithCancel(context.Background())
	var w sync.WaitGroup

	switch *do {
	case "clone":
		doClone(ctx, &w)
	case "pull":
		doPull(ctx, &w)
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		cancel()
	}()

	w.Wait()
	cancel()
}
