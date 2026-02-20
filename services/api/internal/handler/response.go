package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/sitaware/api/internal/model"
)

// errorBody is the JSON error envelope.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode response", "error", err)
		}
	}
}

// Error writes a JSON error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{
		Error: errorDetail{Code: code, Message: message},
	})
}

// HandleError maps domain errors to appropriate HTTP responses.
func HandleError(w http.ResponseWriter, err error) {
	var validationErr *model.ValidationError
	var notFoundErr *model.NotFoundError
	var conflictErr *model.ConflictError
	var forbiddenErr *model.ForbiddenError

	switch {
	case errors.As(err, &validationErr):
		Error(w, http.StatusBadRequest, "validation_error", validationErr.Message)
	case errors.As(err, &notFoundErr):
		Error(w, http.StatusNotFound, "not_found", notFoundErr.Error())
	case errors.As(err, &conflictErr):
		Error(w, http.StatusConflict, "conflict", conflictErr.Message)
	case errors.As(err, &forbiddenErr):
		Error(w, http.StatusForbidden, "forbidden", forbiddenErr.Message)
	default:
		slog.Error("unhandled error", "error", err)
		Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// Decode reads and decodes a JSON request body into the target type.
func Decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, model.ErrValidation("invalid request body")
	}
	return v, nil
}

// PaginationParams extracts page and page_size from query parameters.
func PaginationParams(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("page_size"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	return page, pageSize
}
