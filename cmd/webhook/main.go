package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"net/http"

	"gopkg.in/go-playground/webhooks.v5/github"
)

const (
	path = "/webhooks"
)

func main() {
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
			fmt.Printf("%+v", release)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"action":"release"}`)
			qhook <- release
			return
		case github.PullRequestPayload:
			pullRequest := payload.(github.PullRequestPayload)
			// Do whatever you want from here...
			fmt.Printf("%+v", pullRequest)
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
	cmd := exec.Command("make webhook")
	id := fmt.Sprintf("%d", payload.PullRequest.ID)
	user := payload.PullRequest.User.Login
	branch := payload.PullRequest.Head.Ref
	cmd.Env = append(os.Environ(),
		"ChangeID="+id,
		"name="+user,
		"b="+branch,
	)
	if err := cmd.Run(); err != nil {
		log.Println("run make auto_ci", err)
	}
}
