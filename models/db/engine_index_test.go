// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm/schemas"
)

func TestIndexDefinitionsEqual(t *testing.T) {
	expected := schemas.NewIndex("c_u", schemas.IndexType)
	expected.AddColumn("user_id", "is_deleted", "created_unix")

	sameOrder := schemas.NewIndex("same_order", schemas.IndexType)
	sameOrder.AddColumn("user_id", "is_deleted", "created_unix")
	assert.True(t, indexDefinitionsEqual(expected, sameOrder))

	differentOrder := schemas.NewIndex("different_order", schemas.IndexType)
	differentOrder.AddColumn("created_unix", "user_id", "is_deleted")
	assert.False(t, indexDefinitionsEqual(expected, differentOrder))
}
