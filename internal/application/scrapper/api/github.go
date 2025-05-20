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
	"regexp"
	"strings"
	"time"
)

type GithubUpdater struct {
	Key string
}

func IsGitHubURL(url string) bool {
	patternPulls := `^https://github\.com/[\w\-]+/[\w\-]+/pulls$`
	patternIssues := `^https://github\.com/[\w\-]+/[\w\-]+/issues$`

	patterns := []string{patternPulls, patternIssues}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(url) {
			return true
		}
	}

	return false
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

func (updater *GithubUpdater) GetResponse(owner, repo string, updateType apitypes.GithubType,
	prevUpdateTime time.Time) ([]apitypes.GithubUpdate, error) {
	since := prevUpdateTime.UTC().Format(time.RFC3339)

	urlString := fmt.Sprintf("https://api.github.com/search/issues?q=repo:%s/%s+type:%s+updated:>%v",
		owner, repo, updateType.StringForRequest(), since)

	fmt.Println(urlString)

	req, errMakeReq := http.NewRequest(http.MethodGet, urlString, http.NoBody)
	if errMakeReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errMakeReq.Error()),
			slog.String("method", http.MethodGet),
			slog.String("url", urlString),
		)

		return []apitypes.GithubUpdate{}, e.ErrMakeRequest
	}

	req.Header.Set("Authorization", "Bearer "+updater.Key)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "LinkChecker")

	body, err := doRequest(req)
	if errors.Is(err, e.ErrAPI) {
		return []apitypes.GithubUpdate{}, nil
	}

	if len(body) == 0 {
		return []apitypes.GithubUpdate{}, nil
	}

	var result struct {
		Items []apitypes.GithubUpdate `json:"items"`
	}

	if errDecode := json.Unmarshal(body, &result); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return []apitypes.GithubUpdate{}, e.ErrDecodeJSONBody
	}

	return result.Items, nil
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

	var githubType apitypes.GithubType

	switch parts[5] {
	case "pulls":
		githubType = apitypes.PR
	case "issues":
		githubType = apitypes.Issue
	default:
		log.Printf("Unknown update type(not a pullRequest and not an issue): %s", parts[5])

		return "", prevUpdateTime
	}

	updates, err := updater.GetResponse(owner, repo, githubType, prevUpdateTime)
	if err != nil {
		log.Printf("Error getting updates from Github: %s", err.Error())
		return "", prevUpdateTime
	}

	lastTime := prevUpdateTime

	var filteredUpdates []apitypes.GithubUpdate

	for _, update := range updates {
		updateTime, err := time.Parse(time.RFC3339, update.CreatedAt)
		if err != nil {
			log.Printf("Error parsing time %v for update %s: %s", update.CreatedAt, link, err.Error())

			return "", lastTime
		}

		updateLocalTime := updateTime.In(time.Local)

		if updateLocalTime.After(prevUpdateTime) {
			update.Type = githubType
			update.CreatedAt = updateLocalTime.Format(time.RFC3339)
			filteredUpdates = append(filteredUpdates, update)

			if updateTime.After(lastTime) {
				lastTime = updateLocalTime
			}
		}
	}

	slog.Info("Get Github updates ",
		slog.Int("Number of updates ", len(filteredUpdates)))

	return formatter.FormatMessageForGithub(filteredUpdates), lastTime.In(time.Local)
}
