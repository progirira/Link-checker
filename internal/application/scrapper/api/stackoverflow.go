package api

import (
	"encoding/json"
	"fmt"
	"go-progira/lib/e"
	"log/slog"
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
	return len(url) > 25 && url[:25] == "https://stackoverflow.com"
}

func GetStackOverflowUpdates(link string) (str string, err error) {
	parts := strings.Split(link, "/")
	linkID := parts[4]

	u := url.URL{
		Scheme: "https",
		Host:   "api.stackexchange.com",
		Path:   fmt.Sprintf("2.3/questions/%s/answers", linkID),
	}

	req, errReq := http.NewRequest(http.MethodGet, u.String(), http.NoBody)
	if errReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errReq.Error()),
		)

		return "", errReq
	}

	q := req.URL.Query()
	q.Add("order", "desc")
	q.Add("sort", "activity")
	q.Add("site", "stackoverflow")

	req.URL.RawQuery = q.Encode()

	body, errDoReq := doRequest(req)
	if errDoReq != nil {
		return "", e.ErrDoRequest
	}

	qs := questions{}

	if errDecode := json.Unmarshal(body, &qs); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()))

		return "", err
	}

	if len(qs.Items) > 0 {
		return fmt.Sprintf("%d", qs.Items[0].QuestionID), nil
	}

	return "", nil
}
