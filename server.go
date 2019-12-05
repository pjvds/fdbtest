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

type FdbServer struct {
	dockerID    string
	clusterFile string
}

// OpenDB returns an open database to the temporary cluster created by Start.
//
// Please make sure to have called fdb.APIVersion() before opening a database.
func (s FdbServer) OpenDB() (fdb.Database, error) {
	return fdb.OpenDatabase(s.clusterFile)
}

// Destroy destroys the foundationdb cluster.
func (s *FdbServer) Destroy() error {
	return exec.Command("docker", "rm", "--force", s.dockerID).Run()
}

// Start starts a new foundationdb cluster.
func Start() (*FdbServer, error) {
	// start new foundationdb docker container
	runCmd := exec.Command("docker", "run", "--detach", "foundationdb/foundationdb")
	output, err := runCmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "docker run failed")
	}

	// get docker id from output
	dockerID := strings.TrimSpace(string(output))
	if len(dockerID) != 64 {
		return nil, errors.New("invalid docker id in stdout: " + dockerID)
	}

	// trim docker id
	dockerID = dockerID[:12]

	// initialize new database
	initCmd := exec.Command("docker", "exec", dockerID, "fdbcli", "--exec", "configure new single ssd")
	output, err = initCmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "docker exec failed: "+string(output))
	}
	if !strings.Contains(string(output), "Database created") {
		return nil, errors.New("unexpected configure database output: " + string(output))
	}

	// get container ip
	inspectCmd := exec.Command("docker", "inspect", dockerID, "-f", "{{ .NetworkSettings.Networks.bridge.IPAddress }}")
	output, err = inspectCmd.CombinedOutput()
	if err != nil {
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

	return &FdbServer{
		dockerID, clusterFile.Name(),
	}, nil
}
