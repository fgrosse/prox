package prox

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/fgrosse/zaptest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("shellProcess", func() {
	Describe("Name", func() {
		It("should return the shellProcess.name", func() {
			p := NewShellProcess("foo", "echo foo")
			Expect(p.Name()).To(Equal("foo"))
		})
	})

	Describe("Run", func() {
		It("should start the command line as new process", func() {
			// In this test we want to confirm that a sub process is actually
			// started. In order to do so we execute the test binary itself and
			// run a specifically prepared test case defined below in this file.
			//
			// In order to know that the process was actually started we let it
			// expose a local HTTP server on a given port and then try to
			// connect to it to prove the process is running.
			port := freePort(GinkgoT())
			url := fmt.Sprintf("http://localhost:%d", port)
			log := zaptest.LoggerWriter(GinkgoWriter)
			p := &shellProcess{
				name:   "test",
				script: fmt.Sprintf("%s -test.run=^TestHelperProcess$ -- http %d", os.Args[0], port),
				env:    []string{"GO_WANT_HELPER_PROCESS=1"},
				logger: log.Named("http_server"),
			}

			httpRequest := func() int {
				log := log.Named("http_client")

				log.Debug("Making HTTP request", zap.String("url", url))
				resp, err := http.Get(url)
				if err != nil {
					log.Error("Failed to contact HTTP server", zap.String("error", err.Error()))
					return 0
				}

				resp.Body.Close()
				log.Info("Received HTTP response", zap.String("status", resp.Status))
				return resp.StatusCode
			}

			go func() {
				defer GinkgoRecover()
				p.Run()
			}()

			Eventually(httpRequest).Should(Equal(http.StatusOK), "should eventually answer with status code 200")
		})

		PIt("should send its stdout to the configured writer")
		PIt("should send its stderr to the configured writer")
		PIt("should use the given environment")
		PIt("should replace environment variables in the script")
		PIt("should parse and use environment variables at the beginning of the script")
		PIt("should cancel process execution if the given context is canceled")
	})

	Describe("Interrupt", func() {
		PIt("should stop the running process")
	})
})

// TestHelperProcess is used to test sub processes in unit tests.
// This technique mirrors the approach presented in Mitchell Hashimotos talk
// about "Advanced Testing with Go".
//
// See https://speakerdeck.com/mitchellh/advanced-testing-with-go
func TestHelperProcess(t *testing.T) {
	if v := os.Getenv("GO_WANT_HELPER_PROCESS"); v != "1" {
		t.Logf("Skipping helper process (GO_WANT_HELPER_PROCESS: %q)", v)
		return
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}

		args = args[1:]
	}

	if len(args) == 0 {
		t.Fatal("Too few arguments (pass after --)")
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "http":
		httpServerProcess(t, args)
		os.Exit(0)
	default:
		t.Fatalf("Unknown command %q", cmd)
	}
}

func httpServerProcess(t *testing.T, args []string) {
	if len(args) != 1 {
		t.Fatal("httpserve command needs exactly one argument: the HTTP port to listen on")
	}

	addr := "localhost:" + args[0]
	fmt.Println("Starting HTTP server at " + addr)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello World!")
	})

	http.ListenAndServe(addr, handler)
}

// freePort asks the kernel for a free open port that is ready to be used.
func freePort(t GinkgoTInterface) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
