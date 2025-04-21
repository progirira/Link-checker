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
	"strconv"
	"strings"
	"time"
)

type UpdaterFunc func(url string) (string, error)

var updaters = map[string]Updater{}

func InitUpdaters(stackOverflowKey string) {
	updaters = map[string]Updater{
		"stackoverflow": &StackoverflowUpdater{Key: stackOverflowKey},
		"github":        &GithubUpdater{},
	}
}

func GetUpdater(url string) (Updater, bool) {
	switch {
	case IsStackOverflowURL(url):
		return updaters["stackoverflow"], true
	case IsGitHubURL(url):
		return updaters["github"], true
	default:
		return nil, false
	}
}

type Updater interface {
	GetUpdates(link string, prevUpdateTime time.Time) (string, time.Time)
}

type StackoverflowUpdater struct {
	Key string
}

func IsStackOverflowURL(url string) bool {
	return len(url) > 25 && url[:25] == "https://stackoverflow.com"
}

func (updater *StackoverflowUpdater) GetTitle(questionID int) (string, error) {
	urlString := fmt.Sprintf(
		"https://api.stackexchange.com/2.3/questions/%d?site=stackoverflow&filter=withbody",
		questionID,
	)

	urlString += fmt.Sprintf("&key=%s", updater.Key)

	req, errMakeReq := http.NewRequest(http.MethodGet, urlString, http.NoBody)
	if errMakeReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errMakeReq.Error()),
			slog.String("function", "Github updates"),
			slog.String("method", http.MethodGet),
			slog.String("url", urlString),
		)

		return "", e.ErrMakeRequest
	}

	body, err := doRequest(req)
	if errors.Is(err, e.ErrAPI) {
		return "", nil
	}

	var result struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
	}

	if errDecode := json.Unmarshal(body, &result); errDecode != nil {
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

func (updater *StackoverflowUpdater) GetResponse(questionID int, updateType apitypes.StackOverFlowType,
	prevUpdateTime time.Time) ([]apitypes.StackOverFlowUpdate, error) {
	var format string

	if updateType == apitypes.Answer {
		format = "https://api.stackexchange.com/2.3/questions/%d/answers?order=desc&sort=creation&site=stackoverflow&filter=withbody"
	} else if updateType == apitypes.Comment {
		format = "https://api.stackexchange.com/2.3/questions/%d/comments?order=desc&sort=creation&site=stackoverflow&filter=withbody"
	}

	urlString := fmt.Sprintf(format, questionID)

	if !prevUpdateTime.IsZero() {
		urlString += fmt.Sprintf("&fromdate=%v", prevUpdateTime.Unix()+int64(1))
	}

	urlString += fmt.Sprintf("&key=%s", updater.Key)

	req, errMakeReq := http.NewRequest(http.MethodGet, urlString, http.NoBody)
	if errMakeReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errMakeReq.Error()),
			slog.String("function", "Github updates"),
			slog.String("method", http.MethodGet),
			slog.String("url", urlString),
		)

		return []apitypes.StackOverFlowUpdate{}, e.ErrMakeRequest
	}

	body, err := doRequest(req)
	if errors.Is(err, e.ErrAPI) {
		return []apitypes.StackOverFlowUpdate{}, nil
	}

	var result struct {
		Items []apitypes.StackOverFlowUpdate `json:"items"`
	}

	if errDecode := json.Unmarshal(body, &result); errDecode != nil {
		slog.Error(e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()))

		return nil, err
	}

	slog.Info("Get Stackoverflow updates ",
		slog.Int("Number of Github updates ", len(result.Items)))

	return result.Items, nil
}

func (updater *StackoverflowUpdater) GetUpdates(link string, prevUpdateTime time.Time) (string, time.Time) {
	parts := strings.Split(link, "/")

	ID, errToInt := strconv.Atoi(parts[4])
	if errToInt != nil {
		log.Println("Error converting id to int", errToInt)
		return "", prevUpdateTime
	}

	title, err := updater.GetTitle(ID)
	if err != nil {
		log.Println("Error getting title ", err)
		return "", prevUpdateTime
	}

	var updateType apitypes.StackOverFlowType

	switch parts[5] {
	case "comments":
		updateType = apitypes.Comment
	case "answers":
		updateType = apitypes.Answer
	default:
		log.Printf("Unknown update type(not an answer or comment): %s", parts[5])

		return "", prevUpdateTime
	}

	updates, _ := updater.GetResponse(ID, updateType, prevUpdateTime)
	lastTime := prevUpdateTime

	for _, update := range updates {
		update.Title = title
		lastTime = time.Unix(update.CreatedAt, 0)
	}

	log.Printf("Get %d comments in GetStackOverflowUpdates", len(updates))

	return formatter.FormatMessageForStackOverflow(updates), lastTime
}
