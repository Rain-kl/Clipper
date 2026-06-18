// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"context"
	"fmt"

	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/storage"
	"github.com/Rain-kl/Wavelet/pkg/logger"
)

// ReadOnly checks if the storage system is in read-only maintenance mode.
func ReadOnly(ctx context.Context) bool {
	state := LoadMigrationAccessState(ctx)
	if state.LoadErr != nil {
		logger.ErrorF(ctx, "读取存储维护状态失败: %v", state.LoadErr)
		return true
	}
	return state.ReadOnly
}

// OpenStoredObject opens a stored upload object from its configured backend.
func OpenStoredObject(ctx context.Context, upload *model.Upload) (*storage.Object, error) {
	driver := storage.Driver(upload.StorageDriver)
	if driver == "" {
		driver = storage.DriverLocal
	}
	backend, err := backendForStoredDriver(ctx, driver)
	if err != nil {
		return nil, err
	}
	return backend.Get(ctx, upload.FilePath)
}

func backendForStoredDriver(ctx context.Context, driver storage.Driver) (storage.Backend, error) {
	backend, err := storage.ForDriver(ctx, driver)
	if err == nil {
		return backend, nil
	}

	target, ok, targetErr := CurrentMigrationTargetConfig(ctx)
	if targetErr != nil {
		return nil, targetErr
	}
	if ok && target.Driver == driver {
		return storage.NewBackend(ctx, target, driver)
	}
	return nil, fmt.Errorf("storage configuration for driver %q is unavailable", driver)
}

// CurrentMigrationTargetConfig returns the pending migration target config when available.
func CurrentMigrationTargetConfig(ctx context.Context) (storage.Config, bool, error) {
	state := LoadMigrationAccessState(ctx)
	if state.LoadErr != nil {
		return storage.Config{}, false, state.LoadErr
	}
	if state.TargetErr != nil {
		return storage.Config{}, false, state.TargetErr
	}
	if !state.HasTarget {
		return storage.Config{}, false, nil
	}
	return state.Target, true, nil
}