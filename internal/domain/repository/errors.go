package repository

import "errors"

var (
	ErrPartNotFound      = errors.New("part not found")
	ErrServiceNotFound   = errors.New("service not found")
	ErrWorkOrderNotFound = errors.New("work order not found")
)
