package api

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/domain/types/api_types"
	"go-progira/pkg/e"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type GithubUpdater struct{}

func IsGitHubURL(url string) bool {
	return len(url) >= 18 && url[:18] == "https://github.com"
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

func (updater *GithubUpdater) GetResponse(owner, repo, updateType string, prevUpdateTime time.Time) ([]api_types.GithubUpdate, error) {
	log.Println("In GetNewUpdates function")

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/%s", owner, repo, updateType)

	if !prevUpdateTime.IsZero() {
		url += fmt.Sprintf("?date:>=%d", prevUpdateTime.Format(time.RFC3339))
	}

	resp, err := http.Get(url)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	var result []api_types.GithubUpdate

	if resp.StatusCode != http.StatusOK {
		log.Println(resp.StatusCode, resp.Status)
		return []api_types.GithubUpdate{}, nil
	}

	if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return []api_types.GithubUpdate{}, e.ErrDecodeJSONBody
	}

	fmt.Println("len(newUpdates) ", len(result))

	return result, nil
}

func (updater *GithubUpdater) GetUpdates(link string, prevUpdateTime time.Time) []api_types.GithubUpdate {
	owner, repo, err := GetOwnerAndRepo(link)
	if err != nil {
		slog.Error(err.Error(),
			slog.String("link", link),
		)

		return []api_types.GithubUpdate{}
	}
	parts := strings.Split(link, "/")
	if len(parts) < 6 {
		log.Println("invalid github link: not enough parts")
		return nil
	}
	updateType := parts[5]

	updates, err := updater.GetResponse(owner, repo, updateType, prevUpdateTime)
	if err != nil {
		log.Printf("Error getting updates from Github: %s", err.Error())
		return []api_types.GithubUpdate{}
	}

	var githubType api_types.GithubType

	switch parts[5] {
	case "pulls":
		githubType = api_types.PR
	case "issues":
		githubType = api_types.Issue
	}

	for i := range updates {
		updates[i].Type = githubType
	}

	return updates
}
