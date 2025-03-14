package scrapper

import (
	"encoding/json"
	"fmt"
	"go-progira/lib/e"
	"log/slog"
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

func GetOwnerAndRepo(link string) (owner, repo string, err error) {
	parts := strings.Split(link, "/")

	if len(parts) < 5 {
		if parts[3] != "" {
			err = e.ErrNoRepoInPath
		} else {
			err = e.ErrNoOwnerAndRepoInPath
		}

		return "", "", err
	}

	owner = parts[3]
	repo = parts[4]

	return owner, repo, nil
}

func CheckGitHubUpdates(link string) (string, error) {
	owner, repo, err := GetOwnerAndRepo(link)
	if err != nil {
		slog.Error(err.Error(),
			slog.String("link", link),
		)

		return "", err
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   fmt.Sprintf("repos/%s/%s/commits", owner, repo),
	}

	req, errMakeReq := http.NewRequest(http.MethodGet, u.String(), http.NoBody)
	if errMakeReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errMakeReq.Error()),
			slog.String("function", "Github updates"),
			slog.String("method", http.MethodGet),
			slog.String("url", u.String()),
		)

		return "", e.ErrMakeRequest
	}

	body, err := doRequest(req)
	if err != nil {
		return "", err
	}

	commits := Commits{}

	if errDecode := json.Unmarshal(body, &commits); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return "", e.ErrDecodeJSONBody
	}

	if len(commits) > 0 {
		return commits.getGeneralSha(), nil
	}

	return "", nil
}
