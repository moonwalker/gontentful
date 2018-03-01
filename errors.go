package gontentful

import (
	"net/http"
	"encoding/json"
)

// ErrorResponse model
type ErrorResponse struct {
	Sys       *Sys          `json:"sys"`
	Message   string        `json:"message,omitempty"`
	RequestID string        `json:"requestId,omitempty"`
	Details   *ErrorDetails `json:"details,omitempty"`
}

func (e ErrorResponse) Error() string {
	return e.Message
}

// ErrorDetails model
type ErrorDetails struct {
	Errors []*ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail model
type ErrorDetail struct {
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name,omitempty"`
	Path    interface{} `json:"path,omitempty"`
	Details string      `json:"details,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

// APIError model
type APIError struct {
	req *http.Request
	res *http.Response
	err *ErrorResponse
}

// AccessTokenInvalidError for 401 errors
type AccessTokenInvalidError struct {
	APIError
}

func (e AccessTokenInvalidError) Error() string {
	return e.APIError.err.Message
}

// VersionMismatchError for 409 errors
type VersionMismatchError struct {
	APIError
}

func (e VersionMismatchError) Error() string {
	return "Version " + e.APIError.req.Header.Get("X-Contentful-Version") + " is mismatched"
}

// ValidationFailedError model
type ValidationFailedError struct {
	APIError
}

func (e ValidationFailedError) Error() string {
	msg := ""
	for _, err := range e.APIError.err.Details.Errors {
		if err.Name == "uniqueFieldIds" || err.Name == "uniqueFieldApiNames" {
			return msg
			// msg += err.Value["id"].(string) + " should be unique for " + err.Value["name"].(string) + "\n"
		}
	}

	return msg
}

// NotFoundError for 404 errors
type NotFoundError struct {
	APIError
}

func (e NotFoundError) Error() string {
	return "the requested resource can not be found"
}

// RateLimitExceededError for rate limit errors
type RateLimitExceededError struct {
	APIError
}

func (e RateLimitExceededError) Error() string {
	return e.APIError.err.Message
}

type BadRequestError struct{}
type InvalidQueryError struct{}
type AccessDeniedError struct{}
type ServerError struct{}

func parseError(req *http.Request, res *http.Response) error {
	var e ErrorResponse
	defer res.Body.Close()
	err := json.NewDecoder(res.Body).Decode(&e)
	if err != nil {
		return err
	}

	apiError := APIError{
		req: req,
		res: res,
		err: &e,
	}

	switch errType := e.Sys.ID; errType {
	case "NotFound":
		return NotFoundError{apiError}
	case "RateLimitExceeded":
		return RateLimitExceededError{apiError}
	case "AccessTokenInvalid":
		return AccessTokenInvalidError{apiError}
	case "ValidationFailed":
		return ValidationFailedError{apiError}
	case "VersionMismatch":
		return VersionMismatchError{apiError}
	case "Conflict":
		return VersionMismatchError{apiError}
	default:
		return e
	}
}
