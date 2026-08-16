// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	appgw "github.com/Rain-kl/Wavelet/internal/apps/message_gateway"
	db "github.com/Rain-kl/Wavelet/internal/infra/persistence"
	"github.com/Rain-kl/Wavelet/internal/listener"
	"github.com/Rain-kl/Wavelet/internal/model"
	"github.com/Rain-kl/Wavelet/internal/repository"
	"github.com/Rain-kl/Wavelet/pkg/logger"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway/channel/qq"
	"github.com/Rain-kl/Wavelet/pkg/message_gateway/channel/telegram"
)

const (
	reloadInterval = 5 * time.Second
	lockTTL        = 30 * time.Second
	lockRefresh    = 10 * time.Second
)

var registerFactoriesOnce sync.Once

func registerFactories() {
	registerFactoriesOnce.Do(func() {
		message_gateway.Register(message_gateway.ChannelTypeTelegram, telegram.New)
		message_gateway.Register(message_gateway.ChannelTypeQQ, qq.New)
	})
}

type runningChannel struct {
	ch        message_gateway.Channel
	updatedAt time.Time
	cancel    context.CancelFunc
}

// Start loads enabled channels, connects adapters, and reloads on change.
func Start(ctx context.Context) error {
	registerFactories()
	r := &gateway{
		node:    nodeID(),
		running: map[uint64]*runningChannel{},
	}
	if err := r.sync(ctx); err != nil {
		logger.ErrorF(ctx, "message-gateway initial sync: %v", err)
	}
	ticker := time.NewTicker(reloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			r.stopAll(ctx)
			return ctx.Err()
		case <-ticker.C:
			if err := repository.DeleteExpiredPairingCodes(ctx); err != nil {
				logger.WarnF(ctx, "message-gateway expire pairing: %v", err)
			}
			if err := r.sync(ctx); err != nil {
				logger.ErrorF(ctx, "message-gateway sync: %v", err)
			}
		}
	}
}

type gateway struct {
	node    string
	mu      sync.Mutex
	running map[uint64]*runningChannel
}

func (r *gateway) sync(ctx context.Context) error {
	rows, err := repository.ListMessageChannels(ctx)
	if err != nil {
		return err
	}
	live := make(map[uint64]model.MessageChannel, len(rows))
	for _, row := range rows {
		live[row.ID] = row
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, run := range r.running {
		row, ok := live[id]
		if !ok || !row.Enabled || !row.UpdatedAt.Equal(run.updatedAt) {
			r.stopLocked(ctx, id)
		}
	}

	for id, row := range live {
		if !row.Enabled {
			continue
		}
		if _, ok := r.running[id]; ok {
			continue
		}
		if err := r.startLocked(ctx, row); err != nil {
			logger.ErrorF(ctx, "message-gateway start channel %d: %v", id, err)
		}
	}
	return nil
}

func (r *gateway) startLocked(ctx context.Context, row model.MessageChannel) error {
	if !r.acquireLock(ctx, row.ID) {
		logger.InfoF(ctx, "message-gateway skip channel %d: lock held", row.ID)
		return nil
	}

	creds, err := appgw.DecryptCredentials(row.Credentials)
	if err != nil {
		logger.ErrorF(ctx, "message-gateway decrypt channel %d: %v", row.ID, err)
		return nil
	}
	factory, ok := message_gateway.Lookup(row.Type)
	if !ok {
		return fmt.Errorf("unknown channel type %q", row.Type)
	}

	runCtx, cancel := context.WithCancel(ctx)
	var live message_gateway.Channel
	deps := inboundDeps{
		LookupBinding: repository.GetBindingByChannelPlatform,
		UpsertCode:    repository.UpsertPairingCode,
		GenerateCode:  message_gateway.GenerateCode,
		Emit: func(ctx context.Context, msg message_gateway.InboundMessage) error {
			return listener.EmitMessageGatewayInbound(ctx, msg)
		},
		Send: func(ctx context.Context, to message_gateway.Recipient, msg message_gateway.OutboundMessage) error {
			if live == nil {
				return fmt.Errorf("qq/telegram channel not ready")
			}
			return live.Send(ctx, to, msg)
		},
	}
	ch, err := factory(message_gateway.ChannelConfig{
		ID:          row.ID,
		Type:        row.Type,
		Name:        row.Name,
		Credentials: creds,
		Extra:       appgw.ParseExtra(row.Extra),
	}, func(ctx context.Context, msg message_gateway.InboundMessage) error {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorF(ctx, "message-gateway inbound panic channel %d: %v", row.ID, rec)
			}
		}()
		return deps.Handle(ctx, msg)
	})
	if err != nil {
		cancel()
		return err
	}
	live = ch
	if err := ch.Connect(runCtx); err != nil {
		cancel()
		_ = ch.Disconnect(ctx)
		return err
	}
	r.running[row.ID] = &runningChannel{ch: ch, updatedAt: row.UpdatedAt, cancel: cancel}
	go r.keepLock(runCtx, row.ID)
	logger.InfoF(ctx, "message-gateway connected channel %d type=%s", row.ID, row.Type)
	return nil
}

func (r *gateway) stopLocked(ctx context.Context, id uint64) {
	run, ok := r.running[id]
	if !ok {
		return
	}
	run.cancel()
	if err := run.ch.Disconnect(ctx); err != nil {
		logger.WarnF(ctx, "message-gateway disconnect channel %d: %v", id, err)
	}
	delete(r.running, id)
	r.releaseLock(ctx, id)
}

func (r *gateway) stopAll(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.running {
		r.stopLocked(ctx, id)
	}
}

func (r *gateway) acquireLock(ctx context.Context, id uint64) bool {
	if db.Redis == nil {
		return true
	}
	ok, err := db.Redis.SetNX(ctx, lockKey(id), r.node, lockTTL).Result()
	if err != nil {
		logger.WarnF(ctx, "message-gateway lock channel %d: %v", id, err)
		return true
	}
	return ok
}

func (r *gateway) keepLock(ctx context.Context, id uint64) {
	if db.Redis == nil {
		return
	}
	ticker := time.NewTicker(lockRefresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = db.Redis.Expire(ctx, lockKey(id), lockTTL).Err()
		}
	}
}

func (r *gateway) releaseLock(ctx context.Context, id uint64) {
	if db.Redis == nil {
		return
	}
	_ = db.Redis.Del(ctx, lockKey(id)).Err()
}

func lockKey(id uint64) string {
	return db.PrefixedKey(fmt.Sprintf("wg:channel:%d", id))
}

func nodeID() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}
