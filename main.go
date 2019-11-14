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
	"runtime"

	"github.com/google/go-github/github"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/goext/arr"
	"golang.org/x/oauth2"
)

const cmd = "edist"

var (
	cfg config

	// Command line flags.
	showVersion bool
	editConfig  bool
	username    string
	gistID      string
	newFilename string

	version = "devel"
)

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

	flag.BoolVar(&showVersion, "v", false, "print version number")
	flag.BoolVar(&editConfig, "c", false, "cdit config.")
	flag.StringVar(&username, "l", "", "show list a `user` Gists")
	flag.StringVar(&newFilename, "n", "", "create a new Gist by `filename`")
	flag.StringVar(&gistID, "e", "", "edit Gist that `ID` was specified")
	flag.Usage = usage
}

func main() {
	os.Exit(run())
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", cmd)
	fmt.Fprintln(os.Stderr, "OPTIONS:")
	flag.PrintDefaults()
}

func msg(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %+v\n", cmd, err)
		return 1
	}
	return 0
}

func run() int {
	var client *github.Client
	var err error

	flag.Parse()

	if showVersion {
		fmt.Printf("%s %s (runtime: %s)\n", cmd, version, runtime.Version())
		return 0
	}

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

	if len(username) > 0 {
		return runShowList(client, &ctx)
	} else if len(gistID) > 0 {
		return runEditGist(client, &ctx)
	} else if len(newFilename) > 0 {
		return runCreateGist(client, &ctx)
	} else {
		flag.Usage()
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

func runShowList(client *github.Client, ctx *context.Context) int {
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

func runEditGist(client *github.Client, ctx *context.Context) int {
	gist, _, err := client.Gists.Get(*ctx, gistID)
	if err != nil {
		return msg(err)
	}

	dir, err := ioutil.TempDir("", cmd)
	if err != nil {
		return msg(err)
	}

	defer os.RemoveAll(dir)

	files := make(map[github.GistFilename]github.GistFile)

	for _, file := range gist.Files {
		ext := filepath.Ext(*file.Filename)
		tmpfn := filepath.Join(dir, "tmpfile"+ext)

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

		if bytes.Equal(oldContent, newContent) {
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

func runCreateGist(client *github.Client, ctx *context.Context) int {
	dir, err := ioutil.TempDir("", cmd)
	if err != nil {
		return msg(err)
	}

	defer os.RemoveAll(dir)

	files := make(map[github.GistFilename]github.GistFile)

	tmpfn := filepath.Join(dir, newFilename)

	cmd := exec.Command(cfg.Editor, tmpfn)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return msg(err)
	}

	content, _ := ioutil.ReadFile(tmpfn)

	gFilename := github.GistFilename(newFilename)
	files[gFilename] = github.GistFile{Filename: github.String(newFilename), Content: github.String(string(content))}

	input := &github.Gist{Files: files}
	if _, _, err := client.Gists.Create(*ctx, input); err != nil {
		return msg(err)
	}
	return 0
}
