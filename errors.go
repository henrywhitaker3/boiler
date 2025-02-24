package boiler

import "errors"

var (
	ErrUnknownType   = errors.New("unknown type")
	ErrAlreadyExists = errors.New("type already exists in boiler")
	ErrDoesNotExist  = errors.New("type does not exist in boiler")
	ErrWrongType     = errors.New("wrong type resolved from boiler")
	ErrCouldNotMake  = errors.New("could not make service")
)
