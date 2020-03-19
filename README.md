![](https://github.com/pjvds/fdbtest/workflows/Go/badge.svg)

# fdbtest

Package to support integration tests against FoundationDB.

It provides an Go API to bootstrap an dockerized cluster with initialized database. 
It then generates an `clusterfile` that can be used to connect the client library. 

There are convenient methods for starting and tearing down, or to clear the cluster between tests.

```go
import (
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/pjvds/fdbtest"
)

func init(){
	fdb.MustAPIVersion(620)
}

func TestRoundtrip(t *testing.T) {
	// start foundationdb node
	node := fdbtest.MustStart()
	
	// destroy node at the end of this test
	defer node.Destroy()

	// get the database
	db := node.DB

	// set foo key to bar
	_, err := db.Transact(func(tx fdb.Transaction) (interface{}, error) {
		tx.Set(fdb.Key("foo"), []byte("bar"))
		return nil, nil
	})
	assert.NoError(t, err)

	// get foo key
	value, err := db.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return tx.Get(fdb.Key("foo")).Get()
	})

	// assert result
	assert.NoError(t, err)
	assert.Equal(t, []byte("bar"), value)}
}
```
