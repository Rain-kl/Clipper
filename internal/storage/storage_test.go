// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestStorageCache(t *testing.T) {
	// 1. Reset cache
	ResetCache()

	if activeConfigJSON != "" || activeDriver != "" || activeBackend != nil || !lastChecked.IsZero() {
		t.Fatal("ResetCache did not clear cache variables")
	}

	// 2. Set up cache manually
	expectedConfig := Config{
		Driver: DriverLocal,
		Local:  LocalConfig{Root: "/tmp/wavelet-test"},
	}
	cfgJSON, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatalf("Marshal config failed: %v", err)
	}

	cacheMutex.Lock()
	activeConfigJSON = string(cfgJSON)
	lastChecked = time.Now()
	cacheMutex.Unlock()

	// 3. Call LoadConfig and verify it loads from cache (doesn't hit database, which would fail/panic because DB is not initialized)
	ctx := context.Background()
	loadedCfg, err := LoadConfig(ctx)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loadedCfg.Driver != expectedConfig.Driver || loadedCfg.Local.Root != expectedConfig.Local.Root {
		t.Errorf("Loaded config %+v, expected %+v", loadedCfg, expectedConfig)
	}

	// 4. Test Active() returns cached driver and backend
	mockBnd := &functionBackend{
		put:    func(context.Context, string, io.Reader, int64, string) error { return nil },
		get:    func(context.Context, string) (*Object, error) { return nil, nil },
		delete: func(context.Context, string) error { return nil },
	}

	cacheMutex.Lock()
	activeBackend = mockBnd
	activeDriver = DriverLocal
	cacheMutex.Unlock()

	drv, bnd, err := Active(ctx)
	if err != nil {
		t.Fatalf("Active failed: %v", err)
	}
	if drv != DriverLocal || bnd != mockBnd {
		t.Errorf("Active returned driver %v, backend %v; expected %v, %v", drv, bnd, DriverLocal, mockBnd)
	}

	// 5. Test ForDriver returns cached backend
	bnd2, err := ForDriver(ctx, DriverLocal)
	if err != nil {
		t.Fatalf("ForDriver failed: %v", err)
	}
	if bnd2 != mockBnd {
		t.Errorf("ForDriver returned backend %v, expected %v", bnd2, mockBnd)
	}

	// 6. Test ResetCache again
	ResetCache()
	if activeConfigJSON != "" || activeDriver != "" || activeBackend != nil || !lastChecked.IsZero() {
		t.Fatal("ResetCache did not clear cache variables after setting them")
	}
}
