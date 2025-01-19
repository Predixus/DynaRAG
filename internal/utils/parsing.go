package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

func ParseJsonBody[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var model T

	// confirm content type is json
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		mediaType := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
		if mediaType != "application/json" {
			msg := "Content-Type header is not application/json"
			http.Error(w, msg, http.StatusUnsupportedMediaType)
			return model, errors.New(msg)
		}
	}

	// enforce max size to 1Mb
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	// enforce strict decoding
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&model)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		// catch specific errors and handle them
		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf(
				"Request body contains badly-formed JSON (at position %d)",
				syntaxError.Offset,
			)
			http.Error(w, msg, http.StatusBadRequest)
			return model, errors.New(msg)

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			http.Error(w, msg, http.StatusBadRequest)
			return model, errors.New(msg)

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf(
				"Request body contains an invalid value for the %q field (at position %d)",
				unmarshalTypeError.Field,
				unmarshalTypeError.Offset,
			)
			http.Error(w, msg, http.StatusBadRequest)
			return model, errors.New(msg)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			http.Error(w, msg, http.StatusBadRequest)
			return model, errors.New(msg)

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			http.Error(w, msg, http.StatusBadRequest)
			return model, errors.New(msg)

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			http.Error(w, msg, http.StatusRequestEntityTooLarge)
			return model, errors.New(msg)

		default:
			http.Error(
				w,
				http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError,
			)
			return model, err
		}
	}

	// Check if the decoded struct is empty (all fields are zero values)
	if reflect.ValueOf(model).IsZero() {
		msg := "Request body must not be empty {}"
		http.Error(w, msg, http.StatusBadRequest)
		return model, errors.New(msg)
	}

	// Call decode again to ensure single JSON object
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		msg := "Request body must only contain a single JSON object"
		http.Error(w, msg, http.StatusBadRequest)
		return model, errors.New(msg)
	}

	return model, nil
}
