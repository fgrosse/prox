package prox

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/fgrosse/zaptest"
	"go.uber.org/zap"
)

type TestExecutor struct { // TODO: no need to export these types
	*Executor
	log   *zap.Logger
	Error error

	mu           sync.RWMutex
	executorDone bool
	cancel       func()
}

type TestReporter interface {
	Log(args ...interface{})
	Fatal(args ...interface{})
}

func TestNewExecutor(w io.Writer) *TestExecutor {
	e := &TestExecutor{Executor: NewExecutor(true)}
	e.output = w
	e.log = zaptest.LoggerWriter(w).Named("execut")

	return e
}

func (e *TestExecutor) Run(processes ...process) {
	ctx, cancel := context.WithCancel(context.Background())

	output := &output{writer: e.output}
	for _, p := range processes {
		if n := len(p.Name()); n > output.prefixLength {
			output.prefixLength = n
		}
	}

	e.mu.Lock()
	e.executorDone = false
	e.cancel = cancel
	e.mu.Unlock()

	e.log.Info("Executor starting")
	e.Error = e.Executor.run(ctx, output, processes, e.log)
	e.log.Info("Executor finished")

	e.mu.Lock()
	e.executorDone = true
	e.mu.Unlock()
}

func (e *TestExecutor) IsDone() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.executorDone
}

func (e *TestExecutor) Stop() {
	e.mu.Lock()
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.mu.Unlock()
}

type TestProcess struct {
	name   string // TODO: make settable from the outside
	PID    int
	Uptime time.Duration

	mu          sync.Mutex
	output      io.Writer
	started     bool
	interrupted bool

	finish chan chan bool // bool instead of struct{} for better readability of test code
	fail   chan chan bool

	interruptFinisher chan bool // to optionally block on Interrupt calls
}

func (p *TestProcess) Name() string {
	return p.name
}

func (p *TestProcess) Info() ProcessInfo {
	return ProcessInfo{
		PID:    p.PID,
		Uptime: p.Uptime,
	}
}

func (p *TestProcess) String() string {
	return p.Name()
}

func (p *TestProcess) Run(ctx context.Context, w io.Writer, logger *zap.Logger) error {
	p.mu.Lock()
	if p.started {
		return errors.New("started multiple times")
	}

	p.output = w
	p.started = true
	p.interrupted = false
	p.finish = make(chan chan bool)
	p.fail = make(chan chan bool)
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		logger.Debug("Context is done (interrupted)")
		if p.interruptFinisher != nil {
			logger.Debug("Executing interrupt finisher")
			<-p.interruptFinisher
		}
		p.mu.Lock()
		p.interrupted = true
		p.mu.Unlock()
		return ctx.Err()
	case c := <-p.finish:
		logger.Debug("Received finish signal")
		c <- true
		return nil
	case c := <-p.fail:
		logger.Debug("Received fail signal")
		c <- true
		return fmt.Errorf("TestProcess simulated a failure")
	}
}

func (p *TestProcess) Finish() {
	p.signal(p.finish)
}

func (p *TestProcess) Fail() {
	p.signal(p.fail)
}

func (p *TestProcess) signal(c chan chan bool) {
	syncChan := make(chan bool)
	select {
	case c <- syncChan:
		// sync with the TestProcess.Run goroutine
		<-syncChan
	default:
		// process is not running
	}
}

func (p *TestProcess) HasBeenStarted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.started
}

func (p *TestProcess) HasBeenInterrupted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.interrupted
}

func (p *TestProcess) ShouldBlockOnInterrupt() {
	p.interruptFinisher = make(chan bool)
}

func (p *TestProcess) FinishInterrupt() {
	p.interruptFinisher <- true
}

func (p *TestProcess) ShouldSay(t TestReporter, msg string) {
	_, err := p.output.Write([]byte(msg))
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewServerAndClient(t TestReporter, w io.Writer) (server *Server, client *Client, executor *TestExecutor, done func()) {
	executor = TestNewExecutor(w)
	client = &Client{logger: zaptest.LoggerWriter(w).Named("client")}
	server = &Server{
		Executor: executor.Executor,
		logger:   zaptest.LoggerWriter(w).Named("server"),
	}

	done = func() {
		log := zaptest.LoggerWriter(w).Named("test")
		log.Info("TestNewServerAndClient: done function was called")
		client.Close()
		server.Close()
		executor.Stop()
	}

	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		done()
		t.Fatal(err)
	}

	server.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		done()
		t.Fatal(err)
	}

	client.conn, err = net.Dial("tcp", server.listener.Addr().String())
	if err != nil {
		client.conn = nil
		done()
		t.Fatal(err)
	}

	client.buf = bufio.NewReader(client.conn)

	ctx := context.Background()
	go server.acceptConnections(ctx)

	return server, client, executor, done
}
