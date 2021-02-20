package errors

import "errors"

var ErrServerClosed = errors.New("gateway: Server closed")
var ErrNotFound = errors.New("not found")
