package docker

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Client struct {
	socketPath string
}

func New(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

func (c *Client) RunCommand(ctx context.Context, dir string, command string, logFn func(string)) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir

	env := os.Environ()
	if c.socketPath != "" {
		env = append(env, fmt.Sprintf("DOCKER_HOST=unix://%s", c.socketPath))
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if logFn != nil {
			logFn(line)
		}
	}

	errScanner := bufio.NewScanner(stderr)
	errScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for errScanner.Scan() {
		line := errScanner.Text()
		if logFn != nil {
			logFn(line)
		}
	}

	err = cmd.Wait()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out: %w", err)
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (c *Client) StreamCommand(ctx context.Context, dir string, command string, lines chan<- string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir

	env := os.Environ()
	if c.socketPath != "" {
		env = append(env, fmt.Sprintf("DOCKER_HOST=unix://%s", c.socketPath))
	}
	cmd.Env = env

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case lines <- scanner.Text():
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		}
	}

	errScanner := bufio.NewScanner(stderr)
	errScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for errScanner.Scan() {
		select {
		case lines <- errScanner.Text():
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		}
	}

	return cmd.Wait()
}

func (c *Client) TestConnection() error {
	cmd := exec.Command("docker", "info")
	if c.socketPath != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("DOCKER_HOST=unix://%s", c.socketPath))
	}
	return cmd.Run()
}
