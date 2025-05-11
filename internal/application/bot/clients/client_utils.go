package clients

import (
	"bytes"
	"go-progira/pkg/e"
	"log/slog"
	"net/http"
	"net/url"
)

func DoRequest(client http.Client, method, scheme, host, path string, q url.Values, body []byte, isJSON bool) (*http.Response, error) {
	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}

	req, errMakeReq := http.NewRequest(method, u.String(), bytes.NewBuffer(body))
	if errMakeReq != nil {
		slog.Error(
			e.ErrMakeRequest.Error(),
			slog.String("error", errMakeReq.Error()),
			slog.String("method", method),
			slog.String("url", u.String()),
			slog.String("body", string(body)),
		)

		return nil, e.ErrMakeRequest
	}

	if isJSON {
		req.Header.Set("Content-Type", "application/json")
	}

	req.URL.RawQuery = q.Encode()

	resp, errDoReq := client.Do(req)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return nil, e.ErrDoRequest
	}

	return resp, nil
}
