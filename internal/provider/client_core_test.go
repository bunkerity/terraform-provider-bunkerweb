// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"testing"
)

func TestBunkerWebClientPing(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	payload, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping returned error: %v", err)
	}

	if payload == nil {
		t.Fatalf("expected payload from Ping")
	}

	if val, ok := payload["pong"].(bool); !ok || !val {
		t.Fatalf("expected pong=true in payload: %#v", payload)
	}
}

func TestBunkerWebClientHealth(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	payload, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}

	if payload == nil {
		t.Fatalf("expected payload from Health")
	}

	if val, ok := payload["status"].(string); !ok || val != "healthy" {
		t.Fatalf("expected status=healthy in payload: %#v", payload)
	}
}

func TestBunkerWebClientLogin(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	token, err := client.Login(ctx, "admin", "secret")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if token != "token-admin" {
		t.Fatalf("unexpected token: %s", token)
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	if api.LastAuthorization() != expectedAuth {
		t.Fatalf("expected login to use basic auth header, got %q", api.LastAuthorization())
	}
}

func TestBunkerWebClientLoginValidation(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	if _, err := client.Login(ctx, "", "secret"); err == nil {
		t.Fatalf("expected error for empty username")
	}

	if _, err := client.Login(ctx, "admin", ""); err == nil {
		t.Fatalf("expected error for empty password")
	}

	token, err := client.Login(ctx, "admin", "wrong")
	if err == nil {
		t.Fatalf("expected error for wrong password, got token %q", token)
	}

	var apiErr *bunkerWebAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected bunkerWebAPIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d", apiErr.StatusCode)
	}
}

func TestBunkerWebClientDeleteInstances(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	if _, err := client.CreateInstance(ctx, InstanceCreateRequest{Hostname: "edge-1"}); err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}

	if _, err := client.CreateInstance(ctx, InstanceCreateRequest{Hostname: "edge-2"}); err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}

	if err := client.DeleteInstances(ctx, []string{"edge-1", "edge-2"}); err != nil {
		t.Fatalf("DeleteInstances: %v", err)
	}

	batches := api.DeletedInstanceBatches()
	if len(batches) != 1 {
		t.Fatalf("expected one batch, got %d", len(batches))
	}

	if len(batches[0]) != 2 {
		t.Fatalf("expected both hostnames to be deleted, got %v", batches[0])
	}

	instances, err := client.ListInstances(ctx)
	if err != nil {
		t.Fatalf("ListInstances: %v", err)
	}

	if len(instances) != 0 {
		t.Fatalf("expected no instances remaining, got %d", len(instances))
	}

	if err := client.DeleteInstances(ctx, []string{}); err == nil {
		t.Fatalf("expected validation error for empty hostname slice")
	}
}

func TestBunkerWebClientInstancePingActions(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	if _, err := client.CreateInstance(ctx, InstanceCreateRequest{Hostname: "edge-1"}); err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}

	payload, err := client.PingInstances(ctx)
	if err != nil {
		t.Fatalf("PingInstances: %v", err)
	}

	if val, ok := payload["pinged"].(float64); !ok || int(val) != 1 {
		t.Fatalf("expected pinged=1, got %v", payload["pinged"])
	}

	if _, err := client.PingInstance(ctx, "edge-1"); err != nil {
		t.Fatalf("PingInstance: %v", err)
	}

	hosts := api.PingHosts()
	if len(hosts) != 1 || hosts[0] != "edge-1" {
		t.Fatalf("expected ping host history to contain edge-1, got %v", hosts)
	}
}

func TestBunkerWebClientInstanceReloadActions(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	if _, err := client.CreateInstance(ctx, InstanceCreateRequest{Hostname: "edge-1"}); err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}

	falseVal := false
	if _, err := client.ReloadInstances(ctx, &falseVal); err != nil {
		t.Fatalf("ReloadInstances: %v", err)
	}

	allTests := api.ReloadAllTests()
	if len(allTests) == 0 || allTests[len(allTests)-1] != false {
		t.Fatalf("expected reload all to record test=false, history=%v", allTests)
	}

	if _, err := client.ReloadInstance(ctx, "edge-1", nil); err != nil {
		t.Fatalf("ReloadInstance: %v", err)
	}

	hostCalls := api.ReloadHostCalls()
	if len(hostCalls) == 0 || hostCalls[len(hostCalls)-1].host != "edge-1" {
		t.Fatalf("expected reload host history to include edge-1, got %v", hostCalls)
	}
	if hostCalls[len(hostCalls)-1].test != true {
		t.Fatalf("expected default test flag true when omitted, got %v", hostCalls[len(hostCalls)-1].test)
	}
}

func TestBunkerWebClientInstanceStopActions(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	if _, err := client.CreateInstance(ctx, InstanceCreateRequest{Hostname: "edge-1"}); err != nil {
		t.Fatalf("CreateInstance: %v", err)
	}

	if _, err := client.StopInstances(ctx); err != nil {
		t.Fatalf("StopInstances: %v", err)
	}

	if api.StopAllCount() != 1 {
		t.Fatalf("expected stop all count to increment")
	}

	if _, err := client.StopInstance(ctx, "edge-1"); err != nil {
		t.Fatalf("StopInstance: %v", err)
	}

	hosts := api.StopHosts()
	if len(hosts) == 0 || hosts[len(hosts)-1] != "edge-1" {
		t.Fatalf("expected stop host history to include edge-1, got %v", hosts)
	}
}

func TestBunkerWebClientConvertService(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	svc, err := client.CreateService(ctx, ServiceCreateRequest{
		ServerName: "app.example.com",
		IsDraft:    false,
	})
	if err != nil {
		t.Fatalf("CreateService: %v", err)
	}

	converted, err := client.ConvertService(ctx, svc.ID, "draft")
	if err != nil {
		t.Fatalf("ConvertService: %v", err)
	}

	if !converted.IsDraft {
		t.Fatalf("expected service to be draft after conversion")
	}

	calls := api.ConvertCalls()
	if len(calls) == 0 || calls[len(calls)-1].target != "draft" {
		t.Fatalf("expected convert call history to include draft target, got %v", calls)
	}

	if _, err := client.ConvertService(ctx, svc.ID, "invalid"); err == nil {
		t.Fatalf("expected error for invalid convert target")
	}
}

func TestBunkerWebClientUpdateGlobalConfig(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()

	if _, err := client.UpdateGlobalConfig(ctx, nil); err == nil {
		t.Fatalf("expected error for nil settings map")
	}

	if _, err := client.UpdateGlobalConfig(ctx, map[string]any{}); err == nil {
		t.Fatalf("expected error for empty settings map")
	}

	patch := map[string]any{
		"retry_limit": 10,
		"new_feature": true,
	}

	updated, err := client.UpdateGlobalConfig(ctx, patch)
	if err != nil {
		t.Fatalf("UpdateGlobalConfig: %v", err)
	}

	if val, ok := updated["retry_limit"].(float64); !ok || val != 10 {
		t.Fatalf("expected retry_limit=10, got %#v", updated["retry_limit"])
	}
	if val, ok := updated["new_feature"].(bool); !ok || !val {
		t.Fatalf("expected new_feature=true, got %#v", updated["new_feature"])
	}

	lastPatch := api.LastGlobalPatch()
	if val, ok := lastPatch["retry_limit"].(float64); !ok || val != 10 {
		t.Fatalf("expected fake API to record retry_limit=10, got %#v", lastPatch["retry_limit"])
	}
	if val, ok := lastPatch["new_feature"].(bool); !ok || !val {
		t.Fatalf("expected fake API to record new_feature=true, got %#v", lastPatch["new_feature"])
	}

	config, err := client.GetGlobalConfig(ctx, true, false)
	if err != nil {
		t.Fatalf("GetGlobalConfig: %v", err)
	}
	if val, ok := config["retry_limit"].(float64); !ok || val != 10 {
		t.Fatalf("expected retry_limit updated to 10, got %#v", config["retry_limit"])
	}
	if val, ok := config["new_feature"].(bool); !ok || !val {
		t.Fatalf("expected new_feature=true in global config, got %#v", config["new_feature"])
	}
}

func TestBunkerWebClientDeleteConfigs(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()
	service := "app"
	if _, err := client.CreateConfig(ctx, ConfigCreateRequest{Service: &service, Type: "http", Name: "block", Data: "deny all;"}); err != nil {
		t.Fatalf("CreateConfig: %v", err)
	}

	key := ConfigKey{Service: &service, Type: "http", Name: "block"}
	if err := client.DeleteConfigs(ctx, []ConfigKey{key}); err != nil {
		t.Fatalf("DeleteConfigs: %v", err)
	}

	batches := api.DeletedConfigBatches()
	if len(batches) != 1 {
		t.Fatalf("expected one delete batch, got %d", len(batches))
	}
	if len(batches[0]) != 1 || batches[0][0].Name != "block" {
		t.Fatalf("unexpected delete batch contents: %#v", batches[0])
	}

	if _, err := client.GetConfig(ctx, key, true); err == nil {
		t.Fatalf("expected config to be deleted")
	} else {
		var apiErr *bunkerWebAPIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 bunkerWebAPIError, got %v", err)
		}
	}

	if err := client.DeleteConfigs(ctx, nil); err == nil {
		t.Fatalf("expected error for nil config keys slice")
	}
	if err := client.DeleteConfigs(ctx, []ConfigKey{}); err == nil {
		t.Fatalf("expected error for empty config keys slice")
	}
}

func TestBunkerWebClientUploadConfigs(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()
	files := []ConfigUploadFile{
		{FileName: "main.conf", Content: []byte("content-1")},
		{FileName: "Extra.cfg", Content: []byte("content-2")},
	}

	configs, err := client.UploadConfigs(ctx, ConfigUploadRequest{Service: "web", Type: "http", Files: files})
	if err != nil {
		t.Fatalf("UploadConfigs: %v", err)
	}

	if len(configs) != 2 {
		t.Fatalf("expected two configs returned, got %d", len(configs))
	}

	for i, cfg := range configs {
		if cfg.Service != "web" {
			t.Fatalf("expected config service to be 'web', got %q", cfg.Service)
		}
		expectedData := "content-" + strconv.Itoa(i+1)
		if cfg.Data != expectedData {
			t.Fatalf("expected data %q, got %q", expectedData, cfg.Data)
		}
		svc := cfg.Service
		key := ConfigKey{Service: &svc, Type: cfg.Type, Name: cfg.Name}
		fetched, err := client.GetConfig(ctx, key, true)
		if err != nil {
			t.Fatalf("GetConfig after upload: %v", err)
		}
		if fetched.Data != expectedData {
			t.Fatalf("expected fetched data %q, got %q", expectedData, fetched.Data)
		}
	}

	if _, err := client.UploadConfigs(ctx, ConfigUploadRequest{Service: "web", Type: "http"}); err == nil {
		t.Fatalf("expected error when no files provided")
	}
	if _, err := client.UploadConfigs(ctx, ConfigUploadRequest{Service: "web", Files: files}); err == nil {
		t.Fatalf("expected error when type is missing")
	}
}

func TestBunkerWebClientUpdateConfigFromUpload(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()
	if _, err := client.CreateConfig(ctx, ConfigCreateRequest{Type: "http", Name: "primary", Data: "initial"}); err != nil {
		t.Fatalf("CreateConfig: %v", err)
	}

	originalKey := ConfigKey{Type: "http", Name: "primary"}
	newService := "backend"
	newType := "stream"
	newName := "processed"
	updated, err := client.UpdateConfigFromUpload(ctx, originalKey, ConfigUploadUpdateRequest{
		FileName:   "override.conf",
		Content:    []byte("updated"),
		NewService: &newService,
		NewType:    &newType,
		NewName:    &newName,
	})
	if err != nil {
		t.Fatalf("UpdateConfigFromUpload: %v", err)
	}

	if updated.Service != newService {
		t.Fatalf("expected service to change to %q, got %q", newService, updated.Service)
	}
	if updated.Type != newType {
		t.Fatalf("expected type to change to %q, got %q", newType, updated.Type)
	}
	if updated.Name != newName {
		t.Fatalf("expected name to change to %q, got %q", newName, updated.Name)
	}
	if updated.Data != "updated" {
		t.Fatalf("expected updated data, got %q", updated.Data)
	}

	newKey := ConfigKey{Service: &newService, Type: newType, Name: newName}
	fetched, err := client.GetConfig(ctx, newKey, true)
	if err != nil {
		t.Fatalf("GetConfig for updated config: %v", err)
	}
	if fetched.Data != "updated" {
		t.Fatalf("expected fetched data to be updated, got %q", fetched.Data)
	}

	if _, err := client.GetConfig(ctx, originalKey, true); err == nil {
		t.Fatalf("expected original config location to return not found")
	}

	if _, err := client.UpdateConfigFromUpload(ctx, originalKey, ConfigUploadUpdateRequest{}); err == nil {
		t.Fatalf("expected validation error for missing file name")
	}
}

func TestBunkerWebClientListConfigs(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()
	service := "app"
	if _, err := client.CreateConfig(ctx, ConfigCreateRequest{Service: &service, Type: "http", Name: "app.conf", Data: "app-data"}); err != nil {
		t.Fatalf("CreateConfig app: %v", err)
	}
	if _, err := client.CreateConfig(ctx, ConfigCreateRequest{Type: "stream", Name: "global.conf", Data: "global-data"}); err != nil {
		t.Fatalf("CreateConfig global: %v", err)
	}

	configs, err := client.ListConfigs(ctx, ConfigListOptions{})
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected two configs, got %d", len(configs))
	}
	for _, cfg := range configs {
		if cfg.Data != "" {
			t.Fatalf("expected data omitted when with_data not requested: %#v", cfg)
		}
	}

	withData := true
	configsWithData, err := client.ListConfigs(ctx, ConfigListOptions{WithData: &withData})
	if err != nil {
		t.Fatalf("ListConfigs with data: %v", err)
	}
	if len(configsWithData) != 2 {
		t.Fatalf("expected two configs with data, got %d", len(configsWithData))
	}

	var foundApp bool
	for _, cfg := range configsWithData {
		if cfg.Service == "app" && cfg.Type == "http" && cfg.Name == "app.conf" {
			foundApp = true
			if cfg.Data != "app-data" {
				t.Fatalf("expected app config data preserved, got %q", cfg.Data)
			}
		}
	}
	if !foundApp {
		t.Fatalf("expected to find app config in list: %#v", configsWithData)
	}

	filterType := "http"
	filtered, err := client.ListConfigs(ctx, ConfigListOptions{Service: &service, Type: &filterType, WithData: &withData})
	if err != nil {
		t.Fatalf("ListConfigs with filters: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected one filtered config, got %d", len(filtered))
	}
	if filtered[0].Service != "app" || filtered[0].Type != "http" || filtered[0].Name != "app.conf" {
		t.Fatalf("unexpected filtered config: %#v", filtered[0])
	}
	if filtered[0].Data != "app-data" {
		t.Fatalf("expected filtered config data, got %q", filtered[0].Data)
	}
}

func TestBunkerWebClientBanBulk(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()
	reason := "abuse"
	exp := 3600
	service := "frontend"
	bans := []BanRequest{
		{IP: "10.0.0.1", Reason: &reason, Exp: &exp, Service: &service},
		{IP: "10.0.0.2"},
	}

	if err := client.BanBulk(ctx, bans); err != nil {
		t.Fatalf("BanBulk: %v", err)
	}

	created := api.CreatedBanBatches()
	if len(created) != 1 || len(created[0]) != 2 {
		t.Fatalf("expected one batch of two bans, got %#v", created)
	}

	bansList, err := client.ListBans(ctx)
	if err != nil {
		t.Fatalf("ListBans: %v", err)
	}
	if len(bansList) != 2 {
		t.Fatalf("expected two bans, got %d", len(bansList))
	}

	unbans := []UnbanRequest{{IP: "10.0.0.1", Service: &service}, {IP: "10.0.0.2"}}
	if err := client.UnbanBulk(ctx, unbans); err != nil {
		t.Fatalf("UnbanBulk: %v", err)
	}

	deleted := api.DeletedBanBatches()
	if len(deleted) != 1 || len(deleted[0]) != 2 {
		t.Fatalf("expected one batch of two unbans, got %#v", deleted)
	}

	remaining, err := client.ListBans(ctx)
	if err != nil {
		t.Fatalf("ListBans after unban: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no bans after unban, got %d", len(remaining))
	}

	if err := client.BanBulk(ctx, []BanRequest{}); err == nil {
		t.Fatalf("expected error for empty ban batch")
	}
	if err := client.UnbanBulk(ctx, nil); err == nil {
		t.Fatalf("expected error for empty unban batch")
	}
}

func TestBunkerWebClientPluginLifecycle(t *testing.T) {
	api := newFakeBunkerWebAPI(t)
	client, err := newBunkerWebClient(api.URL(), nil, "", "", "")
	if err != nil {
		t.Fatalf("newBunkerWebClient: %v", err)
	}

	ctx := context.Background()
	plugins, err := client.UploadPlugins(ctx, PluginUploadRequest{
		Method: "custom",
		Files: []PluginUploadFile{
			{FileName: "first.lua", Content: []byte("return 1")},
			{FileName: "second.lua", Content: []byte("return 2")},
		},
	})
	if err != nil {
		t.Fatalf("UploadPlugins: %v", err)
	}
	if len(plugins) != 2 {
		t.Fatalf("expected two plugins returned, got %d", len(plugins))
	}

	uploaded := api.UploadedPluginBatches()
	if len(uploaded) != 1 || len(uploaded[0]) != 2 {
		t.Fatalf("expected fake API to record upload batch, got %#v", uploaded)
	}

	plugin, ok := api.Plugin("first")
	if !ok {
		t.Fatalf("expected plugin 'first' to exist after upload")
	}
	if plugin.Type != "custom" {
		t.Fatalf("expected plugin type to reflect method, got %q", plugin.Type)
	}

	filtered, err := client.ListPlugins(ctx, "custom", false)
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(filtered) == 0 {
		t.Fatalf("expected custom plugins to be returned")
	}

	if err := client.DeletePlugin(ctx, "first"); err != nil {
		t.Fatalf("DeletePlugin: %v", err)
	}

	deleted := api.DeletedPlugins()
	if len(deleted) == 0 || deleted[len(deleted)-1] != "first" {
		t.Fatalf("expected fake API to record plugin deletion, got %#v", deleted)
	}

	if _, err := client.UploadPlugins(ctx, PluginUploadRequest{}); err == nil {
		t.Fatalf("expected error for missing plugin files")
	}
	if err := client.DeletePlugin(ctx, " "); err == nil {
		t.Fatalf("expected validation error for empty plugin id")
	}
}
