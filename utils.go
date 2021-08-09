package jai_docker

// This code is a wrapper around Docker functionality, in the hopes that it won't
// be that much of a pain to set up on windows. We will have to see though. It's
// been cobbled together using a random assortment of blogs and auto-generated
// Docker documentation.

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	ubuntuImage   = "jai-docker/ubuntu"
	compilerImage = "jai-docker/compiler"
)

var (
	ProjectDir = func() string {
		_, filename, _, _ := runtime.Caller(0)
		abs, err := filepath.Abs(filename)
		CheckErr(err)

		return path.Dir(abs)
	}()
)

func RunCmd(ctx context.Context, binary string, args []string) int {
	begin := time.Now()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	CheckErr(err)

	buildImage(cli, ctx, "ubuntu.Dockerfile", ubuntuImage, false)
	buildImage(cli, ctx, "Dockerfile", compilerImage, false)

	cwd, err := os.Getwd()
	CheckErr(err)

	// Build container
	cmdBinary := []string{binary}
	cmdBinary = append(cmdBinary, args...)
	containerConfig := container.Config{
		Image:      compilerImage,
		Cmd:        cmdBinary,
		WorkingDir: "/cwd",
	}
	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: cwd,
				Target: "/cwd",
			},
			{
				Type:   mount.TypeBind,
				Source: ProjectDir,
				Target: "/root/jai-docker",
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, "")
	if resp.Warnings != nil && len(resp.Warnings) != 0 {
		fmt.Printf("%#v\n", resp.Warnings)
	}

	if err != nil {
		buildImage(cli, ctx, "ubuntu.Dockerfile", ubuntuImage, true)
		buildImage(cli, ctx, "Dockerfile", compilerImage, true)

		resp, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, "")
		if resp.Warnings != nil && len(resp.Warnings) != 0 {
			fmt.Printf("%#v\n", resp.Warnings)
		}
	}
	CheckErr(err)

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	CheckErr(err)

	compileBegin := time.Now()
	fmt.Printf("docker stuff took %v seconds\n\n", compileBegin.Sub(begin).Seconds())

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	var commandStatus int64
	select {
	case err := <-errCh:
		CheckErr(err)
	case resp := <-statusCh:
		commandStatus = resp.StatusCode
	}

	logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	out, err := cli.ContainerLogs(ctx, resp.ID, logOptions)
	CheckErr(err)

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}
	err = cli.ContainerRemove(ctx, resp.ID, removeOptions)
	CheckErr(err)

	return int(commandStatus)
}

func needBuild(dockerfilePath, imageName string) bool {
	escapedName := strings.Replace(imageName, string(os.PathSeparator), ".", -1)
	placeholder := filepath.Join(ProjectDir, ".image-"+escapedName)
	dockerfilePath = filepath.Join(ProjectDir, dockerfilePath)

	dockerfileStat, err := os.Stat(dockerfilePath)
	CheckErr(err)
	placeholderStat, err := os.Stat(placeholder)

	if os.IsNotExist(err) {
		file, err := os.Create(placeholder)
		CheckErr(err)
		file.Close()
		return true
	} else {
		currentTime := time.Now().Local()
		err = os.Chtimes(placeholder, currentTime, currentTime)
		CheckErr(err)

		return dockerfileStat.ModTime().After(placeholderStat.ModTime())
	}
}

func buildImage(cli *client.Client, ctx context.Context, dockerfilePath, imageName string, forceBuild bool) {
	if !needBuild(dockerfilePath, imageName) && !forceBuild {
		return
	}

	cmd := exec.Command("docker", "build", "--platform=linux/amd64", "-f", filepath.Join(ProjectDir, dockerfilePath), "--tag", imageName, ".")

	stdout, err := cmd.StdoutPipe()
	CheckErr(err)
	stderr, err := cmd.StderrPipe()
	CheckErr(err)

	finished := make(chan bool)
	ioCopy := func(w io.Writer, r io.Reader) {
		io.Copy(w, r)
		finished <- true
	}

	go ioCopy(os.Stdout, stdout)
	go ioCopy(os.Stderr, stderr)
	err = cmd.Run()
	<-finished
	<-finished

	CheckErr(err)
}

func CheckErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
