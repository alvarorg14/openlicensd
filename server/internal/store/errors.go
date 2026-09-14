package store

import "errors"

var ErrConflict = errors.New("conflict: referenced resource")
var ErrLastAdmin = errors.New("cannot remove the last enabled admin")
