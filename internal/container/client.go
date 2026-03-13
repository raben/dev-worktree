package container

import (
	"github.com/docker/docker/client"
)

// Client wraps the Docker Engine API client.
type Client struct {
	docker *client.Client
}

// NewClient creates a new Docker client using the default environment settings.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{docker: cli}, nil
}

// Close closes the Docker client connection.
func (c *Client) Close() error {
	return c.docker.Close()
}
