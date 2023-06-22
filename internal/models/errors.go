package models

import (
	"errors"
)

var (
	ErrInternal = errors.New("internal")
	ErrNotFound = errors.New("not found")
	ErrExists = errors.New("exists")
	// ErrNotNull = errors.New("not null")
)
