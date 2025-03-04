package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Commits []struct {
	Sha string `json:"sha"`
}

func IsGitHubURL(url string) bool {
	return len(url) >= 18 && url[:18] == "https://github.com"
}

func CheckGitHubUpdates(repoName string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits", repoName)

	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		fmt.Printf("GitHub API error: %s", response.Status)
		return "", fmt.Errorf("GitHub API error: %s", response.Status)
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
		return commits[0].Sha, nil
	}

	return "", nil
}
