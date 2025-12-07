package backend

import "errors"

var (
	errUserNotFound   = errors.New("user not found")
	errUserNoPhoto    = errors.New("user has no profile photo")
	errBotUnavailable = errors.New("bot unavailable")
)
