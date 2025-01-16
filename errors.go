package dynarag

import "errors"

// ErrMissingConnStr is returned when postgres connection string is not provided
var ErrMissingConnStr = errors.New("postgres connection string is required")
