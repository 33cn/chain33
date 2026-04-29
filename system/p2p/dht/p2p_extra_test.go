package dht

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRestartFalse(t *testing.T) {
	p := &P2P{restart: 0}
	assert.False(t, p.isRestart())
}

func TestIsRestartTrue(t *testing.T) {
	p := &P2P{restart: 1}
	assert.True(t, p.isRestart())
}

func TestNewDBDefaults(t *testing.T) {
	dir := t.TempDir()
	db := newDB("", "", dir, 0)
	assert.NotNil(t, db)
	db.Close()
}

func TestNewDBCustomName(t *testing.T) {
	dir := t.TempDir()
	db := newDB("testdb", "leveldb", dir, 128)
	assert.NotNil(t, db)
	db.Close()
}
