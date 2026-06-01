package docker

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Client struct {
	socketPath string
}

func New(socketPath string) *Client {
	return &Client{socketPath: socketPath}
}

func (c *Client) RunCommand(ctx context.Context, dir string, command string, logFn func(string)) error {
	parts := splitCommand(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
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

func (c *Client) TestConnection() error {
	cmd := exec.Command("docker", "info")
	if c.socketPath != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("DOCKER_HOST=unix://%s", c.socketPath))
	}
	return cmd.Run()
}

func splitCommand(command string) []string {
	var parts []string
	current := strings.Builder{}
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(command); i++ {
		c := command[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				current.WriteByte(c)
			}
			continue
		}
		if c == '\'' || c == '"' {
			inQuote = true
			quoteChar = c
			continue
		}
		if c == '&' && i+1 < len(command) && command[i+1] == '&' {
			part := strings.TrimSpace(current.String())
			if part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			i++ // skip second &
			continue
		}
		if c == ' ' {
			part := strings.TrimSpace(current.String())
			if part != "" {
				parts = append(parts, part)
			}
			current.Reset()
			continue
		}
		current.WriteByte(c)
	}
	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}
	return parts
}
