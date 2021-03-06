package prox

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// A Client connects to a Server via a unix socket to provide access to a
// running prox server.
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

// List fetches a list of running processes from the server and prints it via
// the given output.
func (c *Client) List(ctx context.Context, output io.Writer) error {
	err := c.sendMessage(socketMessage{Command: "LIST"})
	if err != nil {
		return err
	}

	var resp []ProcessInfo
	err = json.NewDecoder(c.conn).Decode(&resp)
	if err != nil {
		return errors.Wrap(err, "failed to decode server response")
	}

	w := tabwriter.NewWriter(output, 8, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPID\tUPTIME")

	for _, inf := range resp {
		fmt.Fprintln(w, fmt.Sprintf(
			"%s\t%v\t%v",
			inf.Name, inf.PID, inf.Uptime.Round(time.Second)),
		)
	}

	return w.Flush()
}

// Tail requests and "follows" the logs for a set of processes from a server and
// prints them to the output. This function blocks until the context is done or
// the connection to the server is closed by either side.
func (c *Client) Tail(ctx context.Context, processNames []string, output io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)

	err := c.sendMessage(socketMessage{Command: "TAIL", Args: processNames})
	if err != nil {
		cancel()
		return err
	}

	lines := make(chan string)
	go func() {
		for {
			line, err := c.buf.ReadString('\n')
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

func (c *Client) sendMessage(msg socketMessage) error {
	return json.NewEncoder(c.conn).Encode(msg)
}

// Close closes the socket connection to the prox server.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}

	var _ = c.sendMessage(socketMessage{Command: "EXIT"})
	return c.conn.Close()
}
