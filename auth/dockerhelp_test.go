package auth_test

// The functions contained herein are specific to identifying the IP address of given
// docker containers using their alias identifier from docker-compose files or by ID.
// This is needed because when docker-compose spins up, it sets up its own network
// for local ips known within the docker containers, but our testing from the host is
// performed outside of the containers, and cannot leverage DNS to identify containers
// by their internally recognized hostnames.

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// docker ps -a | grep zk | awk '{print $1}'

// lucasmoten@lucas-ubuntu:~/workspace/cte/object-drive/docker$ docker ps -a | grep zk | awk '{print $1}'
// eb6dde1947ce
// lucasmoten@lucas-ubuntu:~/workspace/cte/object-drive/docker$ docker inspect -f '{{.NetworkSettings.IPAddress }}' eb6dde1947ce

// lucasmoten@lucas-ubuntu:~/workspace/cte/object-drive/docker$ docker inspect -f '{{.HostConfig.NetworkMode }}' eb6dde1947ce
// docker_default
// lucasmoten@lucas-ubuntu:~/workspace/cte/object-drive/docker$ docker inspect -f '{{.NetworkSettings.Networks.docker_default.IPAddress }}' eb6dde1947ce
// 172.18.0.2

// XXX When ports get exposed to the gateway, they may not be reachable by machine IP
// because they get bound to the gateway (gatekeeper does this).  So in that case, you get lies where you are told of an IP, yet
// you can't connect to the port of the IP of the machine that actually runs it.  In this case,
// the port is effectively bound to a different machine (the gateway) from the one that executes the process.
//
// Also, forcing routable addresses is a problem inside a container because even if docker is installed, it won't find the daemon.

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
	t.Logf("NetworkMode reported as %s", networkMode)
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
	t.Logf("Resulting inspection template: %s", goTemplate)
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
	t.Logf("Container ID for %s is %s", name, containerid)
	return containerid
}

func getAddrFromDockerHost(t *testing.T, i string) string {
	// docker_aac_1.docker_default
	// If not docker, then just return the input
	if !strings.Contains(i, "docker") {
		t.Logf("Host is not a docker container")
		return i
	}
	t.Logf("Host is a docker container string.  Inspecting for IP Address.")
	// Assume format is `{container-name}.{NetworkMode}``
	hostparts := strings.Split(i, ".")
	// If not enough parts, just return the input
	if len(hostparts) < 2 {
		return i
	}
	// Build up inspection format using network mode
	containerName := fmt.Sprintf("%s", hostparts[0])
	addr := getIPAddressForContainer(t, containerName)
	t.Logf("Using IP Address %s instead", addr)
	return addr
}
