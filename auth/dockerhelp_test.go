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
)

// Sampling...
// $ docker ps -a | grep zk | awk '{print $1}'
// eb6dde1947ce
// $ docker inspect -f '{{.NetworkSettings.IPAddress }}' eb6dde1947ce
// $ docker inspect -f '{{.HostConfig.NetworkMode }}' eb6dde1947ce
// docker_default
// $ docker inspect -f '{{.NetworkSettings.Networks.docker_default.IPAddress }}' eb6dde1947ce
// 172.18.0.2

// XXX When ports get exposed to the gateway, they may not be reachable by machine IP
// because they get bound to the gateway (gatekeeper does this).  So in that case, you get lies where you are told of an IP, yet
// you can't connect to the port of the IP of the machine that actually runs it.  In this case,
// the port is effectively bound to a different machine (the gateway) from the one that executes the process.
//
// Also, forcing routable addresses is a problem inside a container because even if docker is installed, it won't find the daemon.

func getNetworkModeForContainer(containerID string) (string, error) {

	goTemplate := "'{{ .HostConfig.NetworkMode }}'"
	var cmd *exec.Cmd
	var out bytes.Buffer
	cmd = exec.Command("docker", "inspect", "--format", goTemplate, containerID)
	cmd.Stdout = &out
	err := cmd.Run()
	// If error running, then just return the input
	if err != nil {
		return "", err
	}
	networkMode := getStringFromBuffer(out)
	return networkMode, nil
}

func getIPAddressForContainer(containerID string) (string, error) {
	networkMode, err := getNetworkModeForContainer(containerID)
	if err != nil {
		return "", fmt.Errorf("error getting network mode: %s", err.Error())
	}
	if networkMode == "default" {
		networkMode = ""
	}
	if len(networkMode) > 0 {
		networkMode = ".Networks." + networkMode
	}
	goTemplate := fmt.Sprintf("'{{ .NetworkSettings%s.IPAddress }}'", networkMode)
	var cmd *exec.Cmd
	var out bytes.Buffer
	cmd = exec.Command("docker", "inspect", "--format", goTemplate, containerID)
	cmd.Stdout = &out
	err = cmd.Run()
	// If error running, then just return the input
	if err != nil {
		return "", err
	}
	addr := getStringFromBuffer(out)
	return addr, nil
}

func getStringFromBuffer(b bytes.Buffer) string {
	o := b.String()
	o = strings.TrimSpace(o)
	o = strings.Replace(o, "'", "", -1)
	return o
}

func getDockerContainerIDFromName(name string) (string, error) {
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
			return "", err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr
	// Start each command
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return "", err
		}
	}
	// Wait for each command to complete
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return "", err
		}
	}
	containerid := getStringFromBuffer(output)
	containerid = strings.TrimSpace(strings.Split(containerid, "\n")[0])
	return containerid, nil
}

func getAddrFromDockerHost(i string) (string, error) {
	// docker_aac_1.docker_default
	// If not docker, then just return the input
	if !strings.Contains(i, "docker") {
		//t.Logf("Host is not a docker container")
		return i, nil
	}

	// Assume format is `{container-name}.{NetworkMode}``
	hostparts := strings.Split(i, ".")
	// If not enough parts, just return the input
	if len(hostparts) < 2 {
		return i, nil
	}
	// Build up inspection format using network mode
	containerName := fmt.Sprintf("%s", hostparts[0])
	return getIPAddressForContainer(containerName)

}
