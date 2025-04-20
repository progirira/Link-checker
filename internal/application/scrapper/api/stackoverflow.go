package api

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/domain/types/api_types"
	"go-progira/pkg/e"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//type UpdaterFunc func(url string) (string, error)
//
//var updaters = map[string]UpdaterFunc{
//	"stackoverflow": GetStackOverflowUpdates,
//	"github":        GetGitHubUpdates,
//}
//
//func GetUpdater(url string) (UpdaterFunc, bool) {
//	switch {
//	case IsStackOverflowURL(url):
//		return updaters["stackoverflow"], true
//	case IsGitHubURL(url):
//		return updaters["github"], true
//	default:
//		return nil, false
//	}
//}

type StackoverflowUpdater struct {
	Key string
}

func IsStackOverflowURL(url string) bool {
	return len(url) > 25 && url[:25] == "https://stackoverflow.com"
}

func (updater *StackoverflowUpdater) GetTitle(questionID int) (string, error) {
	url := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/questions/%d?site=stackoverflow&filter=withbody",
		questionID,
	)
	url += fmt.Sprintf("&key=%s", updater.Key)

	resp, err := http.Get(url)
	if err != nil {
		slog.Error(err.Error())
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
	}

	if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
		slog.Error(e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()))
		return "", errDecode
	}

	if len(result.Items) == 0 {
		slog.Error("question not found, error getting title")
		return "", fmt.Errorf("question not found")
	}

	return result.Items[0].Title, nil
}

func (updater *StackoverflowUpdater) GetResponse(questionID int, updateType api_types.StackOverFlowType, prevUpdateTime time.Time) ([]api_types.StackOverFlowUpdate, error) {
	var format string

	if updateType == api_types.Answer {
		format = "https://api.stackexchange.com/2.3/questions/%d/answers?order=desc&sort=creation&site=stackoverflow&filter=withbody"
	} else if updateType == api_types.Comment {
		format = "https://api.stackexchange.com/2.3/questions/%d/comments?order=desc&sort=creation&site=stackoverflow&filter=withbody"
	}

	url := fmt.Sprintf(format, questionID)

	if !prevUpdateTime.IsZero() {
		url += fmt.Sprintf("&fromdate=%v", prevUpdateTime.Unix()+int64(1))
	}
	url += fmt.Sprintf("&key=%s", updater.Key)

	resp, err := http.Get(url)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []api_types.StackOverFlowUpdate `json:"items"`
	}

	if errDecode := json.NewDecoder(resp.Body).Decode(&result); errDecode != nil {
		slog.Error(e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()))
		return nil, err
	}

	return result.Items, nil
}

func (updater *StackoverflowUpdater) GetUpdates(link string, prevUpdateTime time.Time) []api_types.StackOverFlowUpdate {
	parts := strings.Split(link, "/")
	ID, errToInt := strconv.Atoi(parts[4])
	if errToInt != nil {
		log.Println("Error converting id to int", errToInt)
		return []api_types.StackOverFlowUpdate{}
	}

	title, err := updater.GetTitle(ID)
	if err != nil {
		log.Println("Error getting title ", errToInt)
		return []api_types.StackOverFlowUpdate{}
	}

	var updateType api_types.StackOverFlowType

	switch parts[5] {
	case "comments":
		updateType = api_types.Comment
	case "answers":
		updateType = api_types.Answer
	default:
		log.Printf("Unknown update type(not an answer or comment): %s", parts[5])

		return []api_types.StackOverFlowUpdate{}
	}

	updates, _ := updater.GetResponse(ID, updateType, prevUpdateTime)

	for _, update := range updates {
		update.Title = title
		fmt.Println(update.CreatedAt)
		updates = append(updates, update)
	}
	log.Println("Get comments in GetStackOverflowUpdates", updates)

	return updates
}
