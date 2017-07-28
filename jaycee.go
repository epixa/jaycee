package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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

func homeDir() (string, error) {
	var homeDir string
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
	} else {
		homeDir = os.Getenv("HOME")
	}

	if _, err := os.Stat(homeDir); err != nil {
		return "", err
	}

	return homeDir, nil
}

func Extend(slice []int, element int) []int {
  n := len(slice)
  slice = slice[0 : n+1]
  slice[n] = element
  return slice
}

func execute(cmdArgs ...string) (string, error) {
	var (
		err error
		cmdOut []byte
	)
	if cmdOut, err = exec.Command(cmdArgs...).Output(); err != nil {
		return "", err
	}
	return string(cmdOut), nil
}

func git(cmdArgs ...string) (string, error) {
	cmdArgs := append([]string{"sh", "-c"}, args...)
	fmt.Println(cmdArgs)
	output, err := execute(cmdArgs...)
	if err != nil {
		return "", err
	}
	return output, nil
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

				home, err := homeDir()
				if err != nil {
					return err
				}
				tokenPath := filepath.Join(home, ".elastic", "github.token")
				rawToken, err := ioutil.ReadFile(tokenPath)
				if err != nil {
					return err
				}
				token := strings.TrimSpace(string(rawToken))
				if token == "" {
					return errors.New("No token found")
				}

				ctx := context.Background()
				ts := oauth2.StaticTokenSource(
					&oauth2.Token{AccessToken: token},
				)
				tc := oauth2.NewClient(ctx, ts)
				client := github.NewClient(tc)
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

				output, err := git("git", "status")
				if err != nil {
					return err
				}
				fmt.Println(output)

				fmt.Printf("backported %q\n", clictx.Args().Get(0))
				return nil
			},
		},
	}

	app.Run(os.Args)
}
