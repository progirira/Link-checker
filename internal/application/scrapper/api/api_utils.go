package api

import (
	"go-progira/lib/e"
	"io"
	"log/slog"
	"net/http"
)

func doRequest(request *http.Request) (body []byte, err error) {
	client := &http.Client{}

	response, errDoReq := client.Do(request)

	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return nil, e.ErrDoRequest
	}

	if response.StatusCode != 200 {
		slog.Error(
			e.ErrAPI.Error(),
			slog.String("function", "Github updates"),
			slog.Int("status code", response.StatusCode),
		)

		return nil, e.ErrAPI
	}

	body, errRead := io.ReadAll(response.Body)
	if errRead != nil {
		slog.Error(
			e.ErrReadBody.Error(),
			slog.String("error", errRead.Error()),
		)

		return nil, e.ErrReadBody
	}

	closeErr := response.Body.Close()
	if closeErr != nil {
		slog.Error(
			e.ErrCloseBody.Error(),
			slog.String("error", closeErr.Error()),
		)

		return nil, e.ErrCloseBody
	}

	return body, nil
}
