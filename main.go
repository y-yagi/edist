package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

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
	flags.StringVar(&showUserGists, "l", "", "Show list a user Gists.")
	flags.StringVar(&gistID, "e", "", "Edit Gist that ID was specified.")
	flags.Parse(os.Args[1:])

	if editConfig {
		return runEditConfig()
	}

	if err = configure.Load(cmd, &cfg); err != nil {
		return msg(err)
	}

	ctx := context.Background()
	client = buildClient(&ctx)
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

func buildClient(ctx *context.Context) *github.Client {
	if len(cfg.AccessToken) > 0 {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: cfg.AccessToken},
		)
		tc := oauth2.NewClient(*ctx, ts)
		return github.NewClient(tc)
	}

	return github.NewClient(nil)
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
		fmt.Printf("ID: %s files: %s\n", *gist.ID, arr.Join(filenames, " ,"))
	}

	return 0
}

func runEditGist(client *github.Client, ctx *context.Context, gistID string) int {
	gist, _, err := client.Gists.Get(*ctx, gistID)
	if err != nil {
		return msg(err)
	}

	tmpfile, err := ioutil.TempFile("", cmd)
	if err != nil {
		return msg(err)
	}

	defer os.Remove(tmpfile.Name())

	files := make(map[github.GistFilename]github.GistFile)

	for _, file := range gist.Files {
		if _, err := tmpfile.Write([]byte(*file.Content)); err != nil {
			return msg(err)
		}

		cmd := exec.Command(cfg.Editor, tmpfile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err != nil {
			return msg(err)
		}

		dat, _ := ioutil.ReadFile(tmpfile.Name())
		gFilename := github.GistFilename(*file.Filename)
		files[gFilename] = github.GistFile{Filename: github.String(*file.Filename), Content: github.String(string(dat))}

		tmpfile.Write([]byte(""))
	}

	input := &github.Gist{Files: files}
	if _, _, err := client.Gists.Edit(*ctx, gistID, input); err != nil {
		return msg(err)
	}
	return 0
}
