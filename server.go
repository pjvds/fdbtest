package fdbtest

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/pkg/errors"
	"github.com/rs/xid"
)

type Context struct {
	Logger  Logger
	Verbose bool
}

var DefaultContext = Context{
	Logger:  &NilLogger{},
	Verbose: false,
}

type FdbServer struct {
	context          *Context
	runContext       context.Context
	runContextCancel context.CancelFunc
	clusterFile      string
	DB               fdb.Database
}

func (s FdbServer) MustClear() {
	err := s.Clear()
	if err != nil {
		panic(err)
	}
}

func (s FdbServer) Clear() error {
	_, err := s.DB.Transact(func(tx fdb.Transaction) (interface{}, error) {
		tx.ClearRange(fdb.KeyRange{fdb.Key([]byte{0x00}), fdb.Key([]byte{0xff})})
		return nil, nil
	})

	return err
}

// Destroy destroys the foundationdb cluster.
func (s *FdbServer) Destroy() error {
	s.runContextCancel()
	return nil
}
func MustStart() *FdbServer {
	return DefaultContext.MustStart()
}

// MustStart starts a new foundationdb node.
func (c Context) MustStart() *FdbServer {
	s, err := Start()
	if err != nil {
		panic(err)
	}
	return s
}

// Start starts a new foundationdb cluster.
func Start() (*FdbServer, error) {
	return DefaultContext.Start()
}

func (ctx *Context) Start() (*FdbServer, error) {
	name := xid.New().String()
	runContext, cancel := context.WithCancel(context.Background())

	// start new foundationdb docker container
	runCmd := exec.CommandContext(runContext, "docker", "run", "--name", name, "--rm", "foundationdb/foundationdb:6.2.10")
	if ctx.Verbose {
		ctx.Logger.Logf("+%v\n", runCmd.String())
	}

	if err := runCmd.Start(); err != nil {
		cancel()
		return nil, errors.Wrap(err, "docker run failed")
	}

	time.Sleep(time.Second)

	// initialize new database
	initCmd := exec.Command("docker", "exec", name, "fdbcli", "--exec", "configure new single ssd")
	if ctx.Verbose {
		ctx.Logger.Logf("+%v\n", initCmd.String())
	}

	output, err := initCmd.CombinedOutput()
	if err != nil {
		ctx.Logger.Logf("initialize database error: %v\r\n\r\n%v\n", err, string(output))

		cancel()
		return nil, errors.Wrap(err, "docker exec failed: "+string(output))
	}

	if !strings.Contains(string(output), "Database created") {
		cancel()
		return nil, errors.New("unexpected configure database output: " + string(output))
	}

	if ctx.Verbose {
		ctx.Logger.Logf("database initialize command succeeded: %v\n", strings.TrimSpace(string(output)))
	}

	// get container ip
	inspectCmd := exec.Command("docker", "inspect", name, "-f", "{{ .NetworkSettings.Networks.bridge.IPAddress }}")
	if ctx.Verbose {
		ctx.Logger.Logf("+%v\n", inspectCmd.String())
	}
	output, err = inspectCmd.CombinedOutput()
	if err != nil {
		ctx.Logger.Logf("container network ip lookup failed: %v\r\n\r\n%v", err, string(output))

		cancel()
		return nil, errors.Wrap(err, "docker exec inspect: "+string(output))
	}
	ipAddress := strings.TrimSpace(string(output))

	// validate ip
	matched, err := regexp.MatchString("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$", ipAddress)
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "ip address regex match error")
	}

	if !matched {
		cancel()
		return nil, errors.New("invalid ip address: " + ipAddress)
	}

	// generate unique cluster file
	clusterFile, err := ioutil.TempFile(os.TempDir(), "fdb.cluster")
	if err != nil {
		cancel()
		return nil, err
	}
	cluster := fmt.Sprintf("docker:docker@%v:4500", string(ipAddress))
	clusterFile.Write([]byte(cluster))

	if ctx.Verbose {
		ctx.Logger.Logf("cluster available: %v\n", cluster)
	}

	db, err := fdb.OpenDatabase(clusterFile.Name())
	if err != nil {
		cancel()
		return nil, errors.Wrap(err, "error opening database")
	}

	return &FdbServer{
		ctx, runContext, cancel, clusterFile.Name(), db,
	}, nil
}
