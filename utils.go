package dockerci

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"log"
	"os/exec"
)

type Result struct {
	Success bool
	Output  string
	Method  string
}

func (r *Result) ToData() map[string]string {
	var (
		stateKey = fmt.Sprintf("method-%s", r.Method)
		data     = map[string]string{
			fmt.Sprintf("%s-output", stateKey): r.Output,
			fmt.Sprintf("%s-result", stateKey): "failed",
		}
	)

	if r.Success {
		data[fmt.Sprintf("%s-result", stateKey)] = "passed"
	}
	return data
}

func GetRepoNameAndSha(json *simplejson.Json) (string, string, error) {
	repo, pullrequest := json.Get("repository"), json.Get("pull_request")
	repoName, err := repo.Get("name").String()
	if err != nil {
		return "", "", err
	}
	sha, err := pullrequest.Get("head").Get("sha").String()
	if err != nil {
		return "", "", err
	}
	return repoName, sha, nil
}

func Checkout(temp string, json *simplejson.Json) error {
	// git clone -qb master https://github.com/upstream/docker.git our-temp-directory
	base := json.Get("base")
	ref, err := base.Get("ref").String()
	if err != nil {
		return err
	}
	url, err := base.Get("repo").Get("clone_url").String()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", "-qb", ref, url, temp)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(output))
		return err
	}

	head := json.Get("head")
	url, err = head.Get("repo").Get("clone_url").String()
	if err != nil {
		return err
	}
	ref, err = head.Get("ref").String()
	if err != nil {
		return err
	}
	log.Printf("ref=%s url=%s\n", ref, url)
	// cd our-temp-directory && git pull -q https://github.com/some-user/docker.git some-feature-branch
	cmd = exec.Command("git", "pull", "-q", url, ref)
	cmd.Dir = temp
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Println(string(output))
		return err
	}
	return nil

}

func MakeTest(temp, method string) (*Result, error) {
	var (
		result = &Result{Method: method}
		cmd    = exec.Command("make", method)
	)
	cmd.Dir = temp

	output, err := cmd.CombinedOutput()
	if err != nil {
		// it's ok for the make command to return a non-zero exit
		// incase of a failed build
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, err
		}
	} else {
		result.Success = true
	}
	result.Output = string(output)

	return result, nil
}
