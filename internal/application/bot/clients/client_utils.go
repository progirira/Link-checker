package clients

import (
	"bytes"
	"fmt"
	"go-progira/lib/e"
	"io"
	"net/http"
	"net/url"
)

func DoRequest(client http.Client, method, scheme, host, path string, query url.Values, body []byte) (io.ReadCloser, error) {
	const errMsg = "can't do request"

	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}
	fmt.Println(u)

	req, err := http.NewRequest(method, u.String(), bytes.NewBuffer(body))
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
