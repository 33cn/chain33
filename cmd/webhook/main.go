package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"gopkg.in/go-playground/webhooks.v5/github"
)

const (
	path = "/webhooks"
)

var project = flag.String("p", "chain33", "the github project name")

func main() {
	flag.Parse()
	hook, _ := github.New(github.Options.Secret(""))
	qhook := make(chan interface{}, 64)
	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, github.ReleaseEvent, github.PullRequestEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}
		w.Header().Set("Content-type", "application/json")
		switch payload.(type) {
		case github.ReleasePayload:
			release := payload.(github.ReleasePayload)
			// Do whatever you want from here...
			//fmt.Printf("%+v", release)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"action":"release"}`)
			qhook <- release
			return
		case github.PullRequestPayload:
			pullRequest := payload.(github.PullRequestPayload)
			// Do whatever you want from here...
			//fmt.Printf("%+v", pullRequest)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"action":"pullrequest"}`)
			qhook <- pullRequest
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"action":"unkown request event"}`)
	})
	go webhooksProcess(qhook)
	http.ListenAndServe(":3000", nil)
}

func webhooksProcess(ch chan interface{}) {
	for payload := range ch {
		switch payload.(type) {
		case github.ReleasePayload:
			break
		case github.PullRequestPayload:
			processGithubPL(payload.(github.PullRequestPayload))
		}
	}
}

func processGithubPL(payload github.PullRequestPayload) {
	gopath := os.Getenv("GOPATH")

	id := fmt.Sprintf("%d", payload.PullRequest.ID)
	user := payload.PullRequest.User.Login
	branch := payload.PullRequest.Head.Ref

	repo := "https://github.com/" + user + "/" + *project + ".git"
	gitpath := gopath + "/src/github.com/" + user + "/" + *project
	log.Println("git path", gitpath)
	cmd := exec.Command("rm", "-rf", gitpath)
	if err := cmd.Run(); err != nil {
		log.Println("rm -rf ", gitpath, err)
	}
	cmd = exec.Command("git", "clone", "--depth", "50", repo, gitpath)
	if err := cmd.Run(); err != nil {
		log.Println("run git", err)
	}
	cmd = exec.Command("make", "webhook")
	cmd.Dir = gitpath
	cmd.Env = append(os.Environ(),
		"ChangeID="+id,
		"name="+user,
		"b="+branch,
	)
	if err := cmd.Run(); err != nil {
		log.Println("run make webhook", err)
	}
}
