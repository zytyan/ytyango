package bothttp

import (
	"github.com/gin-gonic/gin"
	"strconv"
)

type ErrCode int

const (
	ErrValidFailed ErrCode = 1001 + iota
	ErrExpired
	ErrNoAuth
)
const (
	ErrNoResource ErrCode = 2001 + iota
	ErrArgInvalid
)

const (
	UserNotFound ErrCode = 3001 + iota
	GroupNotFound
	UserNoProfilePhoto
)
const (
	ErrSearchFailed ErrCode = 4001 + iota
)

func (e ErrCode) Msg(msg string) gin.H {
	return gin.H{
		"status": "error",
		"code":   e,
		"error":  msg,
	}
}
func (e ErrCode) Error() string {
	return "error code " + strconv.Itoa(int(e))
}
