package postgres

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pgDSNEnv = "HEALTH_GO_PG_DSN"

func TestNew(t *testing.T) {
	initDB(t)

	check := New(Config{
		DSN: getDSN(t),
	})

	err := check(context.Background())
	require.NoError(t, err)
}

func TestEnsureConnectionIsClosed(t *testing.T) {
	initDB(t)

	pgDSN := getDSN(t)

	db, err := sql.Open("postgres", pgDSN)
	require.NoError(t, err)

	defer func() {
		err := db.Close()
		assert.NoError(t, err)
	}()

	var initialConnections int
	row := db.QueryRow(`SELECT sum(numbackends) FROM pg_stat_database`)
	err = row.Scan(&initialConnections)
	require.NoError(t, err)

	check := New(Config{
		DSN: pgDSN,
	})

	ctx := context.Background()
	for i := 0; i < 10; i++ {
		err := check(ctx)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	var currentConnections int
	row = db.QueryRow(`SELECT sum(numbackends) FROM pg_stat_database`)
	err = row.Scan(&currentConnections)
	require.NoError(t, err)

	assert.Equal(t, initialConnections, currentConnections)
}

func getDSN(t *testing.T) string {
	t.Helper()

	pgDSN, ok := os.LookupEnv(pgDSNEnv)
	require.True(t, ok)

	return pgDSN
}

var dbInit sync.Once

func initDB(t *testing.T) {
	t.Helper()

	dbInit.Do(func() {
		db, err := sql.Open("postgres", getDSN(t))
		require.NoError(t, err)

		defer func() {
			err := db.Close()
			assert.NoError(t, err)
		}()

		_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS test (
  id           TEXT NOT NULL PRIMARY KEY,
  secret       TEXT NOT NULL,
  extra        TEXT NOT NULL,
  redirect_uri TEXT NOT NULL
);
`)
		require.NoError(t, err)
	})
}
