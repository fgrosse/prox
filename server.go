package prox

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// A Server wraps an Executor to expose its functionality via a unix socket.
type Server struct {
	*Executor
	socketPath string
}

// NewExecutorServer creates a new Server. This function does not start the
// Executor nor does it listen on the unix socket just yet. To start the Server
// and Executor the Server.Run(…) function must be used.
func NewExecutorServer(socketPath string, debug bool) *Server {
	return &Server{
		Executor:   NewExecutor(debug),
		socketPath: socketPath,
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

	out := newOutput(pp, s.noColors).nextColored("prox", s.proxLogColor)
	logger := NewLogger(out, s.debug)

	go s.acceptConnections(ctx, l, logger)
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

	var clientID int
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

		clientID++
		connLog := logger.With(zap.Int("client_id", clientID))
		connLog.Info("Accepted new socket connection from prox client")
		go s.handleConnection(ctx, conn, connLog)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Always make sure the connection is closed when we return.

	go func() {
		<-ctx.Done()
		logger.Debug("Closing connection to prox client")
		err := conn.Close()
		if err != nil {
			logger.Error("Failed to close connection", zap.Error(err))
		}
	}()

	r := bufio.NewReader(conn)
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
	case strings.HasPrefix(command, "TAIL "):
		args := strings.Fields(command)
		err = s.handleTailRPC(ctx, conn, args[1:], logger)
	case command == "EXIT":
		logger.Info("Prox client has closed the connection")
		return
	default:
		logger.Error("Unknown command from prox client", zap.String("command", command))
	}

	if err != nil {
		// TODO: send messages back to client
		logger.Error("prox client error", zap.Error(err))
	}
}

func (s *Server) handleTailRPC(ctx context.Context, conn net.Conn, args []string, logger *zap.Logger) error {
	if len(args) == 0 {
		return errors.New("no arguments for tail provided")
	}

	var outputs []*processOutput
	for _, name := range args {
		o, ok := s.Executor.outputs[name]
		if !ok {
			return errors.Errorf("cannot tail unknown process %q", name)
		}

		o.AddWriter(conn)
		outputs = append(outputs, o)
	}

	defer func() {
		for _, o := range outputs {
			o.RemoveWriter(conn)
		}
	}()

	r := bufio.NewReader(conn)
	command, err := r.ReadString('\n')
	if err != nil {
		return err
	}

	command = strings.TrimSpace(command)
	if command != "EXIT" {
		return errors.Errorf("expected EXIT command but got %q", command)
	}

	return nil
}
