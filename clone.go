package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"os"
	"strings"
)

var FileBaseDir string

func Clone(ctx context.Context, proj *Project) error {
	path := FileBaseDir + proj.Language
	_, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		err = os.Mkdir(path, 0755)
		if err != nil {
			fmt.Println("mkdir err: ", err.Error())
			return err
		}
	}

	Printf("start clone " + proj.Name)
	names := strings.Split(proj.Name, "/")
	lname := len(names)
	_, err = git.PlainCloneContext(ctx, path + "/" + names[lname - 1], false, &git.CloneOptions{
		URL: proj.Uri,
		Progress: Screen,
	})
	//cmd := exec.Command("/usr/bin/git clone", proj.Uri, path + "/" + proj.Name)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stdout
	//err = cmd.Run()
	if err != nil {
		fmt.Println("clone " + proj.Name + " err: " + err.Error())
		return err
	}
	Printf("clone " + proj.Name + " complete")

	file, err := os.OpenFile(path + "/projects.md", os.O_RDWR | os.O_APPEND | os.O_CREATE, 0755)
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

type Shower struct {}

func (s *Shower) Write(p []byte) (n int, err error) {
	return Printf("%s", bytes.TrimSpace(p))
}

func Printf(format string, a ...interface{}) (n int, err error) {
	n, err = fmt.Fprintf(os.Stdout, "\r" + format, a...)
	return
}
