package backend

import "errors"

var (
	errGroupNotFound    = errors.New("group not found")
	errUserNotFound     = errors.New("user not found")
	errUserNoPhoto      = errors.New("user has no profile photo")
	errBotUnavailable   = errors.New("bot unavailable")
)
