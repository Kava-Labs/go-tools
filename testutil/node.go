package testutil

import (
	"context"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// The node runner uses a fixed version of the kava image. The volume mount paths (for the config files) depend on the internals of this image.
const kavaImage = "kava/kava:v0.16.1"

type DockerNodeRunner struct {
	dockerClient *client.Client
	containerID  string
	nodeConfig   NodeConfig

	rootDir string
}

func NewDockerNodeRunner(nodeConfig NodeConfig) (*DockerNodeRunner, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerNodeRunner{
		dockerClient: cli,
		nodeConfig:   nodeConfig,

		rootDir: nodeConfig.TMConfig.BaseConfig.RootDir,
	}, nil
}

func (nr *DockerNodeRunner) Init() error {

	if err := WriteNodeConfig(nr.nodeConfig); err != nil {
		return err
	}

	ctx := context.Background()

	reader, err := nr.dockerClient.ImagePull(ctx, kavaImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader) // TODO

	grpcPort := nat.Port(nr.extractGRPCPort() + "/tcp")
	rpcPort := nat.Port(nr.extractRPCPort() + "/tcp")
	apiPort := nat.Port(nr.extractAPIPort() + "/tcp")

	resp, err := nr.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: kavaImage,
			Cmd:   []string{"kava", "start"},
			ExposedPorts: nat.PortSet{
				rpcPort:  struct{}{},
				grpcPort: struct{}{},
				apiPort:  struct{}{},
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				rpcPort: []nat.PortBinding{
					{
						HostIP:   "localhost",
						HostPort: rpcPort.Port(),
					},
				},
				grpcPort: []nat.PortBinding{
					{
						HostIP:   "localhost",
						HostPort: grpcPort.Port(),
					},
				},
				apiPort: []nat.PortBinding{
					{
						HostIP:   "localhost",
						HostPort: apiPort.Port(),
					},
				},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: nr.rootDir,
					Target: "/root/.kava",
				},
			},
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return err
	}

	nr.containerID = resp.ID

	return nil
}

func (nr *DockerNodeRunner) Start() error {
	ctx := context.Background()

	if err := nr.dockerClient.ContainerStart(ctx, nr.containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	if err := nr.waitForNextBlock(); err != nil {
		return err
	}

	return nil
}

func (nr *DockerNodeRunner) waitForNextBlock() error {
	// TODO wait until a block is produced, or the container exits
	// If container exits, the container logs may be useful for the user.
	time.Sleep(3 * time.Second)
	return nil
}

func (nr *DockerNodeRunner) Stop() error {
	timeout := time.Minute
	return nr.dockerClient.ContainerStop(context.Background(), nr.containerID, &timeout)
}

func (nr *DockerNodeRunner) Cleanup() error {
	if err := nr.Stop(); err != nil { // TODO what happens if it's already stopped?
		return err
	}

	err := nr.dockerClient.ContainerRemove(context.Background(), nr.containerID, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// if err := os.RemoveAll(nr.rootDir); err != nil {
	// 	return err
	// }

	return nil
}

func (nr *DockerNodeRunner) RPCAddress() string {
	return "http://localhost:" + nr.extractRPCPort()
}
func (nr *DockerNodeRunner) GRPCAddress() string {
	return "localhost:" + nr.extractGRPCPort()
}
func (nr *DockerNodeRunner) APIAddress() string {
	return "http://localhost:" + nr.extractAPIPort()
}

func (nr *DockerNodeRunner) extractGRPCPort() string {
	port, err := parsePortFromAddress(nr.nodeConfig.AppConfig.GRPC.Address)
	if err != nil {
		return ""
	}
	return port
}

func (nr *DockerNodeRunner) extractRPCPort() string {
	port, err := parsePortFromAddress(nr.nodeConfig.TMConfig.RPC.ListenAddress)
	if err != nil {
		return ""
	}
	return port
}

func (nr *DockerNodeRunner) extractAPIPort() string {
	port, err := parsePortFromAddress(nr.nodeConfig.AppConfig.API.Address)
	if err != nil {
		return ""
	}
	return port
}

func parsePortFromAddress(address string) (string, error) {
	const urlParseErrMsg = "too many colons in address"
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		if !strings.Contains(err.Error(), urlParseErrMsg) {
			return "", err
		}
		uri, err := url.ParseRequestURI(address)
		if err != nil {
			return "", err
		}
		return uri.Port(), nil

	}
	return port, nil
}
