package fdbtest_test

import (
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/pjvds/fdbtest"
	"github.com/stretchr/testify/assert"
)

func TestRoundtrip(t *testing.T) {
	fdbServer := new(fdbtest.FdbServer)
	server, err := fdbtest.Start()

	assert.Nil(t, err, "foundationdb should start")
	defer fdbServer.Destroy()

	db, err := server.OpenDB()
	assert.Nil(t, err, "database should start")

	_, err = db.Transact(func(tx fdb.Transaction) (interface{}, error) {
		tx.Set(fdb.Key("foo"), []byte("bar"))
		return nil, nil
	})
	assert.Nil(t, err, "set foo should succeed")

	value, err := db.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return tx.Get(fdb.Key("foo")).Get()
	})

	assert.Nil(t, err, "get foo should succeed")
	assert.Equal(t, "bar", string(value.([]byte)), "foo value should be bar")
}
