package testutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/server/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	tmconfig "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	tmtypes "github.com/tendermint/tendermint/types"
)

type DockerNodeRunner struct {
	dockerClient *client.Client
	containerID  string
	rootDir      string
}

func NewDockerNodeRunner(
	appConfig *config.Config,
	tmConfig *tmconfig.Config,
	privValidator *privval.FilePV,
	nodeKey *p2p.NodeKey,
	genesisDoc tmtypes.GenesisDoc,
) (*DockerNodeRunner, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	if err := WriteNodeConfig(appConfig, tmConfig, privValidator, nodeKey, genesisDoc); err != nil { // TODO maybe move into Setup()
		return nil, err
	}

	return &DockerNodeRunner{
		dockerClient: cli,
		rootDir:      tmConfig.BaseConfig.RootDir,
	}, nil
}

func (nr *DockerNodeRunner) Start() error {

	ctx := context.Background()

	image := "kava/kava:v0.16.1"
	reader, err := nr.dockerClient.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer reader.Close()
	io.Copy(os.Stdout, reader)

	resp, err := nr.dockerClient.ContainerCreate(
		ctx,
		&container.Config{
			Image: image,
			Cmd:   []string{"kava", "start"},
			Tty:   false,
		},
		&container.HostConfig{
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

	if err := nr.dockerClient.ContainerStart(ctx, nr.containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	fmt.Println("waiting") // TODO wait for startup or exit
	time.Sleep(5 * time.Second)
	statusCh, errCh := nr.dockerClient.ContainerWait(ctx, nr.containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
	}

	out, err := nr.dockerClient.ContainerLogs(ctx, nr.containerID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return err
	}
	defer out.Close()

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return nil
}

func (nr *DockerNodeRunner) Stop() error {
	return fmt.Errorf("TODO")
}

func (nr *DockerNodeRunner) Cleanup() error {
	ctx := context.Background()
	err := nr.dockerClient.ContainerRemove(ctx, nr.containerID, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	// TODO
	// if err := os.RemoveAll(nr.rootDir); err != nil {
	// 	return err
	// }

	return nil
}
