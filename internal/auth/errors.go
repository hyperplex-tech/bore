package auth

import "errors"

// ErrNoAuthMethods is returned when no authentication methods could be resolved.
var ErrNoAuthMethods = errors.New("no SSH authentication methods available")
