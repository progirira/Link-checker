package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Commits []struct {
	Sha string `json:"sha"`
}

func IsGitHubURL(url string) bool {
	return len(url) >= 18 && url[:18] == "https://github.com"
}

func (c Commits) getGeneralSha() string {
	sumSha := strings.Builder{}
	for _, commit := range c {
		sumSha.WriteString(commit.Sha)
	}
	return sumSha.String()
}

func CheckGitHubUpdates(link string) (string, error) {
	parts := strings.Split(link, "/")
	owner := parts[3]
	repo := parts[4]

	u := url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   fmt.Sprintf("repos/%s/%s/commits", owner, repo),
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), http.NoBody)

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(response.Body)

	if response.StatusCode != 200 {
		return "", fmt.Errorf("stack Overflow API error: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	commits := Commits{}

	if err := json.Unmarshal(body, &commits); err != nil {
		return "", err
	}

	if len(commits) > 0 {
		return commits.getGeneralSha(), nil
	}

	return "", nil
}
