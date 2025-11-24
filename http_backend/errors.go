package http_backend

// ErrCode matches the numeric codes exposed in ErrorResponse.
type ErrCode int64

const (
	// 100x auth/validation
	ErrValidFailed ErrCode = 1001 + iota
	ErrExpired
	ErrNoAuth
)

const (
	// 200x resource missing / arg invalid
	ErrNoResource ErrCode = 2001 + iota
	ErrArgInvalid
)

const (
	// 300x user/group domain errors
	UserNotFound ErrCode = 3001 + iota
	GroupNotFound
	UserNoProfilePhoto
)

const (
	// 400x search
	ErrSearchFailed ErrCode = 4001 + iota
)
