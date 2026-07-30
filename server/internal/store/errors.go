package store

import "errors"

var ErrConflict = errors.New("conflict: referenced resource")
