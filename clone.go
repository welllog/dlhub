package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"log/slog"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/welllog/golib/setz"
	"github.com/welllog/golib/slicez"
)

func Clone(ctx context.Context, proj *Project) error {
	path := FileBaseDir

	cloneConfig := &git.CloneOptions{
		URL:      proj.Uri,
		Progress: Screen,
	}
	if proxy := os.Getenv("https_proxy"); proxy != "" {
		slog.Info("use proxy", slog.String("proxy", proxy))
		cloneConfig.ProxyOptions = transport.ProxyOptions{
			URL: proxy,
		}
	}

	slog.Info("start clone " + proj.Name)
	_, err := git.PlainCloneContext(ctx, filepath.Join(path, proj.Name), false, cloneConfig)
	if err != nil {
		slog.Error("clone failed", slog.String("project", proj.Name), slog.String("err", err.Error()))
		return err
	}
	slog.Info("complete clone " + proj.Name)

	file, err := os.OpenFile(path+"/projects.md", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	defer file.Close()

	buf.Reset()
	buf.WriteString("> ### ")
	buf.WriteString(proj.Name)
	buf.WriteString("\n")
	buf.WriteString("> ###### ")
	buf.WriteString(proj.Desc)
	buf.WriteString("\n")
	buf.WriteString("> ")
	for _, v := range proj.Keys {
		buf.WriteString("``")
		buf.WriteString(v)
		buf.WriteString("`` ")
	}
	buf.WriteString("\n")
	buf.WriteString("> \n")
	buf.WriteString("> - star: ")
	buf.WriteString(proj.Star)
	buf.WriteString("\n")
	buf.WriteString("> - language: ")
	buf.WriteString(proj.Language)
	buf.WriteString("\n")
	buf.WriteString("> - updated: ")
	buf.WriteString(proj.UpdateAt)
	buf.WriteString("\n\n")

	file.WriteString(buf.String())

	return nil
}

var Screen = &Shower{}

type Shower struct{}

func (s *Shower) Write(p []byte) (n int, err error) {
	return fmt.Printf("\r %s", strings.TrimSpace(string(p)))
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

func filterGithubPage(g *GitHubPage, existsRepos setz.Set[string]) {
	g.Projects = slicez.FilterInPlace(g.Projects, func(p *Project) bool {
		return !existsRepos.Has(p.Name)
	})
}

func loadExistsRepo(ctx context.Context, dir string) (setz.Set[string], error) {
	set := make(setz.Set[string], 10000)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == dir {
			return nil
		}

		if isCancel(ctx) {
			return ctx.Err()
		}

		suffix := strings.TrimPrefix(path, dir)
		suffix = strings.TrimPrefix(suffix, "/")
		if strings.Count(suffix, "/") == 1 {
			if d.IsDir() {
				set.Add(suffix)
				slog.Debug("load exists repo", slog.String("name", suffix))
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		slog.Error("load exists repo failed", slog.String("err", err.Error()))
		return nil, err
	}

	return set, nil
}
