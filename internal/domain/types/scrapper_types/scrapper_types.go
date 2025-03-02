package scrappertypes

import "time"

type LinkResponse struct {
	ID          int64     `json:"id"`
	URL         string    `json:"url"`
	Tags        []string  `json:"tags"`
	Filters     []string  `json:"filters"`
	LastChecked time.Time `json:"last_checked"`
	LastVersion string    `json:"last_version"`
}

type APIErrorResponse struct {
	Description      string   `json:"description"`
	Code             string   `json:"code"`
	ExceptionName    string   `json:"exceptionName"`
	ExceptionMessage string   `json:"exceptionMessage"`
	Stacktrace       []string `json:"stacktrace,omitempty"`
}

type AddLinkRequest struct {
	Link    string   `json:"link"`
	Tags    []string `json:"tags"`
	Filters []string `json:"filters"`
}

type ListLinksResponse struct {
	Links []LinkResponse `json:"links"`
	Size  int32          `json:"size"`
}

type RemoveLinkRequest struct {
	Link string `json:"link"`
}

type Chat struct {
	ID    int64          `json:"id"`
	Links []LinkResponse `json:"links"`
}
