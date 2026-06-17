// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package idgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextUint64ID(t *testing.T) {
	id, err := NextUint64ID()
	require.NoError(t, err)
	assert.NotZero(t, id)
}