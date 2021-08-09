package main

// This code is a wrapper around Docker functionality, in the hopes that it won't
// be that much of a pain to set up on windows. We will have to see though. It's
// been cobbled together using a random assortment of blogs and auto-generated
// Docker documentation.

import (
	"bufio"
	_ "bytes"
	"context"
	"encoding/json"
	"errors"
	_ "flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	ubuntuImage   = "jai-docker/ubuntu"
	compilerImage = "jai-docker/compiler"
)

var (
	projectDir = func() string {
		_, filename, _, _ := runtime.Caller(0)
		abs, err := filepath.Abs(filename)
		checkErr(err)

		return path.Dir(abs)
	}()
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	checkErr(err)

	commandStatus := runCmd(cli, ctx, os.Args[1:])
	if commandStatus != 0 {
		os.Exit(commandStatus)
	}

	os.Exit(0)
}

func runCmd(cli *client.Client, ctx context.Context, args []string) int {
	begin := time.Now()
	tarOptions := archive.TarOptions{IncludeFiles: []string{}}
	buildImage(cli, ctx, "ubuntu.Dockerfile", ubuntuImage, &tarOptions, false)
	buildImage(cli, ctx, "Dockerfile", compilerImage, &tarOptions, false)

	// Build container
	cmdBinary := []string{"/root/jai-docker/jai/bin/jai-linux"}
	cmdBinary = append(cmdBinary, args...)
	containerConfig := container.Config{
		Image:      compilerImage,
		Cmd:        cmdBinary,
		WorkingDir: "/root/jai-docker",
	}
	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: projectDir,
				Target: "/root/jai-docker",
			},
		},
	}

	resp, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, "")
	if resp.Warnings != nil && len(resp.Warnings) != 0 {
		fmt.Printf("%#v\n", resp.Warnings)
	}

	if err != nil {
		tarOptions := archive.TarOptions{IncludeFiles: []string{}}
		buildImage(cli, ctx, "ubuntu.Dockerfile", ubuntuImage, &tarOptions, true)
		buildImage(cli, ctx, "Dockerfile", compilerImage, &tarOptions, true)

		resp, err = cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, "")
		if resp.Warnings != nil && len(resp.Warnings) != 0 {
			fmt.Printf("%#v\n", resp.Warnings)
		}
	}
	checkErr(err)

	err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	checkErr(err)

	compileBegin := time.Now()
	fmt.Printf("docker stuff took %v seconds\n", compileBegin.Sub(begin).Seconds())

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	var commandStatus int64
	select {
	case err := <-errCh:
		checkErr(err)
	case resp := <-statusCh:
		commandStatus = resp.StatusCode
	}

	logOptions := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	out, err := cli.ContainerLogs(ctx, resp.ID, logOptions)
	checkErr(err)

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}
	err = cli.ContainerRemove(ctx, resp.ID, removeOptions)
	checkErr(err)

	return int(commandStatus)
}

func needBuild(dockerfilePath, imageName string) bool {
	escapedName := strings.Replace(imageName, string(os.PathSeparator), ".", -1)
	placeholder := ".image-" + escapedName
	dockerfilePath = filepath.Join(projectDir, dockerfilePath)

	dockerfileStat, err := os.Stat(dockerfilePath)
	checkErr(err)
	placeholderStat, err := os.Stat(placeholder)

	if os.IsNotExist(err) {
		file, err := os.Create(placeholder)
		checkErr(err)
		file.Close()
		return true
	} else {
		currentTime := time.Now().Local()
		err = os.Chtimes(placeholder, currentTime, currentTime)
		checkErr(err)

		return dockerfileStat.ModTime().After(placeholderStat.ModTime())
	}
}

func buildImage(cli *client.Client, ctx context.Context, dockerfilePath, imageName string, tarOptions *archive.TarOptions, forceBuild bool) {
	if !needBuild(dockerfilePath, imageName) && !forceBuild {
		return
	}

	tar, err := archive.TarWithOptions(projectDir, tarOptions)
	checkErr(err)

	// Build image
	opts := types.ImageBuildOptions{
		Dockerfile: dockerfilePath,
		Tags:       []string{imageName},
		Remove:     true,
	}
	res, err := cli.ImageBuild(ctx, tar, opts)
	checkErr(err)
	err = printImageLogs(res.Body)
	checkErr(err)
	res.Body.Close()
}

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func printImageLogs(rd io.Reader) error {
	type ErrorDetail struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}

	type ErrorLine struct {
		Error       string      `json:"error"`
		ErrorDetail ErrorDetail `json:"errorDetail"`
	}

	type Aux struct {
		Id string `json:"ID"`
	}

	type StreamLine struct {
		Error       string      `json:"error"`
		ErrorDetail ErrorDetail `json:"errorDetail"`
		Value       string      `json:"stream"`
		Status      string      `json:"status"`
		Id          string      `json:"id"`
		Aux         Aux         `json:"aux"`
	}

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		buf := scanner.Bytes()

		var line StreamLine
		err := json.Unmarshal(buf, &line)
		if line.Value == "" && line.Aux.Id != "" {
			break
		}

		if line.Status != "" { // TODO what should we do here?
			continue
		}

		if err != nil || line.Value == "" {
			errLine := ErrorLine{Error: line.Error, ErrorDetail: line.ErrorDetail}
			return errors.New(fmt.Sprintf("%#v", errLine))
		}

		fmt.Print(line.Value)
	}

	return scanner.Err()
}
