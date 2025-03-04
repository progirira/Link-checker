package clients

import (
	"bytes"
	"go-progira/lib/e"
	"io"
	"net/http"
	"net/url"
	"path"
)

func DoRequest(client http.Client, method, host, basePath string, query url.Values, body []byte) (io.ReadCloser, error) {
	const errMsg = "can't do request"

	u := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path.Join(basePath, method),
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, e.Wrap(errMsg, err)
	}

	req.URL.RawQuery = query.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, e.Wrap(errMsg, err)
	}
	//defer resp.Body.Close()

	//respBody, err := resp.Body
	//if err != nil {
	//	return nil, err
	//}

	return resp.Body, nil
}
