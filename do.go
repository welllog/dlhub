package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func doClone(ctx context.Context, w *sync.WaitGroup) {
	repos, err := loadExistsRepo(ctx)
	if err != nil {
		return
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
	filterGithubPage(pageContent, repos)

	w.Add(1)
	go func() {
		var (
			total, incr, page int
		)
		defer w.Done()
		defer func() {
			fmt.Printf("^ -- ^ ######### complete all; clone count: %d, page: %d \n", incr, page)
		}()

		now := time.Now()
		for {
			page = pageContent.CurPage
			for _, v := range pageContent.Projects {
				total++
				if total <= *skip {
					continue
				}

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
					log.Printf("search github err: " + err.Error())
					return
				} else {
					log.Printf("search github err: " + err.Error())
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
			log.Printf("[ERROR] load pulled repo err: %s", err.Error())
			return
		}

		m := make(map[string]struct{}, 100000)

		pullFd, err := os.OpenFile("pull.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		if err != nil {
			log.Printf("[ERROR] open pull.txt err: %s", err.Error())
			return
		}
		defer pullFd.Close()

		err = filepath.WalkDir(FileBaseDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if strings.Contains(err.Error(), "no such file or directory") {
					return nil
				}
				return err
			}

			if isCancel(ctx) {
				return ctx.Err()
			}

			if _, ok := pulled[filepath.Dir(path)]; ok {
				return filepath.SkipDir
			}

			if d.Name() == ".git" {
				repoPath := filepath.Dir(path)
				if _, ok := m[repoPath]; ok {
					log.Printf("[ERROR] repeat repo path: %s", repoPath)
					return nil
				}

				if _, ok := pulled[repoPath]; ok {
					return filepath.SkipDir
				}

				m[repoPath] = struct{}{}

				repo, err := git.PlainOpen(repoPath)
				if err != nil {
					log.Printf("[ERROR] open repo %s err: %s", repoPath, err.Error())
					return err
				}

				wrt, err := repo.Worktree()
				if err != nil {
					log.Printf("[ERROR] get worktree %s err: %s", repoPath, err.Error())
					return err
				}

				log.Printf("[start] %s pull", repoPath)
				err = wrt.PullContext(ctx, &git.PullOptions{Force: true, Progress: Screen})
				if err != nil {
					if errors.Is(err, git.NoErrAlreadyUpToDate) {
						pullFd.WriteString(repoPath + "\n")
						log.Printf("[WARN] %s already up to date", repoPath)
						return filepath.SkipDir
					} else {
						log.Printf("[ERROR] pull %s err: %s", repoPath, err.Error())
						return err
					}
				}

				pullFd.WriteString(repoPath + "\n")
				log.Printf("[success] %s pull complete", repoPath)
				time.Sleep(500 * time.Millisecond)

				return filepath.SkipDir
			}
			return nil
		})

		if err != nil {
			log.Printf("[ERROR] walk dir err: %s", err.Error())
		}
	}()
}

func loadPullRepo() (map[string]struct{}, error) {
	f, err := os.OpenFile("pull.txt", os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	m := make(map[string]struct{}, 10000)

	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("read pull.txt err: %s", err.Error())
			}
			break
		}

		if line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		m[string(line)] = struct{}{}
	}

	return m, nil
}

func isCancel(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
