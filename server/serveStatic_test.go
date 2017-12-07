package server_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/deciphernow/object-drive-server/util"
)

func TestStealEtcPasswd(t *testing.T) {
	attemptToStealFile(t, "/etc/passwd")
}

func TestStealEtcHosts(t *testing.T) {
	attemptToStealFile(t, "/etc/hosts")
}

func attemptToStealFile(t *testing.T, filename string) {
	if testing.Short() {
		t.Skip()
	}
	clientid := 10

	// URL
	escapedPeriod := "%2e" // %u002e
	upAFolder := "/" + escapedPeriod + escapedPeriod

	//uri := mountPoint + "/static/" + upAFolder + upAFolder + upAFolder + upAFolder + upAFolder + upAFolder + "etc/passwd"
	uri := schemeAuthority + "/static" + upAFolder + upAFolder + upAFolder + upAFolder + upAFolder + upAFolder + filename

	// Request
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Logf("Error setting up HTTP Request: %v", err)
		t.FailNow()
	}
	whitelistedDN := "cn=twl-server-generic2,ou=dae,ou=dia,ou=twl-server-generic2,o=u.s. government,c=us"
	req.Header.Add("USER_DN", fakeDN0)
	req.Header.Add("SSL_CLIENT_S_DN", whitelistedDN)
	req.Header.Add("EXTERNAL_SYS_DN", whitelistedDN)
	req.Header.Add("Content-Type", "application/json")
	res, err := clients[clientid].Client.Do(req)
	if err != nil {
		t.Logf("Unable to do request:%v", err)
		t.Skip()
	}
	defer util.FinishBody(res.Body)
	// Response validation
	if res.StatusCode == http.StatusOK {
		t.Logf("bad status: %s", res.Status)
		t.Fail()
	}
	bodyContent, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Logf("err: %v", err)
		t.FailNow()
	} else {
		t.Logf("Contents of %s\n\n%s", filename, string(bodyContent))
	}
}

func getNetworkModeForContainer(t *testing.T, containerID string) string {

	goTemplate := "'{{ .HostConfig.NetworkMode }}'"
	var cmd *exec.Cmd
	var out bytes.Buffer
	cmd = exec.Command("docker", "inspect", "--format", goTemplate, containerID)
	cmd.Stdout = &out
	err := cmd.Run()
	// If error running, then just return the input
	if err != nil {
		t.Logf("WARNING: Error running docker inspect command: %s", err.Error())
		return ""
	}
	networkMode := getStringFromBuffer(out)
	if testing.Verbose() {
		t.Logf("NetworkMode reported as %s", networkMode)
	}
	return networkMode
}

func getIPAddressForContainer(t *testing.T, containerID string) string {
	networkMode := getNetworkModeForContainer(t, containerID)
	if networkMode == "default" {
		t.Logf("Not using NetworkMode for IP lookup")
		networkMode = ""
	}
	if len(networkMode) > 0 {
		networkMode = ".Networks." + networkMode
	}
	goTemplate := fmt.Sprintf("'{{ .NetworkSettings%s.IPAddress }}'", networkMode)
	if testing.Verbose() {
		t.Logf("Resulting inspection template: %s", goTemplate)
	}
	var cmd *exec.Cmd
	var out bytes.Buffer
	cmd = exec.Command("docker", "inspect", "--format", goTemplate, containerID)
	cmd.Stdout = &out
	err := cmd.Run()
	// If error running, then just return the input
	if err != nil {
		t.Logf("WARNING: Error running docker inspect command: %s", err.Error())
		return ""
	}
	addr := getStringFromBuffer(out)
	return addr
}

func getStringFromBuffer(b bytes.Buffer) string {
	o := b.String()
	o = strings.TrimSpace(o)
	o = strings.Replace(o, "'", "", -1)
	return o
}

func getDockerContainerIDFromName(t *testing.T, name string) string {
	// Commands
	var cmds []*exec.Cmd
	cmds = append(cmds, exec.Command("docker", "ps", "-a"))
	cmds = append(cmds, exec.Command("grep", name))
	cmds = append(cmds, exec.Command("awk", "{print $1}"))
	// Pipes
	last := len(cmds) - 1
	var output bytes.Buffer
	var stderr bytes.Buffer
	for i, cmd := range cmds[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			t.Logf("Error piping commands for cmds[%d+1].Stdin = cmd.StdoutPipe. %s", i, err.Error())
			t.FailNow()
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr
	// Start each command
	for i, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			t.Logf("Error starting command %d %s", i, err.Error())
			t.FailNow()
		}
	}
	// Wait for each command to complete
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Logf("Error waiting for command %d %s", i, err.Error())
			t.FailNow()
		}
	}
	containerid := getStringFromBuffer(output)
	containerid = strings.TrimSpace(strings.Split(containerid, "\n")[0])
	if testing.Verbose() {
		t.Logf("Container ID for %s is %s", name, containerid)
	}
	return containerid
}

func getAddrFromDockerHost(t *testing.T, i string) string {
	// If not docker, then just return the input
	if !strings.Contains(i, "docker") {
		t.Logf("Host is not a docker container")
		return i
	}
	if testing.Verbose() {
		t.Logf("Host is a docker container string.  Inspecting for IP Address.")
	}
	// Assume format is `{container-name}.{NetworkMode}``
	hostparts := strings.Split(i, ".")
	// If not enough parts, just return the input
	if len(hostparts) < 2 {
		return i
	}
	// Build up inspection format using network mode
	containerName := fmt.Sprintf("%s", hostparts[0])
	addr := getIPAddressForContainer(t, containerName)
	if testing.Verbose() {
		t.Logf("Using IP Address %s instead", addr)
	}
	return addr
}
