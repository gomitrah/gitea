// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"path/filepath"
	"testing"

	"gitea.dev/models/db"
	issues_model "gitea.dev/models/issues"
	"gitea.dev/models/unittest"
	"gitea.dev/modules/setting"

	_ "gitea.dev/cmd" // for TestPrimaryKeys

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm/schemas"
)

func TestDumpDatabase(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	dir := t.TempDir()

	type Version struct {
		ID      int64 `xorm:"pk autoincr"`
		Version int64
	}
	assert.NoError(t, db.GetEngine(t.Context()).Sync(new(Version)))

	for _, dbType := range setting.SupportedDatabaseTypes {
		assert.NoError(t, db.DumpDatabase(filepath.Join(dir, dbType+".sql"), setting.DatabaseType(dbType)))
	}
}

func TestDeleteOrphanedObjects(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	countBefore, err := db.GetEngine(t.Context()).Count(&issues_model.PullRequest{})
	assert.NoError(t, err)

	_, err = db.GetEngine(t.Context()).Insert(&issues_model.PullRequest{IssueID: 1000}, &issues_model.PullRequest{IssueID: 1001}, &issues_model.PullRequest{IssueID: 1003})
	assert.NoError(t, err)

	orphaned, err := db.CountOrphanedObjects(t.Context(), "pull_request", "issue", "pull_request.issue_id=issue.id")
	assert.NoError(t, err)
	assert.EqualValues(t, 3, orphaned)

	err = db.DeleteOrphanedObjects(t.Context(), "pull_request", "issue", "pull_request.issue_id=issue.id")
	assert.NoError(t, err)

	countAfter, err := db.GetEngine(t.Context()).Count(&issues_model.PullRequest{})
	assert.NoError(t, err)
	assert.Equal(t, countBefore, countAfter)
}

func TestPrimaryKeys(t *testing.T) {
	// Some dbs require that all tables have primary keys, see
	//   https://github.com/go-gitea/gitea/issues/21086
	//   https://github.com/go-gitea/gitea/issues/16802
	// To avoid creating tables without primary key again, this test will check them.
	// Import "gitea.dev/cmd" to make sure each db.RegisterModel in init functions has been called.

	beans, err := db.NamesToBean()
	require.NoError(t, err)

	whitelist := map[string]string{
		"the_table_name_to_skip_checking": "Write a note here to explain why",
	}

	for _, bean := range beans {
		table, err := db.GetXORMEngineForTesting().TableInfo(bean)
		if err != nil {
			t.Fatal(err)
		}
		if why, ok := whitelist[table.Name]; ok {
			t.Logf("ignore %q because %q", table.Name, why)
			continue
		}
		assert.NotEmpty(t, table.PrimaryKeys, "table %q has no primary key", table.Name)
	}
}

func TestSyncAllTablesRestoresActionIndexes(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	engine := db.GetXORMEngineForTesting()
	indexes, err := engine.Dialect().GetIndexes(engine.DB(), t.Context(), "action")
	require.NoError(t, err)

	missingIndex, ok := indexes["c_u"]
	require.True(t, ok)
	_, err = engine.Exec(engine.Dialect().DropIndexSQL("action", missingIndex))
	require.NoError(t, err)

	require.NoError(t, db.SyncAllTables())

	indexes, err = engine.Dialect().GetIndexes(engine.DB(), t.Context(), "action")
	require.NoError(t, err)
	assertActionIndex(t, indexes, "c_u", []string{"user_id", "is_deleted", "created_unix"})
	assertActionIndex(t, indexes, "c_u_d", []string{"created_unix", "user_id", "is_deleted"})

	require.NoError(t, db.SyncAllTables())
	indexes, err = engine.Dialect().GetIndexes(engine.DB(), t.Context(), "action")
	require.NoError(t, err)
	index, ok := indexes["c_u_d"]
	require.True(t, ok)
	_, err = engine.Exec(engine.Dialect().DropIndexSQL("action", index))
	require.NoError(t, err)

	require.NoError(t, db.SyncAllTables())
	indexes, err = engine.Dialect().GetIndexes(engine.DB(), t.Context(), "action")
	require.NoError(t, err)
	assertActionIndex(t, indexes, "c_u", []string{"user_id", "is_deleted", "created_unix"})
	assertActionIndex(t, indexes, "c_u_d", []string{"created_unix", "user_id", "is_deleted"})
}

func assertActionIndex(t *testing.T, indexes map[string]*schemas.Index, name string, columns []string) {
	t.Helper()
	index, ok := indexes[name]
	require.True(t, ok, "expected action index %q", name)
	assert.Equal(t, schemas.IndexType, index.Type)
	assert.Equal(t, columns, index.Cols)
}
