package models

import (
	"errors"
)

var (
	ErrInternal = errors.New("internal")
	ErrNotFound = errors.New("not found")
	ErrExists = errors.New("exists")
	ErrConflict = errors.New("conflict")
	ErrRaiseEx = errors.New("raise")
	// ErrNotNull = errors.New("not null")
)
