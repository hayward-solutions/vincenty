package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vincenty/api/internal/model"
)

func TestJSON_WritesCorrectHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
}

func TestJSON_EncodesBody(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"key": "value"})

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON body: %v", err)
	}
	if body["key"] != "value" {
		t.Errorf("body[key] = %q, want %q", body["key"], "value")
	}
}

func TestJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusNoContent, nil)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestJSON_CustomStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusCreated, map[string]string{"id": "123"})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "validation_error", "name is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse error body: %v", err)
	}
	if body.Error.Code != "validation_error" {
		t.Errorf("error.code = %q, want %q", body.Error.Code, "validation_error")
	}
	if body.Error.Message != "name is required" {
		t.Errorf("error.message = %q", body.Error.Message)
	}
}

func TestHandleError_ValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	HandleError(w, model.ErrValidation("bad input"))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	assertErrorCode(t, w, "validation_error")
}

func TestHandleError_NotFoundError(t *testing.T) {
	w := httptest.NewRecorder()
	HandleError(w, model.ErrNotFound("user"))

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	assertErrorCode(t, w, "not_found")
}

func TestHandleError_ConflictError(t *testing.T) {
	w := httptest.NewRecorder()
	HandleError(w, model.ErrConflict("duplicate"))

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
	assertErrorCode(t, w, "conflict")
}

func TestHandleError_ForbiddenError(t *testing.T) {
	w := httptest.NewRecorder()
	HandleError(w, model.ErrForbidden("access denied"))

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	assertErrorCode(t, w, "forbidden")
}

func TestHandleError_UnknownError(t *testing.T) {
	w := httptest.NewRecorder()
	HandleError(w, &customError{})

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
	assertErrorCode(t, w, "internal_error")
}

func TestDecode_Valid(t *testing.T) {
	body := `{"username":"admin","password":"testpass"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))

	req, err := Decode[model.LoginRequest](r)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if req.Username != "admin" {
		t.Errorf("Username = %q, want %q", req.Username, "admin")
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("not json"))
	_, err := Decode[model.LoginRequest](r)
	if err == nil {
		t.Error("Decode() expected error for invalid JSON")
	}
	// Should return a ValidationError
	_, ok := err.(*model.ValidationError)
	if !ok {
		t.Errorf("expected *model.ValidationError, got %T", err)
	}
}

func TestDecode_EmptyBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(""))
	_, err := Decode[model.LoginRequest](r)
	if err == nil {
		t.Error("Decode() expected error for empty body")
	}
}

func TestPaginationParams_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/test", nil)
	page, pageSize := PaginationParams(r)

	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize = %d, want 20", pageSize)
	}
}

func TestPaginationParams_CustomValues(t *testing.T) {
	r := httptest.NewRequest("GET", "/test?page=3&page_size=50", nil)
	page, pageSize := PaginationParams(r)

	if page != 3 {
		t.Errorf("page = %d, want 3", page)
	}
	if pageSize != 50 {
		t.Errorf("pageSize = %d, want 50", pageSize)
	}
}

func TestPaginationParams_InvalidValues(t *testing.T) {
	r := httptest.NewRequest("GET", "/test?page=abc&page_size=-1", nil)
	page, pageSize := PaginationParams(r)

	// Should fall back to defaults
	if page != 1 {
		t.Errorf("page = %d, want 1 (default for invalid)", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize = %d, want 20 (default for negative)", pageSize)
	}
}

func TestPaginationParams_ZeroPage(t *testing.T) {
	r := httptest.NewRequest("GET", "/test?page=0", nil)
	page, _ := PaginationParams(r)

	if page != 1 {
		t.Errorf("page = %d, want 1 (default for zero)", page)
	}
}

func TestPaginationParams_ExcessivePageSize(t *testing.T) {
	r := httptest.NewRequest("GET", "/test?page_size=500", nil)
	_, pageSize := PaginationParams(r)

	// Max page_size is 100
	if pageSize != 20 {
		t.Errorf("pageSize = %d, want 20 (default for >100)", pageSize)
	}
}

func TestPaginationParams_MaxPageSize(t *testing.T) {
	r := httptest.NewRequest("GET", "/test?page_size=100", nil)
	_, pageSize := PaginationParams(r)

	if pageSize != 100 {
		t.Errorf("pageSize = %d, want 100", pageSize)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type customError struct{}

func (e *customError) Error() string { return "custom error" }

func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse error body: %v", err)
	}
	if body.Error.Code != expectedCode {
		t.Errorf("error.code = %q, want %q", body.Error.Code, expectedCode)
	}
}
