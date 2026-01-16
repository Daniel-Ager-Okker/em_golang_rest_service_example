package storage

import (
	"errors"
)

var (
	ErrSubscribtionNotFound = errors.New("subscription not found")
	ErrSubscriptionExists   = errors.New("subscription exists")
)
