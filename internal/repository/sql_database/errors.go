package repository

import "errors"

var (
	ErrPoolCreate = errors.New("error creating pool")
	ErrCreateChat = errors.New("error creating chat")
	ErrDeleteChat = errors.New("error deleting chat")
	ErrExecQuery  = errors.New("error executing query")
	ErrRemoveLink = errors.New("error removing link")
)
