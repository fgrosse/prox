package prox

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// A Server wraps an Executor to expose its functionality via a unix socket.
type Server struct {
	Executor   *Executor
	socketPath string
	newLogger  func([]Process) *zap.Logger
}

// NewExecutorServer creates a new Server. This function does not start the
// Executor nor does it listen on the unix socket just yet. To start the Server
// and Executor the Server.Run(…) function must be used.
func NewExecutorServer(socketPath string, debug bool) *Server {
	return &Server{
		Executor:   NewExecutor(debug),
		socketPath: socketPath,
		newLogger: func(pp []Process) *zap.Logger {
			out := newOutput(pp).nextColored("prox", colorWhite)
			return NewLogger(out, debug)
		},
	}
}

// Run opens a unix socket using the path that was passed via NewExecutor and
// then starts the Executor. The socket is closed automatically when the
// Executor or the context is done.
func (s *Server) Run(ctx context.Context, pp []Process) error {
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return errors.Wrap(err, "failed to open unix socket")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // always cancel context even if Executor finishes normally

	go s.acceptConnections(ctx, l, s.newLogger(pp))
	return s.Executor.Run(ctx, pp)
}

func (s *Server) acceptConnections(ctx context.Context, l net.Listener, logger *zap.Logger) {
	go func() {
		<-ctx.Done()

		// Closing the listener will unblock the Accept call in the loop below.
		logger.Info("Closing socket listener")
		err := l.Close()
		if err != nil {
			logger.Error("Failed to close socket listener", zap.Error(err))
		}

		// Any already opened connections are closed by the handler if the
		// context is done.
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// error due to closing listener
			default:
				logger.Error("Failed to accept connection", zap.Error(err))
			}
			return
		}

		go s.handleConnection(ctx, conn, logger)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Always make sure the connection is closed when we return.

	go func() {
		<-ctx.Done()
		err := conn.Close()
		if err != nil {
			logger.Error("Failed to close connection", zap.Error(err))
		}
	}()

	r := bufio.NewReader(conn)
	for {
		command, err := r.ReadString('\n')
		if err == io.EOF {
			logger.Warn("Lost connection to prox client")
			return
		}

		if err != nil {
			logger.Error("Failed to read line from client", zap.Error(err))
		}

		command = strings.TrimSpace(command)
		logger.Debug("Received command from prox client", zap.String("command", command))

		switch {
		case strings.HasPrefix(command, "CONNECT "):
			args := strings.Fields(command)
			s.handleConnectCommand(ctx, conn, args[1:], logger)
		case command == "EXIT":
			logger.Info("Prox client has closed the connection")
			return
		default:
			logger.Error("Unknown command from prox client", zap.String("command", command))
		}
	}
}

func (s *Server) handleConnectCommand(ctx context.Context, conn net.Conn, args []string, logger *zap.Logger) {
	// TODO: how to write messages as well as read input
	var i int
	for {
		i++
		logger.Info("Sending next message")
		_, err := fmt.Fprintln(conn, "Hello", i, ":", args)
		if err != nil {
			logger.Error("Failed to send message", zap.Error(err))
			return
		}
		time.Sleep(time.Second)
	}
}
