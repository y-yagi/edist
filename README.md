# Edist

Edist is a CLI tool for editing [GitHub Gist](https://gist.github.com) in the local editor.

## Installation

```bash
$ go get github.com/y-yagi/edist
```

Or you can download from [Releases](https://github.com/y-yagi/edist/releases).

## How to use

### Preparation

You need a GitHub access token to use Edist. Please generate a new token from [Developer settings](https://github.com/settings/tokens). When generates a token, please select `gist` scopes.

### Configure

Edist uses vim by default. You can change the default editor by a config file. Please run `edist -c` and specify your editor to `editor` key.

You can set GitHub access token to a config file. GitHub access token can be specified to ENV(`GITHUB_ACCESS_TOKEN`) also.

### Run

```
$ edist --help
Usage: edist [OPTIONS]

OPTIONS:
  -c	cdit config.
  -e ID
    	edit Gist that ID was specified
  -l user
    	show list a user Gists
  -n filename
    	create a new Gist by filename
  -v	print version number
```

If you want to edit an existing Gist, specify an ID.  "ID" be can confirm in Gist's URL(It is a value of after account name). Or you can check the ID `edist -l` command also.
