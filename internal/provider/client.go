// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type bunkerWebClient struct {
	baseURL     *url.URL
	httpClient  *http.Client
	apiToken    string
	apiUsername string
	apiPassword string
}

type bunkerWebAPIError struct {
	StatusCode int
	Message    string
}

func (e *bunkerWebAPIError) Error() string {
	if e == nil {
		return ""
	}

	if e.Message != "" {
		return fmt.Sprintf("bunkerweb api error (%d): %s", e.StatusCode, e.Message)
	}

	return fmt.Sprintf("bunkerweb api error (%d)", e.StatusCode)
}

type bunkerWebService struct {
	ID         string            `json:"id"`
	ServerName string            `json:"server_name"`
	IsDraft    bool              `json:"is_draft"`
	Variables  map[string]string `json:"variables"`
}

type bunkerWebServicesPayload struct {
	Services []bunkerWebService `json:"services"`
}

// bunkerWebServiceConfig is the shape returned by GET /services/{id}: the service
// id as a string plus a flat map of its non-default settings (keys are unprefixed
// when requested with methods=false).
type bunkerWebServiceConfig struct {
	Service string            `json:"service"`
	Config  map[string]string `json:"config"`
}

type bunkerWebInstance struct {
	Hostname    string  `json:"hostname"`
	Name        *string `json:"name,omitempty"`
	Port        *int    `json:"port,omitempty"`
	ListenHTTPS *bool   `json:"listen_https,omitempty"`
	HTTPSPort   *int    `json:"https_port,omitempty"`
	ServerName  *string `json:"server_name,omitempty"`
	Method      *string `json:"method,omitempty"`
}

type bunkerWebInstancePayload struct {
	Instance bunkerWebInstance `json:"instance"`
}

type bunkerWebInstancesPayload struct {
	Instances []bunkerWebInstance `json:"instances"`
}

// bunkerWebGlobalConfigResponse wraps the settings map the API nests under the
// top-level "settings" key on GET /global_config.
type bunkerWebGlobalConfigResponse struct {
	Settings map[string]any `json:"settings"`
}

type bunkerWebConfig struct {
	Service string `json:"service"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Data    string `json:"data,omitempty"`
	Method  string `json:"method,omitempty"`
}

type bunkerWebConfigPayload struct {
	Config bunkerWebConfig `json:"config"`
}

type bunkerWebConfigsPayload struct {
	Configs []bunkerWebConfig `json:"configs"`
}

type bunkerWebBan struct {
	IP      string  `json:"ip"`
	Reason  string  `json:"reason,omitempty"`
	Exp     int     `json:"exp,omitempty"`
	Service *string `json:"service,omitempty"`
}

type bunkerWebBansPayload struct {
	Bans []bunkerWebBan `json:"bans"`
}

type bunkerWebPlugin struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

type bunkerWebPluginsPayload struct {
	Plugins []bunkerWebPlugin `json:"plugins"`
}

type bunkerWebCacheEntry struct {
	Service  string  `json:"service"`
	Plugin   string  `json:"plugin"`
	JobName  string  `json:"job_name"`
	FileName string  `json:"file_name"`
	Data     *string `json:"data,omitempty"`
}

type bunkerWebCacheEntriesPayload struct {
	Cache []bunkerWebCacheEntry `json:"cache"`
}

type bunkerWebJob struct {
	Plugin  string `json:"plugin"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status,omitempty"`
	LastRun string `json:"last_run,omitempty"`
}

type bunkerWebJobsPayload struct {
	Jobs []bunkerWebJob `json:"jobs"`
}

type bunkerWebLoginPayload struct {
	Token string `json:"token"`
}

// bunkerWebAPIResponse captures the envelope fields the BunkerWeb API places at
// the top level of every response body. The actual payload (e.g. "config",
// "service", "instances", "token") sits alongside these fields at the top level
// too, so success bodies are decoded directly into the target struct rather than
// out of a nested "data" object.
type bunkerWebAPIResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail"`
}

func newBunkerWebClient(endpoint string, httpClient *http.Client, token, username, password string) (*bunkerWebClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("api endpoint must be provided")
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse api endpoint: %w", err)
	}

	if parsed.Scheme == "" {
		parsed.Scheme = "https"
	}

	if !strings.HasSuffix(parsed.Path, "/") {
		parsed.Path = parsed.Path + "/"
	}

	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &bunkerWebClient{
		baseURL:     parsed,
		httpClient:  client,
		apiToken:    token,
		apiUsername: username,
		apiPassword: password,
	}, nil
}

func (c *bunkerWebClient) withEndpoint(endpoint string) (string, error) {
	rel, err := url.Parse(strings.TrimPrefix(endpoint, "/"))
	if err != nil {
		return "", err
	}

	resolved := c.baseURL.ResolveReference(rel)
	return resolved.String(), nil
}

func (c *bunkerWebClient) newRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Request, error) {
	var reader io.Reader
	contentType := ""
	if body != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
		reader = buf
		contentType = "application/json"
	}

	return c.newRawRequest(ctx, method, endpoint, reader, contentType)
}

func (c *bunkerWebClient) newRawRequest(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (*http.Request, error) {
	target, err := c.withEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("build request url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set authentication header
	if c.apiToken != "" {
		// Bearer token authentication
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	} else if c.apiUsername != "" && c.apiPassword != "" {
		// HTTP Basic authentication
		credentials := c.apiUsername + ":" + c.apiPassword
		encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", "Basic "+encoded)
	}

	return req, nil
}

func (c *bunkerWebClient) do(ctx context.Context, req *http.Request, out interface{}) error {
	tflog.Debug(ctx, "bunkerweb api request", map[string]any{
		"method": req.Method,
		"url":    req.URL.String(),
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	statusCode := resp.StatusCode
	httpOK := statusCode >= 200 && statusCode < 300

	if len(body) == 0 {
		if httpOK {
			return nil
		}

		return &bunkerWebAPIError{StatusCode: statusCode, Message: strings.TrimSpace(resp.Status)}
	}

	// Best-effort decode of the top-level envelope fields used only for error
	// gating. The BunkerWeb API returns {"status":"success", <named payload>} on
	// success, {"status":"error","message":...} for handler errors, and
	// {"detail":...} for FastAPI built-in errors (e.g. 404/405/422). POST /auth
	// returns {"token":...} with no status field, so an empty status on a 2xx
	// response is treated as success.
	var meta bunkerWebAPIResponse
	_ = json.Unmarshal(body, &meta)

	status := strings.ToLower(strings.TrimSpace(meta.Status))
	statusOK := status == "" || status == "success" || status == "ok"

	if !httpOK || !statusOK {
		msg := firstNonEmpty(
			strings.TrimSpace(meta.Message),
			detailToString(meta.Detail),
			strings.TrimSpace(string(body)),
			strings.TrimSpace(resp.Status),
		)
		return &bunkerWebAPIError{StatusCode: statusCode, Message: msg}
	}

	if out == nil {
		return nil
	}

	// Payload keys live at the top level next to "status", so decode the whole body.
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response payload: %w", err)
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// detailToString renders a FastAPI "detail" field, which is either a string
// (e.g. "Method Not Allowed") or a list of validation error objects.
func detailToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(string(raw))
}

func (c *bunkerWebClient) CreateService(ctx context.Context, reqPayload ServiceCreateRequest) (*bunkerWebService, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "services", reqPayload)
	if err != nil {
		return nil, err
	}

	// The API responds with {"status":"success","changed_plugins":[...]} and no
	// service object. The identifier is the first whitespace token of server_name
	// (matching the API: server_name.split(" ")[0]).
	if err := c.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &bunkerWebService{
		ID:         firstToken(reqPayload.ServerName),
		ServerName: reqPayload.ServerName,
		IsDraft:    reqPayload.IsDraft,
		Variables:  reqPayload.Variables,
	}, nil
}

func (c *bunkerWebClient) GetService(ctx context.Context, id string) (*bunkerWebServiceConfig, error) {
	// methods=false flattens each setting to its string value (the default,
	// methods=true, wraps every value in an object).
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("services", id)+"?methods=false", nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebServiceConfig
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	if payload.Service == "" {
		payload.Service = id
	}

	return &payload, nil
}

func (c *bunkerWebClient) UpdateService(ctx context.Context, id string, reqPayload ServiceUpdateRequest) (*bunkerWebService, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, path.Join("services", id), reqPayload)
	if err != nil {
		return nil, err
	}

	// PATCH returns status only; reconstruct the resulting service from the request.
	if err := c.do(ctx, req, nil); err != nil {
		return nil, err
	}

	svc := &bunkerWebService{ID: id, Variables: reqPayload.Variables}
	if reqPayload.ServerName != nil {
		svc.ServerName = *reqPayload.ServerName
		svc.ID = firstToken(*reqPayload.ServerName) // id follows server_name on rename
	}
	if reqPayload.IsDraft != nil {
		svc.IsDraft = *reqPayload.IsDraft
	}
	return svc, nil
}

func (c *bunkerWebClient) DeleteService(ctx context.Context, id string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("services", id), nil)
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

func (c *bunkerWebClient) ListServices(ctx context.Context, includeDrafts bool) ([]bunkerWebService, error) {
	query := "services"
	if !includeDrafts {
		query = query + "?with_drafts=false"
	}

	req, err := c.newRequest(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebServicesPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Services, nil
}

type ServiceCreateRequest struct {
	ServerName string            `json:"server_name"`
	IsDraft    bool              `json:"is_draft"`
	Variables  map[string]string `json:"variables,omitempty"`
}

type ServiceUpdateRequest struct {
	ServerName *string           `json:"server_name,omitempty"`
	IsDraft    *bool             `json:"is_draft,omitempty"`
	Variables  map[string]string `json:"variables,omitempty"`
}

type InstanceCreateRequest struct {
	Hostname    string  `json:"hostname"`
	Name        *string `json:"name,omitempty"`
	Port        *int    `json:"port,omitempty"`
	ListenHTTPS *bool   `json:"listen_https,omitempty"`
	HTTPSPort   *int    `json:"https_port,omitempty"`
	ServerName  *string `json:"server_name,omitempty"`
	Method      *string `json:"method,omitempty"`
}

type InstanceUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Port        *int    `json:"port,omitempty"`
	ListenHTTPS *bool   `json:"listen_https,omitempty"`
	HTTPSPort   *int    `json:"https_port,omitempty"`
	ServerName  *string `json:"server_name,omitempty"`
	Method      *string `json:"method,omitempty"`
}

type BanRequest struct {
	IP      string  `json:"ip"`
	Exp     *int    `json:"exp,omitempty"`
	Reason  *string `json:"reason,omitempty"`
	Service *string `json:"service,omitempty"`
}

type UnbanRequest struct {
	IP      string  `json:"ip"`
	Service *string `json:"service,omitempty"`
}

type ConfigCreateRequest struct {
	Service *string `json:"service,omitempty"`
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	Data    string  `json:"data"`
}

type ConfigUpdateRequest struct {
	Service *string `json:"service,omitempty"`
	Type    *string `json:"type,omitempty"`
	Name    *string `json:"name,omitempty"`
	Data    *string `json:"data,omitempty"`
}

type ConfigKey struct {
	Service *string `json:"service,omitempty"`
	Type    string  `json:"type"`
	Name    string  `json:"name"`
}

type InstancesDeleteRequest struct {
	Instances []string `json:"instances"`
}

type ConfigsDeleteRequest struct {
	Configs []ConfigKey `json:"configs"`
}

type ConfigUploadFile struct {
	FileName string
	Content  []byte
}

type ConfigUploadRequest struct {
	Service string
	Type    string
	Files   []ConfigUploadFile
}

type ConfigUploadUpdateRequest struct {
	FileName   string
	Content    []byte
	NewService *string
	NewType    *string
	NewName    *string
}

type ConfigListOptions struct {
	Service    *string
	Type       *string
	WithDrafts *bool
	WithData   *bool
}

type PluginUploadFile struct {
	FileName string
	Content  []byte
}

type PluginUploadRequest struct {
	Method string
	Files  []PluginUploadFile
}

type CacheFileKey struct {
	Service  *string `json:"service,omitempty"`
	Plugin   string  `json:"plugin"`
	JobName  string  `json:"job_name"`
	FileName string  `json:"file_name"`
}

type CacheFilesDeleteRequest struct {
	CacheFiles []CacheFileKey `json:"cache_files"`
}

type JobItem struct {
	Plugin string  `json:"plugin"`
	Name   *string `json:"name,omitempty"`
}

type RunJobsRequest struct {
	Jobs []JobItem `json:"jobs"`
}

func (c *bunkerWebClient) GetGlobalConfig(ctx context.Context, full, methods bool) (map[string]any, error) {
	endpoint := "global_config"
	query := url.Values{}
	if full {
		query.Set("full", "true")
	}
	if methods {
		query.Set("methods", "true")
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint = endpoint + "?" + encoded
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebGlobalConfigResponse
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload.Settings), nil
}

func (c *bunkerWebClient) UpdateGlobalConfig(ctx context.Context, settings map[string]any) (map[string]any, error) {
	if len(settings) == 0 {
		return nil, fmt.Errorf("at least one setting must be provided")
	}

	req, err := c.newRequest(ctx, http.MethodPatch, "global_config", settings)
	if err != nil {
		return nil, err
	}

	// PATCH /global_config returns status only; read the settings back so callers
	// can observe the applied values.
	if err := c.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return c.GetGlobalConfig(ctx, true, false)
}

func (c *bunkerWebClient) CreateInstance(ctx context.Context, reqPayload InstanceCreateRequest) (*bunkerWebInstance, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "instances", reqPayload)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebInstancePayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return &payload.Instance, nil
}

func (c *bunkerWebClient) GetInstance(ctx context.Context, hostname string) (*bunkerWebInstance, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path.Join("instances", hostname), nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebInstancePayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return &payload.Instance, nil
}

func (c *bunkerWebClient) UpdateInstance(ctx context.Context, hostname string, reqPayload InstanceUpdateRequest) (*bunkerWebInstance, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, path.Join("instances", hostname), reqPayload)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebInstancePayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return &payload.Instance, nil
}

func (c *bunkerWebClient) DeleteInstance(ctx context.Context, hostname string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("instances", hostname), nil)
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

func (c *bunkerWebClient) DeleteInstances(ctx context.Context, hostnames []string) error {
	if len(hostnames) == 0 {
		return fmt.Errorf("at least one hostname is required")
	}

	reqPayload := InstancesDeleteRequest{Instances: hostnames}
	req, err := c.newRequest(ctx, http.MethodDelete, "instances", reqPayload)
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

func (c *bunkerWebClient) ListInstances(ctx context.Context) ([]bunkerWebInstance, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "instances", nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebInstancesPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Instances, nil
}

func (c *bunkerWebClient) PingInstances(ctx context.Context) (map[string]any, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "instances/ping", nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload), nil
}

func (c *bunkerWebClient) PingInstance(ctx context.Context, hostname string) (map[string]any, error) {
	if strings.TrimSpace(hostname) == "" {
		return nil, fmt.Errorf("hostname must be provided")
	}

	req, err := c.newRequest(ctx, http.MethodGet, path.Join("instances", hostname, "ping"), nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload), nil
}

func (c *bunkerWebClient) ReloadInstances(ctx context.Context, test *bool) (map[string]any, error) {
	endpoint := "instances/reload"
	if test != nil {
		query := url.Values{}
		query.Set("test", strconv.FormatBool(*test))
		endpoint = endpoint + "?" + query.Encode()
	}

	req, err := c.newRequest(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload), nil
}

func (c *bunkerWebClient) ReloadInstance(ctx context.Context, hostname string, test *bool) (map[string]any, error) {
	if strings.TrimSpace(hostname) == "" {
		return nil, fmt.Errorf("hostname must be provided")
	}

	endpoint := path.Join("instances", hostname, "reload")
	if test != nil {
		query := url.Values{}
		query.Set("test", strconv.FormatBool(*test))
		endpoint = endpoint + "?" + query.Encode()
	}

	req, err := c.newRequest(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload), nil
}

func (c *bunkerWebClient) StopInstances(ctx context.Context) (map[string]any, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "instances/stop", nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload), nil
}

func (c *bunkerWebClient) StopInstance(ctx context.Context, hostname string) (map[string]any, error) {
	if strings.TrimSpace(hostname) == "" {
		return nil, fmt.Errorf("hostname must be provided")
	}

	req, err := c.newRequest(ctx, http.MethodPost, path.Join("instances", hostname, "stop"), nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return ensureMap(payload), nil
}

func (c *bunkerWebClient) Ban(ctx context.Context, req BanRequest) error {
	request, err := c.newRequest(ctx, http.MethodPost, "bans", []BanRequest{req})
	if err != nil {
		return err
	}

	return c.do(ctx, request, nil)
}

func (c *bunkerWebClient) Unban(ctx context.Context, req UnbanRequest) error {
	request, err := c.newRequest(ctx, http.MethodDelete, "bans", []UnbanRequest{req})
	if err != nil {
		return err
	}

	return c.do(ctx, request, nil)
}

func (c *bunkerWebClient) ListBans(ctx context.Context) ([]bunkerWebBan, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "bans", nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebBansPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Bans, nil
}

func (c *bunkerWebClient) BanBulk(ctx context.Context, reqs []BanRequest) error {
	if len(reqs) == 0 {
		return fmt.Errorf("at least one ban request is required")
	}

	request, err := c.newRequest(ctx, http.MethodPost, "bans/ban", reqs)
	if err != nil {
		return err
	}

	return c.do(ctx, request, nil)
}

func (c *bunkerWebClient) UnbanBulk(ctx context.Context, reqs []UnbanRequest) error {
	if len(reqs) == 0 {
		return fmt.Errorf("at least one unban request is required")
	}

	request, err := c.newRequest(ctx, http.MethodPost, "bans/unban", reqs)
	if err != nil {
		return err
	}

	return c.do(ctx, request, nil)
}

func (c *bunkerWebClient) CreateConfig(ctx context.Context, input ConfigCreateRequest) (*bunkerWebConfig, error) {
	req, err := c.newRequest(ctx, http.MethodPost, "configs", input)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebConfigPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return &payload.Config, nil
}

func (c *bunkerWebClient) ListConfigs(ctx context.Context, opts ConfigListOptions) ([]bunkerWebConfig, error) {
	query := url.Values{}
	if opts.Service != nil {
		if trimmed := strings.TrimSpace(*opts.Service); trimmed != "" {
			query.Set("service", trimmed)
		}
	}
	if opts.Type != nil {
		if trimmed := strings.TrimSpace(*opts.Type); trimmed != "" {
			query.Set("type", trimmed)
		}
	}
	if opts.WithDrafts != nil {
		query.Set("with_drafts", strconv.FormatBool(*opts.WithDrafts))
	}
	if opts.WithData != nil {
		query.Set("with_data", strconv.FormatBool(*opts.WithData))
	}

	endpoint := "configs"
	if encoded := query.Encode(); encoded != "" {
		endpoint = endpoint + "?" + encoded
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebConfigsPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Configs, nil
}

func (c *bunkerWebClient) GetConfig(ctx context.Context, key ConfigKey, withData bool) (*bunkerWebConfig, error) {
	endpoint := configPath(key)
	if withData {
		endpoint = endpoint + "?with_data=true"
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebConfigPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return &payload.Config, nil
}

func (c *bunkerWebClient) UpdateConfig(ctx context.Context, key ConfigKey, input ConfigUpdateRequest) (*bunkerWebConfig, error) {
	req, err := c.newRequest(ctx, http.MethodPatch, configPath(key), input)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebConfigPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return &payload.Config, nil
}

func (c *bunkerWebClient) DeleteConfig(ctx context.Context, key ConfigKey) error {
	req, err := c.newRequest(ctx, http.MethodDelete, configPath(key), nil)
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

func (c *bunkerWebClient) DeleteConfigs(ctx context.Context, keys []ConfigKey) error {
	if len(keys) == 0 {
		return fmt.Errorf("at least one config key is required")
	}

	reqPayload := ConfigsDeleteRequest{Configs: keys}
	req, err := c.newRequest(ctx, http.MethodDelete, "configs", reqPayload)
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

// bunkerWebUploadResult is the shape returned by the multipart upload endpoints
// (POST /configs/upload, POST /plugins/upload): a list of created identifiers and
// any per-file errors, rather than the created objects themselves.
type bunkerWebUploadResult struct {
	Created []string         `json:"created"`
	Errors  []map[string]any `json:"errors"`
}

func uploadErrorsText(errs []map[string]any) string {
	if len(errs) == 0 {
		return "no error details returned"
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		if msg, ok := e["error"]; ok {
			parts = append(parts, fmt.Sprintf("%v", msg))
		}
	}
	if len(parts) == 0 {
		return "upload failed"
	}
	return strings.Join(parts, "; ")
}

// UploadConfigs uploads custom config files and returns the created config
// identifiers ("service/type/name"); the API does not echo the config objects.
func (c *bunkerWebClient) UploadConfigs(ctx context.Context, input ConfigUploadRequest) ([]string, error) {
	if strings.TrimSpace(input.Type) == "" {
		return nil, fmt.Errorf("type must be provided")
	}
	if len(input.Files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if input.Service != "" {
		if err := writer.WriteField("service", input.Service); err != nil {
			return nil, fmt.Errorf("encode service field: %w", err)
		}
	}
	if err := writer.WriteField("type", input.Type); err != nil {
		return nil, fmt.Errorf("encode type field: %w", err)
	}

	for _, file := range input.Files {
		name := strings.TrimSpace(file.FileName)
		if name == "" {
			return nil, fmt.Errorf("file name must be provided")
		}
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := part.Write(file.Content); err != nil {
			return nil, fmt.Errorf("write file content: %w", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart body: %w", err)
	}

	req, err := c.newRawRequest(ctx, http.MethodPost, "configs/upload", body, contentType)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebUploadResult
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Created, nil
}

func (c *bunkerWebClient) UpdateConfigFromUpload(ctx context.Context, key ConfigKey, input ConfigUploadUpdateRequest) (*bunkerWebConfig, error) {
	name := strings.TrimSpace(input.FileName)
	if name == "" {
		return nil, fmt.Errorf("file name must be provided")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(input.Content); err != nil {
		return nil, fmt.Errorf("write file content: %w", err)
	}

	if input.NewService != nil {
		if err := writer.WriteField("new_service", strings.TrimSpace(*input.NewService)); err != nil {
			return nil, fmt.Errorf("encode new_service field: %w", err)
		}
	}
	if input.NewType != nil {
		if err := writer.WriteField("new_type", strings.TrimSpace(*input.NewType)); err != nil {
			return nil, fmt.Errorf("encode new_type field: %w", err)
		}
	}
	if input.NewName != nil {
		if err := writer.WriteField("new_name", strings.TrimSpace(*input.NewName)); err != nil {
			return nil, fmt.Errorf("encode new_name field: %w", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart body: %w", err)
	}

	endpoint := path.Join(configPath(key), "upload")
	req, err := c.newRawRequest(ctx, http.MethodPatch, endpoint, body, contentType)
	if err != nil {
		return nil, err
	}

	// PATCH .../upload returns only {"status":"success"}; read the (possibly
	// renamed) config back to report its current state.
	if err := c.do(ctx, req, nil); err != nil {
		return nil, err
	}

	effectiveKey := key
	if input.NewService != nil {
		effectiveKey.Service = stringPointer(strings.TrimSpace(*input.NewService))
	}
	if input.NewType != nil && strings.TrimSpace(*input.NewType) != "" {
		effectiveKey.Type = strings.TrimSpace(*input.NewType)
	}
	if input.NewName != nil && strings.TrimSpace(*input.NewName) != "" {
		effectiveKey.Name = strings.TrimSpace(*input.NewName)
	}

	return c.GetConfig(ctx, effectiveKey, true)
}

func (c *bunkerWebClient) ConvertService(ctx context.Context, id string, convertTo string) (*bunkerWebService, error) {
	convertTo = strings.TrimSpace(strings.ToLower(convertTo))
	if convertTo != "online" && convertTo != "draft" {
		return nil, fmt.Errorf("convert_to must be 'online' or 'draft'")
	}

	endpoint := path.Join("services", id, "convert")
	query := url.Values{}
	query.Set("convert_to", convertTo)
	endpoint = endpoint + "?" + query.Encode()

	req, err := c.newRequest(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Convert returns {"status":"success","changed_plugins":[...]} with no service
	// object; derive the resulting draft flag from the requested target state.
	if err := c.do(ctx, req, nil); err != nil {
		return nil, err
	}

	return &bunkerWebService{ID: id, IsDraft: convertTo == "draft"}, nil
}

func ensureMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	return in
}

func (c *bunkerWebClient) ListPlugins(ctx context.Context, pluginType string, withData bool) ([]bunkerWebPlugin, error) {
	query := url.Values{}
	if pluginType != "" {
		query.Set("type", pluginType)
	}
	if withData {
		query.Set("with_data", "true")
	}
	endpoint := "plugins"
	if encoded := query.Encode(); encoded != "" {
		endpoint = endpoint + "?" + encoded
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebPluginsPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Plugins, nil
}

// UploadPlugins uploads plugin archives and returns the created plugin ids; the
// API does not echo plugin objects.
func (c *bunkerWebClient) UploadPlugins(ctx context.Context, input PluginUploadRequest) ([]string, error) {
	if len(input.Files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	method := strings.TrimSpace(input.Method)
	if method != "" {
		if err := writer.WriteField("method", method); err != nil {
			return nil, fmt.Errorf("encode method field: %w", err)
		}
	}

	for _, file := range input.Files {
		name := strings.TrimSpace(file.FileName)
		if name == "" {
			return nil, fmt.Errorf("file name must be provided")
		}
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := part.Write(file.Content); err != nil {
			return nil, fmt.Errorf("write file content: %w", err)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart body: %w", err)
	}

	req, err := c.newRawRequest(ctx, http.MethodPost, "plugins/upload", body, contentType)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebUploadResult
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	if len(payload.Created) == 0 {
		return nil, fmt.Errorf("plugin upload created no plugins: %s", uploadErrorsText(payload.Errors))
	}

	return payload.Created, nil
}

func (c *bunkerWebClient) DeletePlugin(ctx context.Context, pluginID string) error {
	if strings.TrimSpace(pluginID) == "" {
		return fmt.Errorf("plugin id must be provided")
	}

	req, err := c.newRequest(ctx, http.MethodDelete, path.Join("plugins", pluginID), nil)
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

func (c *bunkerWebClient) ListCacheEntries(ctx context.Context, filters url.Values) ([]bunkerWebCacheEntry, error) {
	endpoint := "cache"
	if filters != nil {
		if encoded := filters.Encode(); encoded != "" {
			endpoint = endpoint + "?" + encoded
		}
	}

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebCacheEntriesPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Cache, nil
}

func (c *bunkerWebClient) ListJobs(ctx context.Context) ([]bunkerWebJob, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "jobs", nil)
	if err != nil {
		return nil, err
	}

	var payload bunkerWebJobsPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	return payload.Jobs, nil
}

func (c *bunkerWebClient) RunJobs(ctx context.Context, jobs []JobItem) error {
	if len(jobs) == 0 {
		return fmt.Errorf("at least one job is required")
	}

	req, err := c.newRequest(ctx, http.MethodPost, "jobs/run", RunJobsRequest{Jobs: jobs})
	if err != nil {
		return err
	}

	return c.do(ctx, req, nil)
}

func configPath(key ConfigKey) string {
	svc := "global"
	if key.Service != nil {
		trimmed := strings.TrimSpace(*key.Service)
		if trimmed != "" {
			svc = trimmed
		}
	}

	return path.Join("configs", svc, key.Type, key.Name)
}

func (c *bunkerWebClient) Ping(ctx context.Context) (map[string]any, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "ping", nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	if payload == nil {
		payload = map[string]any{}
	}

	return payload, nil
}

func (c *bunkerWebClient) Health(ctx context.Context) (map[string]any, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "health", nil)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := c.do(ctx, req, &payload); err != nil {
		return nil, err
	}

	if payload == nil {
		payload = map[string]any{}
	}

	return payload, nil
}

func (c *bunkerWebClient) Login(ctx context.Context, username, password string) (string, error) {
	if strings.TrimSpace(username) == "" {
		return "", fmt.Errorf("username must be provided")
	}
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password must be provided")
	}

	body := map[string]string{
		"username": username,
		"password": password,
	}

	req, err := c.newRequest(ctx, http.MethodPost, "auth", body)
	if err != nil {
		return "", err
	}

	credentials := username + ":" + password
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	req.Header.Set("Authorization", "Basic "+encoded)

	var payload bunkerWebLoginPayload
	if err := c.do(ctx, req, &payload); err != nil {
		return "", err
	}

	c.apiToken = payload.Token

	return payload.Token, nil
}
