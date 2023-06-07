package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var FileBaseDir string

func Clone(ctx context.Context, proj *Project) error {
	path := FileBaseDir + proj.Language
	_, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		err = os.MkdirAll(path, os.ModeDir)
		if err != nil {
			fmt.Println("mkdir err: ", err.Error())
			return err
		}
	}

	log.Printf("start clone " + proj.Name)
	names := strings.Split(proj.Name, "/")
	lname := len(names)
	_, err = git.PlainCloneContext(ctx, path+"/"+names[lname-1], false, &git.CloneOptions{
		URL:      proj.Uri,
		Progress: Screen,
	})
	//cmd := exec.Command("/usr/bin/git clone", proj.Uri, path + "/" + proj.Name)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stdout
	//err = cmd.Run()
	if err != nil {
		log.Printf("clone " + proj.Name + " err: " + err.Error())
		return err
	}
	log.Printf("complete clone " + proj.Name)

	file, err := os.OpenFile(path+"/projects.md", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return err
	}

	defer file.Close()

	var content strings.Builder
	content.WriteString("> ### ")
	content.WriteString(proj.Name)
	content.WriteString("\n")
	content.WriteString("> ###### ")
	content.WriteString(proj.Desc)
	content.WriteString("\n")
	content.WriteString("> ")
	for _, v := range proj.Keys {
		content.WriteString("``")
		content.WriteString(v)
		content.WriteString("`` ")
	}
	content.WriteString("\n")
	content.WriteString("> \n")
	content.WriteString("> - star: ")
	content.WriteString(proj.Star)
	content.WriteString("\n")
	content.WriteString("> - language: ")
	content.WriteString(proj.Language)
	content.WriteString("\n")
	content.WriteString("> - updated: ")
	content.WriteString(proj.UpdateAt)
	content.WriteString("\n\n")

	file.WriteString(content.String())

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

func filterGithubPage(g *GitHubPage, existsRepos map[string]struct{}) {
	var remain int
	for i, v := range g.Projects {
		index := strings.Index(v.Name, "/")
		if index > 0 {
			repo := v.Name[index+1:]
			if _, ok := existsRepos[repo]; !ok {
				g.Projects[i], g.Projects[remain] = g.Projects[remain], g.Projects[i]
				remain++
			}
		}
	}
	g.Projects = g.Projects[:remain]
}

func loadExistsRepo(ctx context.Context) (map[string]struct{}, error) {
	m := make(map[string]struct{}, 10000)
	err := filepath.WalkDir(FileBaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if strings.Contains(err.Error(), "no such file or directory") {
				return nil
			}
			return err
		}

		if isCancel(ctx) {
			return ctx.Err()
		}

		if d.Name() == ".git" {
			repoPath := filepath.Dir(path)

			repo, err := git.PlainOpen(repoPath)
			if err != nil {
				return err
			}

			// 获取仓库名称
			remotes, err := repo.Remotes()
			if err != nil {
				return err
			}
			name := remotes[0].Config().URLs[0]
			index := strings.LastIndex(name, "/")
			if index > 0 {
				name = name[index+1:]
				name = strings.TrimSuffix(name, ".git")
				m[name] = struct{}{}
			}

			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		log.Printf("filepath.WalkDir() returned %v\n", err)
		return nil, err
	}
	return m, nil
}
