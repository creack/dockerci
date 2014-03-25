package main

import (
	"encoding/json"
	"github.com/bitly/go-nsq"
	"github.com/drone/go-github/github"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

var writer *nsq.Writer

func pullRequest(w http.ResponseWriter, r *http.Request) {
	hook, err := github.ParsePullRequestHook([]byte(r.FormValue("payload")))
	if err != nil {
		log.Fatal(err)
	}
	switch hook.Action {
	case "open", "synchronize":
		data, err := json.Marshal(hook.PullRequest)
		if err != nil {
			log.Fatal(err)
		}
		if err := writer.PublishAsync("builds", data, nil); err != nil {
			log.Fatal(err)
		}
	default:
		log.Printf("event=%s action=%s\n", r.Header.Get("X-Github-Event"), hook.Action)
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.Host("api.github.com")
	r.HandleFunc("/", ping).Headers("X-Github-Event", "ping").Methods("POST")
	r.HandleFunc("/", pullRequest).Headers("X-Github-Event", "pull_request").Methods("POST")
	return r
}

func main() {
	writer = nsq.NewWriter(os.Getenv("NSQD"))
	defer writer.Close()

	if err := http.ListenAndServe(":80", newRouter()); err != nil {
		log.Fatal(err)
	}
}
