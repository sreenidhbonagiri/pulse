package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/sreenidhbonagiri/pulse/backend/internal/models"
)

type monitorRequest struct {
	Name                 string `json:"name"`
	URL                  string `json:"url"`
	HTTPMethod           string `json:"http_method"`
	CheckIntervalSeconds int    `json:"check_interval_seconds"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	ExpectedStatusCode   int    `json:"expected_status_code"`
	IsActive             *bool  `json:"is_active"`
}

var allowedHTTPMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodPost:    {},
	http.MethodPut:     {},
	http.MethodPatch:   {},
	http.MethodDelete:  {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

func decodeMonitorRequest(w http.ResponseWriter, r *http.Request) (monitorRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req monitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return monitorRequest{}, errors.New("invalid JSON body")
	}
	return req, nil
}

func (req monitorRequest) validate() error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if err := validateURL(req.URL); err != nil {
		return err
	}
	if err := validateHTTPMethod(req.HTTPMethod); err != nil {
		return err
	}
	if req.CheckIntervalSeconds <= 0 {
		return errors.New("check_interval_seconds must be greater than 0")
	}
	if req.TimeoutSeconds <= 0 {
		return errors.New("timeout_seconds must be greater than 0")
	}
	if req.ExpectedStatusCode != 0 && (req.ExpectedStatusCode < 100 || req.ExpectedStatusCode > 599) {
		return errors.New("expected_status_code must be a valid HTTP status code")
	}
	return nil
}

func validateURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("url is required")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Host == "" {
		return errors.New("url is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url must start with http:// or https://")
	}
	return nil
}

func validateHTTPMethod(method string) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return errors.New("http_method is required")
	}
	if _, ok := allowedHTTPMethods[method]; !ok {
		return errors.New("http_method is invalid")
	}
	return nil
}

func (req monitorRequest) toMonitor() *models.Monitor {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	statusCode := req.ExpectedStatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return &models.Monitor{
		Name:                 strings.TrimSpace(req.Name),
		URL:                  strings.TrimSpace(req.URL),
		HTTPMethod:           strings.ToUpper(strings.TrimSpace(req.HTTPMethod)),
		CheckIntervalSeconds: req.CheckIntervalSeconds,
		TimeoutSeconds:       req.TimeoutSeconds,
		ExpectedStatusCode:   statusCode,
		IsActive:             isActive,
	}
}
