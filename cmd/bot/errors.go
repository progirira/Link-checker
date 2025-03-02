package main

import "errors"

var ErrNoToken = errors.New("token is not specified")
var ErrNoBotHost = errors.New("bot host is not specified")
