package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func main() {
	poolingFrequency := flag.Int("pooling-freq", 60, "default pooling frequency in seconds for repos")
	repoLocation := flag.String("repo-location", "", "location of repo to poll, currenly only supports git:// or https:// urls")
	repoUser := flag.String("repo-user", "", "repo username")
	repoPass := flag.String("repo-pass", "", "repo password")
	script := flag.String("script", "", "script to use on version change")
	workspace := flag.String("workspace-dir", "", "script to use on version change")
	verbose := flag.Bool("verbose", false, "verbose logging")
	flag.Parse()
	if *repoLocation == "" {
		fmt.Println("-repo-location is empty please set to run program")
		os.Exit(1)
	}
	if *repoUser == "" {
		fmt.Println("-repo-user is empty please set to run program")
		os.Exit(1)
	}
	if *repoPass == "" {
		fmt.Println("-repo-pass is empty please set to run program")
		os.Exit(1)
	}
	if *script == "" {
		fmt.Println("-script is empty please set to run program")
		os.Exit(1)
	}
	if *workspace == "" {
		fmt.Println("-workspace-dir is empty please set to run program")
		os.Exit(1)
	}
	currentCommit := ""
	for {
		isEmpty, err := IsEmpty(*workspace)
		if err != nil {
			log.Printf("unable to check of workspace %v is empty due to error '%v'", *workspace, err)
			continue
		}
		if isEmpty {
			if *verbose {
				log.Printf("workspace is empty so cloning")
			}
			r, err := git.PlainClone(*workspace, false, &git.CloneOptions{
				Progress: os.Stdout,
				//Depth: 1, cannot use depth due to this https://github.com/go-git/go-git/issues/328
				URL: *repoLocation,
				Auth: &http.BasicAuth{
					Username: *repoUser,
					Password: *repoPass,
				},
			})
			if *verbose {
				log.Printf("cloning complete")
			}
			if err != nil {
				log.Printf("unable to pull repo with error '%v'. sleeping", err)
				continue
			}
			h, err := r.ResolveRevision("HEAD")
			if err != nil {
				log.Printf("unable to read revision history due to error '%v'", err)
				continue
			}
			currentCommit = h.String()
			if *verbose {
				log.Printf("current commit is set to %v", currentCommit)
			}
			res, err := runScript(*script)
			if err != nil {
				log.Printf("unable to execute script '%v' due to error '%v'", script, err)
				continue
			}
			log.Println(res)
		} else {
			if *verbose {
				log.Println("pulling latest")
			}
			r, err := git.PlainOpen(*workspace)
			if err != nil {
				log.Printf("unable to read workspace %v due to error '%v'", *workspace, err)
				continue
			}
			w, err := r.Worktree()
			if err != nil {
				log.Printf("unable setup worktree for the workspace %v due to error '%v'", *workspace, err)
				continue
			}
			err = w.Pull(&git.PullOptions{
				Progress:   os.Stdout,
				RemoteName: "origin",
				Auth: &http.BasicAuth{
					Username: *repoUser,
					Password: *repoPass,
				}})
			if err != nil && err != git.NoErrAlreadyUpToDate {
				log.Printf("unable to pull repo due to error '%v'", err)
				continue
			}
			h, err := r.ResolveRevision("HEAD")
			if err != nil {
				log.Printf("unable to read revision due to error '%v'", err)
				continue
			}
			if h.String() != currentCommit {
				log.Printf("new commit is out %v, running script %v", h.String(), *script)
				res, err := runScript(*script)
				if err != nil {
					log.Printf("unable to execute script '%v' due to error '%v'", script, err)
					continue
				}
				log.Println(res)
			} else if *verbose {
				log.Printf("commit is still set to %v no need to run script", currentCommit)
			}
		}
		time.Sleep(time.Duration(*poolingFrequency) * time.Second)
	}
}

func runScript(script string) (string, error) {
	res, err := exec.Command(script).CombinedOutput()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("OUTPUT: %s", res), nil
}

// IsEmpty is borrowed from https://stackoverflow.com/questions/30697324/how-to-check-if-directory-on-path-is-empty
func IsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
