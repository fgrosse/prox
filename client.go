package prox

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// A Client connects to a Server via a unix socket to provide advanced
// functionality on a running prox server.
type Client struct {
	conn   net.Conn
	logger *zap.Logger
	buf    *bufio.Reader
}

// NewClient creates a new prox Client and immediately connects it to a prox
// Server via a unix socket. It is the callers responsibility to eventually
// close the client to release the underlying socket connection.
func NewClient(socketPath string, debug bool) (*Client, error) {
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to prox socket")
	}

	return &Client{
		conn:   c,
		logger: NewLogger(os.Stderr, debug),
		buf:    bufio.NewReader(c),
	}, nil
}

func (c *Client) Tail(ctx context.Context, processNames []string, output io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	err := c.write("TAIL " + strings.Join(processNames, " "))
	if err != nil {
		return err
	}

	lines := make(chan string)
	go func() {
		for {
			line, err := c.readLine()
			if err != nil {
				if err == io.EOF {
					c.logger.Info("Server closed connection")
				} else {
					c.logger.Error(err.Error())
				}

				// Cancel the context to return from the loop below.
				cancel()
				return
			}
			lines <- line
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case l := <-lines:
			_, err = fmt.Fprint(output, l)
			if err != nil {
				return err
			}
		}
	}
}

func (c *Client) write(msg string) error {
	_, err := fmt.Fprintln(c.conn, msg)
	return err
}

func (c *Client) readLine() (string, error) {
	return c.buf.ReadString('\n')
}

// Close closes the socket connection to the prox server.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	var _ = c.write("EXIT")
	return c.conn.Close()
}
