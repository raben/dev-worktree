package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/term"
)

// Exec runs a command inside a running container and returns the combined
// stdout/stderr output.
func (c *Client) Exec(ctx context.Context, containerID string, cmd []string) (string, error) {
	execID, err := c.docker.ContainerExecCreate(ctx, containerID, containertypes.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("exec create: %w", err)
	}

	resp, err := c.docker.ContainerExecAttach(ctx, execID.ID, containertypes.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, resp.Reader); err != nil {
		return "", fmt.Errorf("read exec output: %w", err)
	}

	inspect, err := c.docker.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return buf.String(), fmt.Errorf("exec inspect: %w", err)
	}
	if inspect.ExitCode != 0 {
		return buf.String(), fmt.Errorf("exec exited with code %d", inspect.ExitCode)
	}

	return buf.String(), nil
}

// ExecInteractive runs a command inside a container with stdin/stdout attached,
// suitable for interactive sessions like shells or CLI tools.
func (c *Client) ExecInteractive(ctx context.Context, containerID string, cmd []string) error {
	execID, err := c.docker.ContainerExecCreate(ctx, containerID, containertypes.ExecOptions{
		Cmd:          cmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	})
	if err != nil {
		return fmt.Errorf("exec create: %w", err)
	}

	resp, err := c.docker.ContainerExecAttach(ctx, execID.ID, containertypes.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		return fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	// Put the local terminal into raw mode so keystrokes are forwarded directly.
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("set terminal raw mode: %w", err)
		}
		defer term.Restore(fd, oldState)
	}

	// Handle terminal resize signals.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
	}()

	go func() {
		for range sigCh {
			c.resizeExecTTY(ctx, execID.ID)
		}
	}()

	// Set initial terminal size.
	c.resizeExecTTY(ctx, execID.ID)

	// Stream I/O between the local terminal and the container.
	outputDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(os.Stdout, resp.Reader)
		outputDone <- err
	}()

	inputDone := make(chan error, 1)
	go func() {
		_, err := io.Copy(resp.Conn, os.Stdin)
		inputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return fmt.Errorf("copy output: %w", err)
		}
	case <-inputDone:
		// Wait briefly for remaining output after stdin closes.
		select {
		case err := <-outputDone:
			if err != nil {
				return fmt.Errorf("copy output: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// resizeExecTTY updates the exec session's TTY size to match the local terminal.
func (c *Client) resizeExecTTY(ctx context.Context, execID string) {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		return
	}
	_ = c.docker.ContainerExecResize(ctx, execID, containertypes.ResizeOptions{
		Width:  uint(width),
		Height: uint(height),
	})
}
