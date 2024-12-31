package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/welllog/golib/setz"
)

func doClone(ctx context.Context, w *sync.WaitGroup) {
	_, err := os.Stat(FileBaseDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("make root dir failed", slog.String("project", FileBaseDir), slog.String("err", err.Error()))
			return
		}

		err = os.MkdirAll(FileBaseDir, 0755)
		if err != nil {
			slog.Error("make root dir failed", slog.String("project", FileBaseDir), slog.String("err", err.Error()))
			return
		}
	}

	repos, err := loadExistsRepo(ctx, FileBaseDir)
	if err != nil {
		return
	}

	pageContent, err := searchInGithub(*lang, *query, 1)
	if err != nil {
		slog.Error("search github", slog.String("err", err.Error()))
		return
	}
	filterGithubPage(pageContent, repos)

	w.Add(1)
	go func() {
		var (
			incr, page int
		)
		defer w.Done()
		defer func() {
			slog.Info(fmt.Sprintf("^ -- ^ ######### complete all; clone count: %d, page: %d \n", incr, page))
		}()

		now := time.Now()
		for {
			page = pageContent.CurPage
			for _, v := range pageContent.Projects {
				select {
				case <-ctx.Done():
					return
				default:
				}

				if err := Clone(ctx, v); err == nil {
					incr++
					if incr >= *limit {
						return
					}
				}
			}
			if pageContent.TotalPage <= pageContent.CurPage {
				return
			}
			nextPage := pageContent.CurPage + 1
			if time.Since(now).Seconds() < 1 {
				time.Sleep(time.Second)
			}
			for retry := 0; retry <= 3; retry++ {
				pageContent, err = searchInGithub(*lang, *query, nextPage)
				if err == nil {
					filterGithubPage(pageContent, repos)
					break
				} else if retry == 3 {
					slog.Error("retry search github failed", slog.String("err", err.Error()))
					return
				} else {
					slog.Error("search github failed, start retry",
						slog.String("err", err.Error()), slog.Int("retry", retry),
					)
					time.Sleep(2 * time.Second)
				}
			}
			now = time.Now()
		}
	}()

}

func doPull(ctx context.Context, w *sync.WaitGroup) {
	w.Add(1)
	go func() {
		defer w.Done()

		pulled, err := loadPullRepo()
		if err != nil {
			slog.Error("load pulled repo", slog.String("err", err.Error()))
			return
		}

		pullFd, err := os.OpenFile("pull.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			slog.Error("open pull.txt", slog.String("err", err.Error()))
			return
		}
		defer pullFd.Close()

		err = filepath.WalkDir(FileBaseDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if isCancel(ctx) {
				return ctx.Err()
			}

			repoPath := strings.TrimPrefix(path, FileBaseDir)
			repoPath = strings.TrimPrefix(repoPath, "/")
			if strings.Count(repoPath, "/") != 1 {
				return nil
			}

			if !d.IsDir() {
				return nil
			}

			if pulled.Has(repoPath) {
				return filepath.SkipDir
			}

			repo, err := git.PlainOpen(path)
			if err != nil {
				slog.Error("open repo", slog.String("repo", path), slog.String("err", err.Error()))
				return filepath.SkipDir
			}

			wrt, err := repo.Worktree()
			if err != nil {
				slog.Error("get worktree", slog.String("repo", repoPath), slog.String("err", err.Error()))
				return err
			}

			pullConfig := &git.PullOptions{Force: true, Progress: Screen}
			if proxy := os.Getenv("https_proxy"); proxy != "" {
				slog.Info("use proxy", slog.String("proxy", proxy))
				pullConfig.ProxyOptions = transport.ProxyOptions{
					URL: proxy,
				}
			}

			slog.Info(fmt.Sprintf("v -- v ######### start pull %s \n", repoPath))
			err = wrt.PullContext(ctx, pullConfig)
			if err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					pullFd.WriteString(repoPath + "\n")
					slog.Info(fmt.Sprintf("v -- v ######### %s already up to date \n", repoPath))
					return filepath.SkipDir
				} else {
					slog.Error("pull repo", slog.String("repo", repoPath), slog.String("err", err.Error()))
					return err
				}
			}

			pullFd.WriteString(repoPath + "\n")
			slog.Info(fmt.Sprintf("v -- v ######### %s pull complete \n", repoPath))
			pulled.Add(repoPath)
			time.Sleep(500 * time.Millisecond)

			return filepath.SkipDir
		})

		if err != nil {
			slog.Error("walk dir", slog.String("err", err.Error()))
		}
	}()
}

func loadPullRepo() (setz.Set[string], error) {
	f, err := os.OpenFile("pull.txt", os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	set := make(setz.Set[string], 10000)

	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				slog.Error("read pull.txt", slog.String("err", err.Error()))
			}
			break
		}

		if line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		set.Add(string(line))
	}

	return set, nil
}

func isCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
