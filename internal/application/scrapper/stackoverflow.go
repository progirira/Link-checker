package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type questions struct {
	Items []item `json:"items"`
}

type item struct {
	QuestionID int64 `json:"question_id"`
}

func IsStackOverflowURL(url string) bool {
	return len(url) >= 25 && url[:25] == "https://stackoverflow.com"
}

func GetStackOverflowUpdates(link string) (string, error) {
	parts := strings.Split(link, "/")
	linkID := parts[4]

	u := url.URL{
		Scheme: "https",
		Host:   "api.stackexchange.com",
		Path:   fmt.Sprintf("2.3/questions/%s/answers", linkID),
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), http.NoBody)

	q := req.URL.Query()
	q.Add("order", "desc")
	q.Add("sort", "activity")
	q.Add("site", "stackoverflow")

	req.URL.RawQuery = q.Encode()
	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("stack Overflow API error: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	qs := questions{}

	if err := json.Unmarshal(body, &qs); err != nil {
		return "", err
	}

	if len(qs.Items) > 0 {
		return fmt.Sprintf("%d", qs.Items[0].QuestionID), nil
	}

	return "", nil
}
