package fdbtest_test

import (
	"os"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/pjvds/fdbtest"
)

func init() {
	fdb.MustAPIVersion(610)
}

func BenchmarkRoundtrip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		node := fdbtest.MustStart()
		node.Destroy()
	}
}

func TestRoundtrip(t *testing.T) {
	context := fdbtest.Context{
		Logger:  fdbtest.WriterLogger{os.Stdout},
		Verbose: true,
	}

	// start foundationdb node
	node := context.MustStart()
	defer node.Destroy()

	// get the database
	db := node.DB

	// set foo key to bar
	_, err := db.Transact(func(tx fdb.Transaction) (interface{}, error) {
		tx.Set(fdb.Key("foo"), []byte("bar"))
		return nil, nil
	})
	if err != nil {
		t.Fatalf("set foo key failed: %v", err.Error())
	}

	// get foo key
	value, err := db.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return tx.Get(fdb.Key("foo")).Get()
	})
	if err != nil {
		t.Fatalf("get foo key failed: %v", err.Error())
	}

	// assert foo value
	if "bar" != string(value.([]byte)) {
		t.Fatalf("expected bar, got %v", string(value.([]byte)))
	}
}
