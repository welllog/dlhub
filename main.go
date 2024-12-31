package main

import (
	"bytes"
	"context"
	"flag"
	"log/slog"
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
	dir   = flag.String("dir", "", "dir")
	do    = flag.String("do", "pull", "do something like clone or pull")
)

var (
	FileBaseDir string
	buf         bytes.Buffer
)

func main() {
	flag.Parse()

	slog.Info("",
		slog.String("lang", *lang),
		slog.String("query", *query),
		slog.Int("limit", *limit),
		slog.String("dir", *dir),
		slog.String("do", *do),
	)
	slog.SetLogLoggerLevel(slog.LevelDebug)

	FileBaseDir = strings.TrimPrefix(*dir, "./")
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
