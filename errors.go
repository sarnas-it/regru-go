package regru

import (
	"encoding/json"
	"fmt"
)

// APIError — ошибка, которую вернул REG.API (result = "error").
// Для программной обработки используйте Code: текст Text может измениться.
type APIError struct {
	Code   string
	Text   string
	Params json.RawMessage
}

func (e *APIError) Error() string {
	if e.Text == "" {
		return "regru: " + e.Code
	}
	return fmt.Sprintf("regru: %s: %s", e.Code, e.Text)
}

// Is позволяет сравнивать ошибки по коду: errors.Is(err, regru.ErrDomainNotFound).
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	return ok && t.Code == e.Code
}

// Часто встречающиеся ошибки для errors.Is.
var (
	ErrPasswordAuthFailed = &APIError{Code: "PASSWORD_AUTH_FAILED"}
	ErrAccessDenied       = &APIError{Code: "ACCESS_DENIED"}
	ErrAccessDeniedFromIP = &APIError{Code: "ACCESS_DENIED_FROM_IP"}
	ErrIPRateLimit        = &APIError{Code: "IP_EXCEEDED_ALLOWED_CONNECTION_RATE"}
	ErrAccountRateLimit   = &APIError{Code: "ACCOUNT_EXCEEDED_ALLOWED_CONNECTION_RATE"}
	ErrDomainNotFound     = &APIError{Code: "DOMAIN_NOT_FOUND"}
	ErrServiceUnavailable = &APIError{Code: "SERVICE_UNAVAILABLE"}
	ErrTooManyObjects     = &APIError{Code: "TOO_MANY_OBJECTS_IN_ONE_REQUEST"}
)

// itemResult — общий хвост элемента в ответах, работающих со списком услуг:
// по каждому элементу приходит свой result и, при ошибке, свой error_code.
type itemResult struct {
	Result    string `json:"result"`
	ErrorCode string `json:"error_code"`
	ErrorText string `json:"error_text"`
}

func (r itemResult) err() error {
	if r.Result == "" || r.Result == "success" {
		return nil
	}
	code := r.ErrorCode
	if code == "" && isErrorCode(r.Result) {
		code = r.Result // в ответах по зоне код ошибки приходит прямо в result
	}
	if code == "" {
		code = "UNKNOWN"
	}
	text := r.ErrorText
	if text == "" {
		text = r.Result
	}
	return &APIError{Code: code, Text: text}
}

// isErrorCode сообщает, похожа ли строка на код ошибки REG.API (A-Z, 0-9, "_").
func isErrorCode(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}
