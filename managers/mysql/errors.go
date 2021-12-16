package mysql

import "errors"

const (
	ERR_NO_SUCH_TABLE = 1146
)

var (
	ErrNoDatabase = errors.New("no database defined")
)
