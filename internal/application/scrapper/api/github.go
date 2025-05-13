package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-progira/internal/domain/types/apitypes"
	"go-progira/internal/formatter"
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

func (updater *GithubUpdater) GetResponse(owner, repo, updateType string, prevUpdateTime time.Time) ([]apitypes.GithubUpdate, error) {
	log.Println("In GetNewUpdates function")

	urlString := fmt.Sprintf("https://api.github.com/repos/%s/%s/%s", owner, repo, updateType)

	if !prevUpdateTime.IsZero() {
		urlString += fmt.Sprintf("?date:>=%v", prevUpdateTime.Format(time.RFC3339))
	}

	req, errMakeReq := http.NewRequest(http.MethodGet, urlString, http.NoBody)
	if errMakeReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errMakeReq.Error()),
			slog.String("function", "Github updates"),
			slog.String("method", http.MethodGet),
			slog.String("url", urlString),
		)

		return []apitypes.GithubUpdate{}, e.ErrMakeRequest
	}

	body, err := doRequest(req)
	if errors.Is(err, e.ErrAPI) {
		return []apitypes.GithubUpdate{}, nil
	}

	var result []apitypes.GithubUpdate

	if errDecode := json.Unmarshal(body, &result); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return []apitypes.GithubUpdate{}, e.ErrDecodeJSONBody
	}

	slog.Info("Get Github updates ",
		slog.Int("Number of Github updates ", len(result)))

	return result, nil
}

func (updater *GithubUpdater) GetUpdates(link string, prevUpdateTime time.Time) (string, time.Time) {
	owner, repo, err := GetOwnerAndRepo(link)
	if err != nil {
		slog.Error(err.Error(),
			slog.String("link", link),
		)

		return "", prevUpdateTime
	}

	parts := strings.Split(link, "/")
	if len(parts) < 6 {
		log.Println("invalid github link: not enough parts")
		return "", prevUpdateTime
	}

	updateType := parts[5]

	updates, err := updater.GetResponse(owner, repo, updateType, prevUpdateTime)
	if err != nil {
		log.Printf("Error getting updates from Github: %s", err.Error())
		return "", prevUpdateTime
	}

	var githubType apitypes.GithubType

	lastTime := prevUpdateTime

	switch parts[5] {
	case "pulls":
		githubType = apitypes.PR
	case "issues":
		githubType = apitypes.Issue
	}

	for i, update := range updates {
		updates[i].Type = githubType

		t, err := time.Parse(time.RFC3339, update.CreatedAt)
		if err != nil {
			log.Printf("Error parsing time %v for update %s: %s", update.CreatedAt, link, err.Error())

			return "", lastTime
		}

		lastTime = t
	}

	return formatter.FormatMessageForGithub(updates), lastTime
}
