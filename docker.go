package main

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

var Context context.Context = context.Background()

type ExecResult struct {
	StdOut   string
	StdErr   string
	ExitCode int
}

type dockerDirFileContent struct {
	FileName      string
	FilePath      string
	FileExtension string
	FileContents  []byte
}

func getDockerDirContents(dirPath string, fileExtension string) ([]dockerDirFileContent, error) {
	DockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	reader, _, err := DockerClient.CopyFromContainer(Context, DockerContainerName, dirPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)

	var Files []dockerDirFileContent
Loop:
	for {
		Header, err := tarReader.Next()
		switch {
		case err != nil:
			break Loop
		case Header.FileInfo().IsDir():
			continue
		case fileExtension != "" && filepath.Ext(Header.FileInfo().Name()) != fileExtension:
			continue
		}
		File := dockerDirFileContent{
			FileName:      Header.FileInfo().Name(),
			FilePath:      "/" + Header.Name,
			FileExtension: filepath.Ext(Header.FileInfo().Name()),
		}
		fileContents, err := io.ReadAll(tarReader)
		if err == nil {
			File.FileContents = fileContents
		}

		Files = append(Files, File)
	}
	return Files, nil
}

func Exec(Context context.Context, ContainerID string, Commands []string) (ExecResult, error) {
	// DockerClient, err := client.NewEnvClient()
	var ExecResult ExecResult

	DockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		// return types.IDResponse{}, err
		return ExecResult, err
	}
	defer DockerClient.Close()

	Config := types.ExecConfig{
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          Commands,
	}

	IDResponse, err := DockerClient.ContainerExecCreate(Context, ContainerID, Config)
	if err != nil {
		return ExecResult, err
	}

	return InspectExecResp(Context, IDResponse.ID)

}

func InspectExecResp(Context context.Context, ExecID string) (ExecResult, error) {
	var ExecResult ExecResult
	// DockerClient, err := client.NewEnvClient()
	DockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return ExecResult, err
	}
	defer DockerClient.Close()

	Resonse, err := DockerClient.ContainerExecAttach(Context, ExecID, types.ExecStartCheck{})
	if err != nil {
		return ExecResult, err
	}
	defer Resonse.Close()

	// read the output
	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, Resonse.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return ExecResult, err
		}
		break

	case <-Context.Done():
		return ExecResult, Context.Err()
	}

	stdout, err := io.ReadAll(&outBuf)
	if err != nil {
		return ExecResult, err
	}
	stderr, err := io.ReadAll(&errBuf)
	if err != nil {
		return ExecResult, err
	}

	res, err := DockerClient.ContainerExecInspect(Context, ExecID)
	if err != nil {
		return ExecResult, err
	}

	ExecResult.ExitCode = res.ExitCode
	ExecResult.StdOut = string(stdout)
	ExecResult.StdErr = string(stderr)
	return ExecResult, nil
}
