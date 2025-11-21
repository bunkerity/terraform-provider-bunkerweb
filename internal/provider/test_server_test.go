// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode"
)

type fakeBunkerWebAPI struct {
	t                      *testing.T
	server                 *httptest.Server
	mu                     sync.Mutex
	services               map[string]*bunkerWebService
	instances              map[string]*bunkerWebInstance
	globalConfig           map[string]any
	configs                map[string]*bunkerWebConfig
	bans                   map[string]*bunkerWebBan
	plugins                map[string]*bunkerWebPlugin
	cache                  map[string]*bunkerWebCacheEntry
	jobs                   []bunkerWebJob
	runJobs                []RunJobsRequest
	pingPayload            map[string]any
	healthStatus           map[string]any
	authCreds              map[string]string
	authTokens             map[string]string
	lastAuth               string
	deletedInstanceBatches [][]string
	pingAllCount           int
	pingHosts              []string
	reloadAllTests         []bool
	reloadHostCalls        []instanceActionCall
	stopAllCount           int
	stopHosts              []string
	convertCalls           []serviceConvertCall
	lastGlobalPatch        map[string]any
	deletedConfigBatches   [][]ConfigKey
	createdBanBatches      [][]BanRequest
	deletedBanBatches      [][]UnbanRequest
	uploadedPluginBatches  [][]string
	deletedPlugins         []string
}

type instanceActionCall struct {
	host string
	test bool
}

type serviceConvertCall struct {
	serviceID string
	target    string
}

func newFakeBunkerWebAPI(t *testing.T) *fakeBunkerWebAPI {
	api := &fakeBunkerWebAPI{
		t:            t,
		services:     make(map[string]*bunkerWebService),
		instances:    make(map[string]*bunkerWebInstance),
		globalConfig: map[string]any{"some_setting": "value", "feature_enabled": true, "retry_limit": 5},
		configs:      make(map[string]*bunkerWebConfig),
		bans:         make(map[string]*bunkerWebBan),
		plugins: map[string]*bunkerWebPlugin{
			"ui-dashboard": {ID: "ui-dashboard", Type: "ui", Version: "1.0.0", Description: "Dashboard"},
		},
		cache: map[string]*bunkerWebCacheEntry{
			"global|reporter|daily|summary.txt": {
				Service:  "global",
				Plugin:   "reporter",
				JobName:  "daily",
				FileName: "summary.txt",
				Data:     ptr("compressed content"),
			},
		},
		jobs: []bunkerWebJob{
			{Plugin: "reporter", Name: "daily", Status: "idle"},
		},
		pingPayload:  map[string]any{"pong": true, "now": "2024-01-01T00:00:00Z"},
		healthStatus: map[string]any{"status": "healthy", "uptime_seconds": 1234},
		authCreds: map[string]string{
			"admin": "secret",
		},
		authTokens: make(map[string]string),
	}

	api.server = httptest.NewServer(http.HandlerFunc(api.handle))
	t.Cleanup(api.server.Close)

	return api
}

func (f *fakeBunkerWebAPI) URL() string {
	return f.server.URL
}

func (f *fakeBunkerWebAPI) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/ping":
		f.handlePing(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		f.handleHealth(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/auth":
		f.handleLogin(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/services":
		f.handleCreateService(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/services":
		f.handleListServices(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/services/") && strings.HasSuffix(r.URL.Path, "/convert"):
		f.handleConvertService(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/services/"):
		f.handleGetService(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/services/"):
		f.handleUpdateService(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/services/"):
		f.handleDeleteService(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/instances":
		f.handleCreateInstance(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/instances":
		f.handleListInstances(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/instances":
		f.handleDeleteInstances(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/instances/ping":
		f.handlePingInstances(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/instances/reload":
		f.handleReloadInstances(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/instances/stop":
		f.handleStopInstances(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/instances/"):
		f.routeInstanceGet(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/instances/"):
		f.routeInstancePost(w, r)
	case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/instances/"):
		f.handleUpdateInstance(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/instances/"):
		f.handleDeleteInstance(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/global_config":
		f.handleGetGlobalConfig(w, r)
	case r.Method == http.MethodPatch && r.URL.Path == "/global_config":
		f.handlePatchGlobalConfig(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/configs":
		f.handleListConfigs(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/configs":
		f.handleCreateConfig(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/configs":
		f.handleDeleteConfigs(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/configs/upload":
		f.handleUploadConfigs(w, r)
	case strings.HasPrefix(r.URL.Path, "/configs/") && strings.HasSuffix(r.URL.Path, "/upload") && r.Method == http.MethodPatch:
		f.handleUploadConfigUpdate(w, r)
	case strings.HasPrefix(r.URL.Path, "/configs/") && r.Method == http.MethodGet:
		f.handleGetConfig(w, r)
	case strings.HasPrefix(r.URL.Path, "/configs/") && r.Method == http.MethodPatch:
		f.handleUpdateConfig(w, r)
	case strings.HasPrefix(r.URL.Path, "/configs/") && r.Method == http.MethodDelete:
		f.handleDeleteConfig(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/bans":
		f.handleListBans(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/bans":
		f.handleCreateBan(w, r)
	case r.Method == http.MethodDelete && r.URL.Path == "/bans":
		f.handleDeleteBan(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/bans/ban":
		f.handleCreateBan(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/bans/unban":
		f.handlePostUnban(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/plugins":
		f.handleListPlugins(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/plugins/upload":
		f.handleUploadPlugins(w, r)
	case strings.HasPrefix(r.URL.Path, "/plugins/") && r.Method == http.MethodDelete:
		f.handleDeletePlugin(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/cache":
		f.handleListCache(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/jobs":
		f.handleListJobs(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/jobs/run":
		f.handleRunJobs(w, r)
	default:
		f.writeError(w, http.StatusNotFound, "not found")
	}
}

func (f *fakeBunkerWebAPI) handlePing(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	payload := cloneAnyMap(f.pingPayload)
	f.mu.Unlock()
	f.writeSuccess(w, payload)
}

func (f *fakeBunkerWebAPI) handleHealth(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	payload := cloneAnyMap(f.healthStatus)
	f.mu.Unlock()
	f.writeSuccess(w, payload)
}

func (f *fakeBunkerWebAPI) handleLogin(w http.ResponseWriter, r *http.Request) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))

	f.mu.Lock()
	f.lastAuth = authHeader
	f.mu.Unlock()

	username, password, err := f.extractCredentials(r, authHeader)
	if err != nil {
		f.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if username == "" || password == "" {
		f.writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	f.mu.Lock()
	expected, ok := f.authCreds[username]
	if !ok || expected != password {
		f.mu.Unlock()
		f.writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token := fmt.Sprintf("token-%s", username)
	f.authTokens[username] = token
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebLoginPayload{Token: token})
}

func (f *fakeBunkerWebAPI) extractCredentials(r *http.Request, authHeader string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(authHeader), "basic ") {
		encoded := strings.TrimSpace(authHeader[6:])
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", "", fmt.Errorf("invalid basic auth header")
		}
		parts := strings.SplitN(string(raw), ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid basic auth header")
		}
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
	}

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", "", fmt.Errorf("invalid request body")
	}
	return strings.TrimSpace(body.Username), strings.TrimSpace(body.Password), nil
}

func (f *fakeBunkerWebAPI) LastAuthorization() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastAuth
}

func (f *fakeBunkerWebAPI) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var req ServiceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ServerName) == "" {
		f.writeError(w, http.StatusBadRequest, "server_name required")
		return
	}

	id := deriveServiceIdentifier(req.ServerName)
	svc := &bunkerWebService{
		ID:         id,
		ServerName: req.ServerName,
		IsDraft:    req.IsDraft,
		Variables:  cloneStringMap(req.Variables),
	}

	f.mu.Lock()
	f.services[id] = svc
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebServicePayload{Service: *svc})
}

func (f *fakeBunkerWebAPI) handleListServices(w http.ResponseWriter, r *http.Request) {
	includeDrafts := true
	if withDrafts := strings.TrimSpace(r.URL.Query().Get("with_drafts")); withDrafts != "" {
		parsed, err := strconv.ParseBool(withDrafts)
		if err == nil {
			includeDrafts = parsed
		}
	}

	f.mu.Lock()
	services := make([]bunkerWebService, 0, len(f.services))
	for _, svc := range f.services {
		if !includeDrafts && svc.IsDraft {
			continue
		}
		services = append(services, *svc)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebServicesPayload{Services: services})
}

func (f *fakeBunkerWebAPI) handleGetService(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/services/")
	id = strings.Trim(id, "/")

	f.mu.Lock()
	svc, ok := f.services[id]
	f.mu.Unlock()

	if !ok {
		f.writeError(w, http.StatusNotFound, "service not found")
		return
	}

	f.writeSuccess(w, bunkerWebServicePayload{Service: *svc})
}

func (f *fakeBunkerWebAPI) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/services/")
	id = strings.Trim(id, "/")

	var req ServiceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f.mu.Lock()
	svc, ok := f.services[id]
	if !ok {
		f.mu.Unlock()
		f.writeError(w, http.StatusNotFound, "service not found")
		return
	}

	if req.ServerName != nil {
		svc.ServerName = *req.ServerName
	}
	if req.IsDraft != nil {
		svc.IsDraft = *req.IsDraft
	}
	if req.Variables != nil {
		svc.Variables = cloneStringMap(req.Variables)
	}

	if req.ServerName != nil {
		newID := deriveServiceIdentifier(*req.ServerName)
		if newID != id {
			delete(f.services, id)
			svc.ID = newID
			f.services[newID] = svc
		}
	}

	updated := *svc
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebServicePayload{Service: updated})
}

func (f *fakeBunkerWebAPI) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/services/")
	id = strings.Trim(id, "/")

	f.mu.Lock()
	if _, ok := f.services[id]; !ok {
		f.mu.Unlock()
		f.writeError(w, http.StatusNotFound, "service not found")
		return
	}
	delete(f.services, id)
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleConvertService(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/services/"), "/convert")
	serviceID := strings.Trim(trimmed, "/")
	convertTo := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("convert_to")))
	if convertTo == "" {
		f.writeError(w, http.StatusBadRequest, "convert_to is required")
		return
	}
	if convertTo != "online" && convertTo != "draft" {
		f.writeError(w, http.StatusBadRequest, "invalid convert_to value")
		return
	}

	f.mu.Lock()
	svc, ok := f.services[serviceID]
	if !ok {
		f.mu.Unlock()
		f.writeError(w, http.StatusNotFound, "service not found")
		return
	}
	svc.IsDraft = convertTo == "draft"
	updated := *svc
	f.convertCalls = append(f.convertCalls, serviceConvertCall{serviceID: serviceID, target: convertTo})
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebServicePayload{Service: updated})
}

func (f *fakeBunkerWebAPI) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	var req InstanceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Hostname) == "" {
		f.writeError(w, http.StatusBadRequest, "hostname required")
		return
	}

	inst := &bunkerWebInstance{Hostname: req.Hostname}
	if req.Name != nil {
		name := *req.Name
		inst.Name = &name
	}
	if req.Port != nil {
		port := *req.Port
		inst.Port = &port
	}
	if req.ListenHTTPS != nil {
		val := *req.ListenHTTPS
		inst.ListenHTTPS = &val
	}
	if req.HTTPSPort != nil {
		port := *req.HTTPSPort
		inst.HTTPSPort = &port
	}
	if req.ServerName != nil {
		server := *req.ServerName
		inst.ServerName = &server
	}
	if req.Method != nil {
		method := *req.Method
		inst.Method = &method
	}

	f.mu.Lock()
	f.instances[inst.Hostname] = inst
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebInstancePayload{Instance: *inst})
}

func (f *fakeBunkerWebAPI) handleListInstances(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	instances := make([]bunkerWebInstance, 0, len(f.instances))
	for _, inst := range f.instances {
		instances = append(instances, *inst)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebInstancesPayload{Instances: instances})
}

func (f *fakeBunkerWebAPI) handleDeleteInstances(w http.ResponseWriter, r *http.Request) {
	var req InstancesDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Instances) == 0 {
		f.writeError(w, http.StatusBadRequest, "no instances provided")
		return
	}

	batch := make([]string, 0, len(req.Instances))

	f.mu.Lock()
	for _, h := range req.Instances {
		hostname := strings.TrimSpace(h)
		if hostname == "" {
			continue
		}
		delete(f.instances, hostname)
		batch = append(batch, hostname)
	}
	if len(batch) > 0 {
		copyBatch := make([]string, len(batch))
		copy(copyBatch, batch)
		f.deletedInstanceBatches = append(f.deletedInstanceBatches, copyBatch)
	}
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handlePingInstances(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	f.pingAllCount++
	count := f.pingAllCount
	total := len(f.instances)
	f.mu.Unlock()

	f.writeSuccess(w, map[string]any{"pinged": total, "count": count})
}

func (f *fakeBunkerWebAPI) routeInstanceGet(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/ping"):
		f.handlePingInstance(w, r)
	case strings.HasSuffix(r.URL.Path, "/reload"):
		f.handleReloadInstance(w, r)
	case strings.HasSuffix(r.URL.Path, "/stop"):
		f.handleStopInstance(w, r)
	default:
		f.handleGetInstance(w, r)
	}
}

func (f *fakeBunkerWebAPI) routeInstancePost(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/reload"):
		f.handleReloadInstance(w, r)
	case strings.HasSuffix(r.URL.Path, "/stop"):
		f.handleStopInstance(w, r)
	default:
		f.writeError(w, http.StatusNotFound, "not found")
	}
}

func (f *fakeBunkerWebAPI) handlePingInstance(w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/instances/"), "/ping")
	hostname = strings.Trim(hostname, "/")

	f.mu.Lock()
	_, ok := f.instances[hostname]
	if ok {
		f.pingHosts = append(f.pingHosts, hostname)
	}
	f.mu.Unlock()

	if !ok {
		f.writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	f.writeSuccess(w, map[string]any{"host": hostname, "pong": true})
}

func (f *fakeBunkerWebAPI) handleReloadInstances(w http.ResponseWriter, r *http.Request) {
	testFlag := true
	if raw := r.URL.Query().Get("test"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err == nil {
			testFlag = parsed
		}
	}

	f.mu.Lock()
	f.reloadAllTests = append(f.reloadAllTests, testFlag)
	f.mu.Unlock()

	f.writeSuccess(w, map[string]any{"reload": "all", "test": testFlag})
}

func (f *fakeBunkerWebAPI) handleReloadInstance(w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/instances/"), "/reload")
	hostname = strings.Trim(hostname, "/")
	testFlag := true
	if raw := r.URL.Query().Get("test"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err == nil {
			testFlag = parsed
		}
	}

	f.mu.Lock()
	_, ok := f.instances[hostname]
	if ok {
		f.reloadHostCalls = append(f.reloadHostCalls, instanceActionCall{host: hostname, test: testFlag})
	}
	f.mu.Unlock()

	if !ok {
		f.writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	f.writeSuccess(w, map[string]any{"host": hostname, "test": testFlag})
}

func (f *fakeBunkerWebAPI) handleStopInstances(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	f.stopAllCount++
	f.mu.Unlock()

	f.writeSuccess(w, map[string]any{"stopped": "all"})
}

func (f *fakeBunkerWebAPI) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/instances/"), "/stop")
	hostname = strings.Trim(hostname, "/")

	f.mu.Lock()
	inst, ok := f.instances[hostname]
	if ok {
		f.stopHosts = append(f.stopHosts, hostname)
	}
	f.mu.Unlock()

	if !ok {
		f.writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	f.writeSuccess(w, bunkerWebInstancePayload{Instance: *inst})
}

func (f *fakeBunkerWebAPI) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimPrefix(r.URL.Path, "/instances/")
	hostname = strings.Trim(hostname, "/")

	f.mu.Lock()
	inst, ok := f.instances[hostname]
	f.mu.Unlock()

	if !ok {
		f.writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	f.writeSuccess(w, bunkerWebInstancePayload{Instance: *inst})
}

func (f *fakeBunkerWebAPI) handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimPrefix(r.URL.Path, "/instances/")
	hostname = strings.Trim(hostname, "/")

	var req InstanceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f.mu.Lock()
	inst, ok := f.instances[hostname]
	if !ok {
		f.mu.Unlock()
		f.writeError(w, http.StatusNotFound, "instance not found")
		return
	}

	if req.Name != nil {
		name := *req.Name
		inst.Name = &name
	}
	if req.Port != nil {
		port := *req.Port
		inst.Port = &port
	}
	if req.ListenHTTPS != nil {
		val := *req.ListenHTTPS
		inst.ListenHTTPS = &val
	}
	if req.HTTPSPort != nil {
		port := *req.HTTPSPort
		inst.HTTPSPort = &port
	}
	if req.ServerName != nil {
		server := *req.ServerName
		inst.ServerName = &server
	}
	if req.Method != nil {
		method := *req.Method
		inst.Method = &method
	}

	updated := *inst
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebInstancePayload{Instance: updated})
}

func (f *fakeBunkerWebAPI) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimPrefix(r.URL.Path, "/instances/")
	hostname = strings.Trim(hostname, "/")

	f.mu.Lock()
	delete(f.instances, hostname)
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleGetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	includeMethods := r.URL.Query().Get("methods") == "true"

	f.mu.Lock()
	configCopy := make(map[string]any, len(f.globalConfig))
	for k, v := range f.globalConfig {
		configCopy[k] = v
	}
	f.mu.Unlock()

	if includeMethods {
		configCopy["__methods__"] = map[string]string{"example": "patch"}
	}

	f.writeSuccess(w, configCopy)
}

func (f *fakeBunkerWebAPI) handlePatchGlobalConfig(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(payload) == 0 {
		f.writeError(w, http.StatusBadRequest, "no settings provided")
		return
	}

	f.mu.Lock()
	for k, v := range payload {
		if v == nil {
			delete(f.globalConfig, k)
		} else {
			f.globalConfig[k] = v
		}
	}
	f.lastGlobalPatch = cloneAnyMap(payload)
	updated := make(map[string]any, len(f.globalConfig))
	for k, v := range f.globalConfig {
		updated[k] = v
	}
	f.mu.Unlock()

	f.writeSuccess(w, updated)
}

func (f *fakeBunkerWebAPI) handleCreateConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Name) == "" {
		f.writeError(w, http.StatusBadRequest, "type and name required")
		return
	}

	service := normalizeConfigService(req.Service)
	key := configStorageKey(service, req.Type, req.Name)

	f.mu.Lock()
	cfg := &bunkerWebConfig{Service: service, Type: req.Type, Name: req.Name, Data: req.Data, Method: "api"}
	f.configs[key] = cfg
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebConfigPayload{Config: *cfg})
}

func (f *fakeBunkerWebAPI) handleListConfigs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filterService := strings.TrimSpace(query.Get("service"))
	filterType := strings.TrimSpace(query.Get("type"))
	withData := query.Get("with_data") == "true"

	f.mu.Lock()
	configs := make([]bunkerWebConfig, 0, len(f.configs))
	for _, cfg := range f.configs {
		if filterService != "" && cfg.Service != filterService {
			continue
		}
		if filterType != "" && cfg.Type != filterType {
			continue
		}
		copyCfg := *cfg
		if !withData {
			copyCfg.Data = ""
		}
		configs = append(configs, copyCfg)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebConfigsPayload{Configs: configs})
}

func (f *fakeBunkerWebAPI) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	service, cfgType, name, err := parseConfigPathParts(r.URL.Path)
	if err != nil {
		f.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	f.mu.Lock()
	cfg, ok := f.configs[configStorageKey(service, cfgType, name)]
	f.mu.Unlock()

	if !ok {
		f.writeError(w, http.StatusNotFound, "config not found")
		return
	}

	includeData := r.URL.Query().Get("with_data") == "true"
	resp := *cfg
	if !includeData {
		resp.Data = ""
	}

	f.writeSuccess(w, bunkerWebConfigPayload{Config: resp})
}

func (f *fakeBunkerWebAPI) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	service, cfgType, name, err := parseConfigPathParts(r.URL.Path)
	if err != nil {
		f.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	key := configStorageKey(service, cfgType, name)

	f.mu.Lock()
	cfg, ok := f.configs[key]
	if !ok {
		f.mu.Unlock()
		f.writeError(w, http.StatusNotFound, "config not found")
		return
	}

	if req.Data != nil {
		cfg.Data = *req.Data
	}

	newService := service
	if req.Service != nil {
		newService = normalizeConfigService(req.Service)
	}
	newType := cfgType
	if req.Type != nil && strings.TrimSpace(*req.Type) != "" {
		newType = strings.TrimSpace(*req.Type)
	}
	newName := name
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		newName = strings.TrimSpace(*req.Name)
	}

	if newService != service || newType != cfgType || newName != name {
		delete(f.configs, key)
		cfg.Service = newService
		cfg.Type = newType
		cfg.Name = newName
		key = configStorageKey(newService, newType, newName)
		f.configs[key] = cfg
	}

	updated := *cfg
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebConfigPayload{Config: updated})
}

func (f *fakeBunkerWebAPI) handleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	service, cfgType, name, err := parseConfigPathParts(r.URL.Path)
	if err != nil {
		f.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	f.mu.Lock()
	delete(f.configs, configStorageKey(service, cfgType, name))
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleDeleteConfigs(w http.ResponseWriter, r *http.Request) {
	var req ConfigsDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Configs) == 0 {
		f.writeError(w, http.StatusBadRequest, "no configs provided")
		return
	}

	batch := make([]ConfigKey, 0, len(req.Configs))

	f.mu.Lock()
	for _, key := range enumerateConfigKeys(req.Configs) {
		service := normalizeConfigService(key.Service)
		storeKey := configStorageKey(service, key.Type, key.Name)
		delete(f.configs, storeKey)
		var servicePtr *string
		if service != "global" {
			svcCopy := service
			servicePtr = &svcCopy
		}
		batch = append(batch, ConfigKey{Service: servicePtr, Type: key.Type, Name: key.Name})
	}
	if len(batch) > 0 {
		f.deletedConfigBatches = append(f.deletedConfigBatches, batch)
	}
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleUploadConfigs(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		f.writeError(w, http.StatusBadRequest, "missing files part")
		return
	}

	cfgType := strings.TrimSpace(r.FormValue("type"))
	if cfgType == "" {
		f.writeError(w, http.StatusBadRequest, "type field required")
		return
	}
	service := normalizeConfigService(optionalStringPointer(r.FormValue("service")))

	created := make([]bunkerWebConfig, 0, len(files))

	f.mu.Lock()
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			f.mu.Unlock()
			f.writeError(w, http.StatusBadRequest, "unable to read uploaded file")
			return
		}
		content, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			f.mu.Unlock()
			f.writeError(w, http.StatusBadRequest, "unable to read uploaded file")
			return
		}

		name := sanitizeConfigFileName(fh.Filename)
		key := configStorageKey(service, cfgType, name)
		cfg := &bunkerWebConfig{Service: service, Type: cfgType, Name: name, Data: string(content), Method: "api"}
		f.configs[key] = cfg
		created = append(created, *cfg)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebConfigsPayload{Configs: created})
}

func (f *fakeBunkerWebAPI) handleUploadConfigUpdate(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimSuffix(r.URL.Path, "/upload")
	service, cfgType, name, err := parseConfigPathParts(trimmed)
	if err != nil {
		f.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		f.writeError(w, http.StatusBadRequest, "missing file part")
		return
	}

	fh := files[0]
	file, err := fh.Open()
	if err != nil {
		f.writeError(w, http.StatusBadRequest, "unable to read uploaded file")
		return
	}
	content, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		f.writeError(w, http.StatusBadRequest, "unable to read uploaded file")
		return
	}

	newService := normalizeConfigService(optionalStringPointer(r.FormValue("new_service")))
	if newService == "" {
		newService = service
	}
	newType := strings.TrimSpace(r.FormValue("new_type"))
	if newType == "" {
		newType = cfgType
	}
	newName := strings.TrimSpace(r.FormValue("new_name"))
	if newName == "" {
		newName = name
	}
	newName = sanitizeConfigFileName(newName)

	originalKey := configStorageKey(service, cfgType, name)
	newKey := configStorageKey(newService, newType, newName)

	f.mu.Lock()
	cfg, ok := f.configs[originalKey]
	if !ok {
		cfg = &bunkerWebConfig{Service: service, Type: cfgType, Name: name, Method: "api"}
	}
	cfg.Service = newService
	cfg.Type = newType
	cfg.Name = newName
	cfg.Data = string(content)
	f.configs[newKey] = cfg
	if newKey != originalKey {
		delete(f.configs, originalKey)
	}
	updated := *cfg
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebConfigPayload{Config: updated})
}

func (f *fakeBunkerWebAPI) handleListBans(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	bans := make([]bunkerWebBan, 0, len(f.bans))
	for _, ban := range f.bans {
		bans = append(bans, *ban)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebBansPayload{Bans: bans})
}

func (f *fakeBunkerWebAPI) handleCreateBan(w http.ResponseWriter, r *http.Request) {
	reqs, err := decodeBanRequests(r.Body)
	if err != nil {
		f.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(reqs) == 0 {
		f.writeError(w, http.StatusBadRequest, "no ban requests provided")
		return
	}

	batch := make([]BanRequest, 0, len(reqs))

	f.mu.Lock()
	for _, req := range reqs {
		ip := strings.TrimSpace(req.IP)
		if ip == "" {
			continue
		}
		reason := "api"
		if req.Reason != nil && strings.TrimSpace(*req.Reason) != "" {
			reason = strings.TrimSpace(*req.Reason)
		}
		exp := 0
		if req.Exp != nil {
			exp = *req.Exp
		}
		service := normalizeBanService(req.Service)
		storedService := &service
		if service == "" {
			storedService = nil
		}
		f.bans[banStorageKey(ip, optionalStringPointer(service))] = &bunkerWebBan{IP: ip, Reason: reason, Exp: exp, Service: storedService}

		expCopy := exp
		reasonCopy := reason
		copyReq := BanRequest{IP: ip, Exp: &expCopy, Reason: &reasonCopy}
		if service != "" {
			svcCopy := service
			copyReq.Service = &svcCopy
		}
		batch = append(batch, copyReq)
	}
	if len(batch) > 0 {
		f.createdBanBatches = append(f.createdBanBatches, batch)
	}
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleDeleteBan(w http.ResponseWriter, r *http.Request) {
	var req []UnbanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f.processUnbanRequests(w, req)
}

func (f *fakeBunkerWebAPI) handlePostUnban(w http.ResponseWriter, r *http.Request) {
	var req []UnbanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f.processUnbanRequests(w, req)
}

func (f *fakeBunkerWebAPI) processUnbanRequests(w http.ResponseWriter, req []UnbanRequest) {
	batch := make([]UnbanRequest, 0, len(req))

	f.mu.Lock()
	for _, item := range req {
		ip := strings.TrimSpace(item.IP)
		if ip == "" {
			continue
		}
		service := normalizeBanService(item.Service)
		delete(f.bans, banStorageKey(ip, optionalStringPointer(service)))
		copyReq := UnbanRequest{IP: ip}
		if service != "" {
			svcCopy := service
			copyReq.Service = &svcCopy
		}
		batch = append(batch, copyReq)
	}
	if len(batch) > 0 {
		f.deletedBanBatches = append(f.deletedBanBatches, batch)
	}
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	filterType := strings.TrimSpace(r.URL.Query().Get("type"))

	f.mu.Lock()
	plugins := make([]bunkerWebPlugin, 0, len(f.plugins))
	for _, plugin := range f.plugins {
		if filterType != "" && filterType != "all" && plugin.Type != filterType {
			continue
		}
		plugins = append(plugins, *plugin)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebPluginsPayload{Plugins: plugins})
}

func (f *fakeBunkerWebAPI) handleUploadPlugins(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		f.writeError(w, http.StatusBadRequest, "missing files part")
		return
	}

	method := strings.TrimSpace(r.FormValue("method"))
	if method == "" {
		method = "ui"
	}

	ids := make([]string, 0, len(files))
	created := make([]bunkerWebPlugin, 0, len(files))

	f.mu.Lock()
	for _, fh := range files {
		file, err := fh.Open()
		if err != nil {
			f.mu.Unlock()
			f.writeError(w, http.StatusBadRequest, "unable to read uploaded file")
			return
		}
		_, _ = io.Copy(io.Discard, file)
		_ = file.Close()

		base := filepath.Base(fh.Filename)
		id := strings.TrimSuffix(base, filepath.Ext(base))
		if id == "" {
			id = base
		}
		plugin := &bunkerWebPlugin{
			ID:          id,
			Type:        method,
			Version:     "uploaded",
			Description: fmt.Sprintf("uploaded from %s", fh.Filename),
		}
		f.plugins[id] = plugin
		ids = append(ids, id)
		created = append(created, *plugin)
	}
	if len(ids) > 0 {
		copyBatch := make([]string, len(ids))
		copy(copyBatch, ids)
		f.uploadedPluginBatches = append(f.uploadedPluginBatches, copyBatch)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebPluginsPayload{Plugins: created})
}

func (f *fakeBunkerWebAPI) handleDeletePlugin(w http.ResponseWriter, r *http.Request) {
	pluginID := strings.TrimPrefix(r.URL.Path, "/plugins/")
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		f.writeError(w, http.StatusBadRequest, "plugin id required")
		return
	}

	f.mu.Lock()
	if _, ok := f.plugins[pluginID]; !ok {
		f.mu.Unlock()
		f.writeError(w, http.StatusNotFound, "plugin not found")
		return
	}
	delete(f.plugins, pluginID)
	f.deletedPlugins = append(f.deletedPlugins, pluginID)
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) handleListCache(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filterService := strings.TrimSpace(query.Get("service"))
	filterPlugin := strings.TrimSpace(query.Get("plugin"))
	filterJob := strings.TrimSpace(query.Get("job_name"))
	withData := query.Get("with_data") == "true"

	f.mu.Lock()
	cacheEntries := make([]bunkerWebCacheEntry, 0, len(f.cache))
	for _, entry := range f.cache {
		if filterService != "" && entry.Service != filterService {
			continue
		}
		if filterPlugin != "" && entry.Plugin != filterPlugin {
			continue
		}
		if filterJob != "" && entry.JobName != filterJob {
			continue
		}
		copyEntry := *entry
		if !withData {
			copyEntry.Data = nil
		}
		cacheEntries = append(cacheEntries, copyEntry)
	}
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebCacheEntriesPayload{Cache: cacheEntries})
}

func (f *fakeBunkerWebAPI) handleListJobs(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	jobs := make([]bunkerWebJob, len(f.jobs))
	copy(jobs, f.jobs)
	f.mu.Unlock()

	f.writeSuccess(w, bunkerWebJobsPayload{Jobs: jobs})
}

func (f *fakeBunkerWebAPI) handleRunJobs(w http.ResponseWriter, r *http.Request) {
	var req RunJobsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Jobs) == 0 {
		f.writeError(w, http.StatusBadRequest, "at least one job required")
		return
	}

	f.mu.Lock()
	f.runJobs = append(f.runJobs, req)
	f.mu.Unlock()

	f.writeSuccess(w, struct{}{})
}

func (f *fakeBunkerWebAPI) DeletedInstanceBatches() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([][]string, len(f.deletedInstanceBatches))
	for i, batch := range f.deletedInstanceBatches {
		copyBatch := make([]string, len(batch))
		copy(copyBatch, batch)
		result[i] = copyBatch
	}
	return result
}

func (f *fakeBunkerWebAPI) PingHosts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.pingHosts))
	copy(result, f.pingHosts)
	return result
}

func (f *fakeBunkerWebAPI) ReloadAllTests() []bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]bool, len(f.reloadAllTests))
	copy(result, f.reloadAllTests)
	return result
}

func (f *fakeBunkerWebAPI) ReloadHostCalls() []instanceActionCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]instanceActionCall, len(f.reloadHostCalls))
	copy(result, f.reloadHostCalls)
	return result
}

func (f *fakeBunkerWebAPI) StopAllCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopAllCount
}

func (f *fakeBunkerWebAPI) StopHosts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.stopHosts))
	copy(result, f.stopHosts)
	return result
}

func (f *fakeBunkerWebAPI) ConvertCalls() []serviceConvertCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]serviceConvertCall, len(f.convertCalls))
	copy(result, f.convertCalls)
	return result
}

func (f *fakeBunkerWebAPI) LastGlobalPatch() map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneAnyMap(f.lastGlobalPatch)
}

func (f *fakeBunkerWebAPI) DeletedConfigBatches() [][]ConfigKey {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([][]ConfigKey, len(f.deletedConfigBatches))
	for i, batch := range f.deletedConfigBatches {
		copyBatch := make([]ConfigKey, len(batch))
		copy(copyBatch, batch)
		result[i] = copyBatch
	}
	return result

}

func (f *fakeBunkerWebAPI) CreatedBanBatches() [][]BanRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([][]BanRequest, len(f.createdBanBatches))
	for i, batch := range f.createdBanBatches {
		copyBatch := make([]BanRequest, len(batch))
		copy(copyBatch, batch)
		result[i] = copyBatch
	}
	return result
}

func (f *fakeBunkerWebAPI) DeletedBanBatches() [][]UnbanRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([][]UnbanRequest, len(f.deletedBanBatches))
	for i, batch := range f.deletedBanBatches {
		copyBatch := make([]UnbanRequest, len(batch))
		copy(copyBatch, batch)
		result[i] = copyBatch
	}
	return result
}

func (f *fakeBunkerWebAPI) UploadedPluginBatches() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([][]string, len(f.uploadedPluginBatches))
	for i, batch := range f.uploadedPluginBatches {
		copyBatch := make([]string, len(batch))
		copy(copyBatch, batch)
		result[i] = copyBatch
	}
	return result
}

func (f *fakeBunkerWebAPI) DeletedPlugins() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]string, len(f.deletedPlugins))
	copy(result, f.deletedPlugins)
	return result
}

func (f *fakeBunkerWebAPI) RunJobsHistory() []RunJobsRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]RunJobsRequest, len(f.runJobs))
	copy(result, f.runJobs)
	return result
}

func (f *fakeBunkerWebAPI) Config(service, cfgType, name string) (*bunkerWebConfig, bool) {
	key := configStorageKey(normalizeConfigService(optionalStringPointer(service)), cfgType, name)
	f.mu.Lock()
	cfg, ok := f.configs[key]
	f.mu.Unlock()
	if !ok {
		return nil, false
	}
	copyCfg := *cfg
	return &copyCfg, true
}

func (f *fakeBunkerWebAPI) Ban(ip, service string) (*bunkerWebBan, bool) {
	key := banStorageKey(strings.TrimSpace(ip), optionalStringPointer(strings.TrimSpace(service)))
	f.mu.Lock()
	ban, ok := f.bans[key]
	f.mu.Unlock()
	if !ok {
		return nil, false
	}
	copyBan := *ban
	return &copyBan, true
}

func (f *fakeBunkerWebAPI) Plugin(id string) (*bunkerWebPlugin, bool) {
	f.mu.Lock()
	plugin, ok := f.plugins[id]
	f.mu.Unlock()
	if !ok {
		return nil, false
	}
	copyPlugin := *plugin
	return &copyPlugin, true
}

func (f *fakeBunkerWebAPI) writeSuccess(w http.ResponseWriter, payload any) {
	w.WriteHeader(http.StatusOK)
	body := map[string]any{
		"status": "ok",
		"data":   payload,
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		f.t.Fatalf("failed to serialize payload: %v", err)
	}
}

func (f *fakeBunkerWebAPI) writeError(w http.ResponseWriter, status int, message string) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "error",
		"message": message,
		"data":    nil,
	})
}

func decodeBanRequests(body io.Reader) ([]BanRequest, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("unable to read request body")
	}
	if len(bytesTrimSpace(data)) == 0 {
		return nil, nil
	}

	if data[0] == '[' {
		var reqs []BanRequest
		if err := json.Unmarshal(data, &reqs); err != nil {
			return nil, fmt.Errorf("invalid request body")
		}
		return reqs, nil
	}

	var single BanRequest
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("invalid request body")
	}
	return []BanRequest{single}, nil
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	for ; start < len(b); start++ {
		if !unicode.IsSpace(rune(b[start])) {
			break
		}
	}
	end := len(b)
	for end > start {
		if !unicode.IsSpace(rune(b[end-1])) {
			break
		}
		end--
	}
	return b[start:end]
}

func parseConfigPathParts(fullPath string) (service string, cfgType string, name string, err error) {
	trimmed := strings.TrimPrefix(fullPath, "/configs/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid path")
	}

	if len(parts) == 2 {
		service = "global"
		cfgType = parts[0]
		name = parts[1]
	} else {
		service = parts[0]
		cfgType = parts[1]
		name = strings.Join(parts[2:], "/")
	}

	service = normalizeConfigService(optionalStringPointer(service))
	cfgType = strings.TrimSpace(cfgType)
	name = strings.TrimSpace(name)
	if cfgType == "" || name == "" {
		return "", "", "", fmt.Errorf("invalid path")
	}

	return service, cfgType, name, nil
}

func sanitizeConfigFileName(raw string) string {
	base := path.Base(raw)
	if base == "." || base == "/" {
		return "config"
	}
	return base
}

func normalizeConfigService(service *string) string {
	if service == nil {
		return "global"
	}
	trimmed := strings.TrimSpace(*service)
	if trimmed == "" || strings.EqualFold(trimmed, "global") {
		return "global"
	}
	return trimmed
}

func normalizeBanService(service *string) string {
	if service == nil {
		return ""
	}
	trimmed := strings.TrimSpace(*service)
	if strings.EqualFold(trimmed, "global") {
		return ""
	}
	return trimmed
}

func configStorageKey(service, cfgType, name string) string {
	if service == "" {
		service = "global"
	}
	return fmt.Sprintf("%s|%s|%s", service, cfgType, name)
}

func banStorageKey(ip string, service *string) string {
	if service == nil {
		return ip
	}
	return fmt.Sprintf("%s|%s", ip, *service)
}

func enumerateConfigKeys(keys []ConfigKey) []ConfigKey {
	result := make([]ConfigKey, 0, len(keys))
	for _, key := range keys {
		if key.Name == "" || key.Type == "" {
			continue
		}
		service := "global"
		if key.Service != nil {
			service = normalizeConfigService(key.Service)
		}
		var servicePtr *string
		if service != "global" {
			svcCopy := service
			servicePtr = &svcCopy
		}
		result = append(result, ConfigKey{Service: servicePtr, Type: key.Type, Name: key.Name})
	}
	return result
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func ptr[T any](v T) *T {
	return &v
}
