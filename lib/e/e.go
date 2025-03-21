package e

import (
	"errors"
)

var (
	ErrOpenFile   = errors.New("error opening file")
	ErrCloseFile  = errors.New("error closing file")
	ErrNoValInEnv = errors.New("value is not specified in .env file")
	ErrOsSetEnv   = errors.New("error os setting env")
	ErrScanFile   = errors.New("error scanning file")

	ErrMakeRequest      = errors.New("error making request")
	ErrDoRequest        = errors.New("error doing request")
	ErrWrongURLFormat   = errors.New("URL this form does not require")
	ErrMethodNotAllowed = errors.New("method not allowed")

	ErrMarshalJSON    = errors.New("error marshaling json")
	ErrDecodeJSONBody = errors.New("error decoding json body")
	ErrEncodeToJSON   = errors.New("error encoding json")

	ErrNoOwnerAndRepoInPath = errors.New("no owner and repository in URL")
	ErrNoRepoInPath         = errors.New("repository is not specified in URL")
	ErrAPI                  = errors.New("API returned error")
	ErrReadBody             = errors.New("read body error")
	ErrCloseBody            = errors.New("close body error")

	ErrWrite        = errors.New("write error")
	ErrServerFailed = errors.New("server failed")
	ErrScheduler    = errors.New("scheduler error")

	ErrChatNotFound      = errors.New("chat not found")
	ErrLinkAlreadyExists = errors.New("link already exists")
	ErrChatAlreadyExists = errors.New("link already exists")
)
