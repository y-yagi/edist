package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/github"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/goext/arr"
	"golang.org/x/oauth2"
)

const cmd = "edist"

var cfg config

type config struct {
	AccessToken string `toml:"access_token"`
	Editor      string `toml:"editor"`
}

func init() {
	if !configure.Exist(cmd) {
		cfg.AccessToken = ""
		cfg.Editor = "vim"
		configure.Save(cmd, cfg)
	}
}

func main() {
	os.Exit(run())
}

func msg(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %+v\n", cmd, err)
		return 1
	}
	return 0
}

func run() int {
	var editConfig bool
	var showUserGists string
	var gistID string
	var client *github.Client
	var err error

	flags := flag.NewFlagSet(cmd, flag.ExitOnError)
	flags.BoolVar(&editConfig, "c", false, "Edit config.")
	flags.StringVar(&showUserGists, "l", "", "Show list a `user` Gists.")
	flags.StringVar(&gistID, "e", "", "Edit Gist that `ID` was specified.")
	flags.Parse(os.Args[1:])

	if editConfig {
		return runEditConfig()
	}

	if err = configure.Load(cmd, &cfg); err != nil {
		return msg(err)
	}

	ctx := context.Background()
	client, err = buildClient(&ctx)
	if err != nil {
		return msg(err)
	}

	if len(showUserGists) > 0 {
		return runShowList(client, &ctx, showUserGists)
	} else if len(gistID) > 0 {
		return runEditGist(client, &ctx, gistID)
	} else {
		flags.PrintDefaults()
		return 0
	}

}

func buildClient(ctx *context.Context) (*github.Client, error) {
	accessToken := cfg.AccessToken
	if token := os.Getenv("GITHUB_ACCESS_TOKEN"); len(token) > 0 {
		accessToken = token
	}

	if len(accessToken) == 0 {
		return nil, errors.New("GitHub access token is not found. Please specify access token to config file")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.AccessToken},
	)
	tc := oauth2.NewClient(*ctx, ts)
	return github.NewClient(tc), nil
}

func runEditConfig() int {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	if err := configure.Edit(cmd, editor); err != nil {
		return msg(err)
	}

	return 0
}

func runShowList(client *github.Client, ctx *context.Context, username string) int {
	gists, _, err := client.Gists.List(*ctx, username, nil)
	if err != nil {
		return msg(err)
	}

	for _, gist := range gists {
		var filenames []string
		for _, file := range gist.Files {
			filenames = append(filenames, *file.Filename)
		}
		fmt.Printf("[ID]: %s [desc]: %s [files]: %s\n", *gist.ID, *gist.Description, arr.Join(filenames, ", "))
	}

	return 0
}

func runEditGist(client *github.Client, ctx *context.Context, gistID string) int {
	gist, _, err := client.Gists.Get(*ctx, gistID)
	if err != nil {
		return msg(err)
	}

	dir, err := ioutil.TempDir("", cmd)
	if err != nil {
		return msg(err)
	}

	defer os.Remove(dir)

	files := make(map[github.GistFilename]github.GistFile)
	tmpfn := filepath.Join(dir, "tmpfile")

	for _, file := range gist.Files {
		oldContent := []byte(*file.Content)
		if err := ioutil.WriteFile(tmpfn, oldContent, 0644); err != nil {
			return msg(err)
		}

		cmd := exec.Command(cfg.Editor, tmpfn)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			return msg(err)
		}

		newContent, _ := ioutil.ReadFile(tmpfn)

		if bytes.Compare(oldContent, newContent) == 0 {
			continue
		}

		gFilename := github.GistFilename(*file.Filename)
		files[gFilename] = github.GistFile{Filename: github.String(*file.Filename), Content: github.String(string(newContent))}
	}

	if len(files) == 0 {
		return 0
	}

	input := &github.Gist{Files: files}
	if _, _, err := client.Gists.Edit(*ctx, gistID, input); err != nil {
		return msg(err)
	}
	return 0
}
