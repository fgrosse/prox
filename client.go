package prox

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/pkg/errors"
)

type Client struct {
	conn net.Conn
	buf  *bufio.Reader
}

func NewClient(socketPath string) (*Client, error) {
	c, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to prox socket")
	}

	return &Client{
		conn: c,
		buf:  bufio.NewReader(c),
	}, nil
}

func (c *Client) Connect(ctx context.Context, processNames []string, output io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	err := c.write("CONNECT " + strings.Join(processNames, " "))
	if err != nil {
		return err
	}

	lines := make(chan string)
	go func() {
		for {
			// TODO: improve and test this mess
			line, err := c.readLine()
			// TODO: err == io.EOF if server closes connection
			if err != nil {
				log.Println("ERROR: ", err) // TODO: logging
				cancel()
				return
			}
			lines <- line
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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

func (c *Client) Close() error {
	var _ = c.write("EXIT")
	return c.conn.Close()
}
