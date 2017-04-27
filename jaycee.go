package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"os"
	"regexp"
	"strconv"
	//"golang.org/x/oauth2"
	"gopkg.in/urfave/cli.v1"
)

func extractPullInfo(url string) (string, string, int, error) {
	r, _ := regexp.Compile(`https\:\/\/github.com\/([^\/]+)\/([^\/]+)\/pull\/(\d+)`)
	matches := r.FindStringSubmatch(url)
	if len(matches) != 4 {
		return "", "", -1, errors.New(fmt.Sprintf("Invalid pull request URL: %s", url))
	}
	intNumber, _ := strconv.Atoi(matches[3])
	return matches[1], matches[2], intNumber, nil
}

func validateBranch(branch string) error {
	matched, err := regexp.MatchString(`^((\d+\.(\d+|x))|master)$`, branch)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New(fmt.Sprintf("Invalid branch name: %s", branch))
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "jaycee"
	app.Version = "1.0.0-dev"
	app.Usage = "a backporting tool, and maybe some other stuff"

	app.Commands = []cli.Command{
		{
			Name:  "backport",
			Usage: "backports a pull request to another branch",
			Action: func(clictx *cli.Context) error {
				if clictx.NArg() == 0 {
					return cli.NewExitError("You must specify a pull request URL to backport", 1)
				}

				url := clictx.Args().Get(0)
				orgName, repoName, pullNumber, err := extractPullInfo(url)
				if err != nil {
					return err
				}

				branch := clictx.Args().Get(1)
				err = validateBranch(branch)
				if err != nil {
					return err
				}

				ctx := context.Background()
				/*
				   ts := oauth2.StaticTokenSource(
				     &oauth2.Token{AccessToken: ""},
				   )
				   tc := oauth2.NewClient(ctx, ts)
				   client := github.NewClient(tc)
				*/
				client := github.NewClient(nil)
				pr, response, err := client.PullRequests.Get(ctx, orgName, repoName, pullNumber)
				fmt.Printf("%d of %d remaining\n", response.Remaining, response.Limit)
				if _, ok := err.(*github.RateLimitError); ok {
					fmt.Println("hit rate limit")
					return err
				}
				if pr == nil {
					return errors.New("Can not find PR")
				}
				fmt.Println(*pr.Title)

				fmt.Printf("backported %q\n", clictx.Args().Get(0))
				return nil
			},
		},
	}

	app.Run(os.Args)
}
