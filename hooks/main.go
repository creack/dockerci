package main

import (
	"github.com/bitly/go-nsq"
	"github.com/bitly/go-simplejson"
	"github.com/crosbymichael/dockerci"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

var (
	writer *nsq.Writer
	store  *dockerci.Store
)

func pullRequest(w http.ResponseWriter, r *http.Request) {
	var (
		rawPayload, json = getPayloadAndJson(r)
		action           = getAction(json)
	)

	log.Printf("event=%s action=%s\n", r.Header.Get("X-Github-Event"), action)
	if err := processAction(action, json, rawPayload); err != nil {
		log.Fatal(err)
	}
}

func getPayloadAndJson(r *http.Request) ([]byte, *simplejson.Json) {
	rawPayload := []byte(r.FormValue("payload"))

	json, err := simplejson.NewJson(rawPayload)
	if err != nil {
		log.Fatal(err)
	}
	return rawPayload, json
}

func getAction(json *simplejson.Json) string {
	action, err := json.Get("action").String()
	if err != nil {
		log.Fatal(err)
	}
	return action
}

func processAction(action string, json *simplejson.Json, rawPayload []byte) error {
	sha, err := dockerci.GetSha(json)
	if err != nil {
		return err
	}
	number, err := json.Get("number").Int()
	if err != nil {
		return err
	}
	if err := store.IncrementRequest(action); err != nil {
		return err
	}

	switch action {
	case "opened", "synchronize":
		// check that the commit for this PR is not already in the queue or processed
		if err := store.AtomicSaveState(sha, "pending"); err != nil {
			if err == dockerci.ErrKeyIsAlreadySet {
				return nil
			}
			return err
		}
		if err := store.SaveCommitForPullRequest(number, sha); err != nil {
			return err
		}
		if err := writer.PublishAsync("builds", rawPayload, nil); err != nil {
			return err
		}
	}
	return nil
}

func ping(w http.ResponseWriter, r *http.Request) {
	log.Printf("event=ping\n")
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/github", ping).Headers("X-Github-Event", "ping").Methods("POST")
	r.HandleFunc("/github", pullRequest).Headers("X-Github-Event", "pull_request").Methods("POST").Host("api.github.com")
	return r
}

func main() {
	writer = nsq.NewWriter(os.Getenv("NSQD"))
	store = dockerci.New(os.Getenv("REDIS"), os.Getenv("REDIS_AUTH"))
	defer store.Close()

	if err := http.ListenAndServe(":80", newRouter()); err != nil {
		log.Fatal(err)
	}
}
