package fdbtest

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/pkg/errors"
)

type TestContext struct {
	Logger  Logger
	Verbose bool
}

var DefaultContext = TestContext{
	Logger:  &NilLogger{},
	Verbose: false,
}

type FdbServer struct {
	context     *TestContext
	dockerID    string
	clusterFile string
}

// OpenDB returns an open database to the temporary cluster created by Start.
//
// Please make sure to have called fdb.APIVersion() before opening a database.
func (s FdbServer) OpenDB() (fdb.Database, error) {
	db, err := fdb.OpenDatabase(s.clusterFile)
	if s.context.Verbose {
		if err != nil {
			s.context.Logger.Logf("open datatabase error: %v\n", err)
		} else {
			s.context.Logger.Logf("database opened\n")
		}
	}

	return db, err
}

// Destroy destroys the foundationdb cluster.
func (s *FdbServer) Destroy() error {
	return exec.Command("docker", "rm", "--force", s.dockerID).Run()
}

// Start starts a new foundationdb cluster.
func Start() (*FdbServer, error) {
	return DefaultContext.Start()
}

func (ctx *TestContext) Start() (*FdbServer, error) {
	// start new foundationdb docker container
	runCmd := exec.Command("docker", "run", "--detach", "foundationdb/foundationdb:6.2.10")
	if ctx.Verbose {
		ctx.Logger.Logf("+%v\n", runCmd.String())
	}

	output, err := runCmd.CombinedOutput()
	if err != nil {
		ctx.Logger.Logf("docker run error: %v\n\n%v\n", err, output)
		return nil, errors.Wrap(err, "docker run failed")
	}
	// get docker id from output
	dockerID := strings.TrimSpace(string(output))
	if len(dockerID) != 64 {
		return nil, errors.New("invalid docker id in stdout: " + dockerID)
	}
	// trim docker id
	dockerID = dockerID[:12]

	if ctx.Verbose {
		ctx.Logger.Logf("foundationdb container started with id %v\n", dockerID)
	}

	// initialize new database
	initCmd := exec.Command("docker", "exec", dockerID, "fdbcli", "--exec", "configure new single ssd")
	if ctx.Verbose {
		ctx.Logger.Logf("+%v\n", initCmd.String())
	}

	output, err = initCmd.CombinedOutput()
	if err != nil {
		ctx.Logger.Logf("initialize database error: %v\r\n\r\n%v\n", err, string(output))
		return nil, errors.Wrap(err, "docker exec failed: "+string(output))
	}

	if !strings.Contains(string(output), "Database created") {
		return nil, errors.New("unexpected configure database output: " + string(output))
	}

	if ctx.Verbose {
		ctx.Logger.Logf("database initialize command succeeded: %v\n", strings.TrimSpace(string(output)))
	}

	// get container ip
	inspectCmd := exec.Command("docker", "inspect", dockerID, "-f", "{{ .NetworkSettings.Networks.bridge.IPAddress }}")
	if ctx.Verbose {
		ctx.Logger.Logf("+%v\n", inspectCmd.String())
	}
	output, err = inspectCmd.CombinedOutput()
	if err != nil {
		ctx.Logger.Logf("container network ip lookup failed: %v\r\n\r\n%v", err, string(output))
		return nil, errors.Wrap(err, "docker exec inspect: "+string(output))
	}
	ipAddress := strings.TrimSpace(string(output))

	// validate ip
	matched, err := regexp.MatchString("^[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}\\.[0-9]{1,3}$", ipAddress)
	if err != nil {
		return nil, errors.Wrap(err, "ip address regex match error")
	}

	if !matched {
		return nil, errors.New("invalid ip address: " + ipAddress)
	}

	// generate unique cluster file
	clusterFile, err := ioutil.TempFile(os.TempDir(), "fdb.cluster")
	if err != nil {
		return nil, err
	}
	cluster := fmt.Sprintf("docker:docker@%v:4500", string(ipAddress))
	clusterFile.Write([]byte(cluster))

	if ctx.Verbose {
		ctx.Logger.Logf("cluster available: %v\n", cluster)
	}

	return &FdbServer{
		ctx, dockerID, clusterFile.Name(),
	}, nil
}
