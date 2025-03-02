package scrapper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func CheckStackOverflowUpdates(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("Stack Overflow API error: %s", response.Status)
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
		return fmt.Sprintf("%d", qs.Items[0].QuestionID), nil // Возвращаем ID самого нового вопроса
	}

	return "", nil
}
