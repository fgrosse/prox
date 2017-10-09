package prox

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"sort"
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
	Args    []string
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
		s.logger = s.Executor.proxLogger(pp)
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
		connLog.Debug("Accepted new socket connection from prox client")
		go s.handleConnection(ctx, conn, connLog)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn, logger *zap.Logger) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // Always make sure the connection is closed when we return.

	go func() {
		<-ctx.Done()
		logger.Info("Closing connection to prox client")
		err := conn.Close()
		if err != nil && !isClosedConnectionError(err) {
			logger.Error("Failed to close connection", zap.Error(err))
		}
	}()

	msg, err := s.readMessage(conn)
	if errors.Cause(err) == io.EOF {
		logger.Error("Lost connection to prox client")
		return
	}
	if err != nil {
		logger.Error("Failed to read message from client", zap.Error(err))
		return
	}

	logger.Info("Received command from prox client", zap.Any("msg", msg))

	switch {
	case msg.Command == "LIST":
		err = s.handleListCommand(ctx, conn, msg, logger)
	case msg.Command == "TAIL":
		err = s.handleTailCommand(ctx, conn, msg, logger)
	case msg.Command == "EXIT":
		logger.Info("Prox client has closed the connection")
		return
	default:
		logger.Error("Unknown command from prox client", zap.Any("msg", msg))
		return
	}

	if err != nil && !isClosedConnectionError(err) {
		// TODO: send messages back to client
		logger.Error("prox client error", zap.Error(err))
	}
}

func (s *Server) readMessage(conn net.Conn) (socketMessage, error) {
	var msg socketMessage
	err := json.NewDecoder(conn).Decode(&msg)
	if err != nil {
		return msg, errors.Wrap(err, "failed to decode message")
	}

	return msg, nil
}

func (s *Server) handleListCommand(ctx context.Context, conn net.Conn, msg socketMessage, logger *zap.Logger) error {
	var names []string
	for name := range s.Executor.running {
		names = append(names, name)
	}

	sort.Strings(names)
	resp := make([]ProcessInfo, len(names))
	for i, name := range names {
		resp[i] = s.Executor.Info(name)
	}

	return json.NewEncoder(conn).Encode(resp)
}

func (s *Server) handleTailCommand(ctx context.Context, conn net.Conn, msg socketMessage, logger *zap.Logger) error {
	if len(msg.Args) == 0 {
		return errors.New("no arguments for tail provided")
	}

	var outputs []*multiWriter
	for _, name := range msg.Args {
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

	msg, err := s.readMessage(conn)
	if err != nil {
		return err
	}

	if msg.Command != "EXIT" {
		return errors.Errorf("expected EXIT command but got %q", msg.Command)
	}

	logger.Info("Client closed TAIL connection")
	return nil
}

// Close closes the Servers listener.
func (s *Server) Close() error {
	if s.listener == nil {
		return nil
	}

	// TODO: improve closing (wait until listener loop has actually finished)

	s.logger.Info("Closing unix socket")
	return s.listener.Close()
}

func isClosedConnectionError(err error) bool {
	return strings.Contains(err.Error(), "use of closed network connection")
}
