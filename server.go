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

type Server struct {
	Executor   *Executor
	socketPath string
	newLogger  func([]Process) *zap.Logger
}

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
	go closeWhenDone(ctx, "socket listener", l, logger)

	for {
		conn, err := l.Accept()
		if err != nil {
			// TODO do not log error due to closed listener
			logger.Error("Failed to accept connection", zap.Error(err))
		}

		go s.handleConnection(ctx, conn, logger)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go closeWhenDone(ctx, "connection", conn, logger)

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

func closeWhenDone(ctx context.Context, name string, c io.Closer, logger *zap.Logger) {
	<-ctx.Done()
	err := c.Close()
	if err != nil {
		logger.Error("Failed to close "+name, zap.Error(err))
	}
}
