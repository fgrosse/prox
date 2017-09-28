package prox

import (
	"bufio"
	"context"
	"io"
	"net"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// A Server wraps an Executor to expose its functionality via a unix socket.
type Server struct {
	*Executor
	socketPath string
	listener   net.Listener
	logger     *zap.Logger
}

// socketMessage is the underlying message type that is passed between a prox
// Server and Client.
type socketMessage struct {
	Command string
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
// then starts the Executor. It is the callers responsibility to eventually call
// Server.Close() in order to close the unix socket connect.
func (s *Server) Run(ctx context.Context, pp []Process) error {
	if s.logger == nil {
		out := newOutput(pp, s.noColors, os.Stdout).nextColored("prox", s.proxLogColor)
		s.logger = NewLogger(out, s.debug)
	}

	var err error
	s.listener, err = net.Listen("unix", s.socketPath)
	if err != nil {
		s.logger.Error("Failed to open unix socket: " + err.Error())
		return errors.Wrap(err, "failed to open unix socket")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // always cancel context even if Executor finishes normally

	go s.acceptConnections(ctx)
	return s.Executor.Run(ctx, pp)
}

func (s *Server) acceptConnections(ctx context.Context) {
	var clientID int
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				// error due to closing listener
			default:
				s.logger.Error("Failed to accept connection", zap.Error(err))
			}
			return
		}

		clientID++
		connLog := s.logger.With(zap.Int("client_id", clientID))
		connLog.Info("Accepted new socket connection from prox client")
		go s.handleConnection(ctx, conn, connLog)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Always make sure the connection is closed when we return.

	go func() {
		<-ctx.Done()
		logger.Debug("Closing connection to prox client because context is done")
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

// Close closes the Servers listener.
func (s *Server) Close() error {
	if s.listener == nil {
		return nil
	}

	// TODO: improve closing (wait until listener loop has actually finished)

	s.logger.Info("Closing unix socket")
	return s.listener.Close() // TODO: this will cause an error message to be logged
}
