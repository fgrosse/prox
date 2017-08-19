package prox

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fgrosse/zaptest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
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
		var log *zap.Logger

		BeforeEach(func() {
			log = zaptest.LoggerWriter(GinkgoWriter)
		})

		It("should start the command line as new process", func() {
			// In this test we want to confirm that a sub process is actually
			// started. In order to do so we execute the test binary itself and
			// run a specifically prepared test case defined below in this file.
			//
			// In order to know that the process was actually started we let it
			// expose a local HTTP server on a given port and then try to
			// connect to it to prove the process is running.
			port := freePort(GinkgoT())
			url := "http://localhost:" + port
			p := &shellProcess{
				name:   "test",
				script: testProcessScript("http", port),
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

		It("should send its stdout to the configured writer", func() {
			w := NewBuffer()
			p := &shellProcess{
				name:   "test",
				script: testProcessScript("echo", "hello", "world"),
				env:    []string{"GO_WANT_HELPER_PROCESS=1"},
				logger: log.Named("process"),
				writer: w,
			}

			go func() {
				defer GinkgoRecover()
				p.Run()
			}()

			Eventually(w).Should(Say(`hello`))
			Eventually(w).Should(Say(`world`))
		})

		It("should send its stderr to the configured writer", func() {
			w := NewBuffer()
			p := &shellProcess{
				name:   "test",
				script: testProcessScript("echo", "-stderr", "hello", "world"),
				env:    []string{"GO_WANT_HELPER_PROCESS=1"},
				logger: log.Named("process"),
				writer: w,
			}

			go func() {
				defer GinkgoRecover()
				p.Run()
			}()

			Eventually(w).Should(Say(`hello`))
			Eventually(w).Should(Say(`world`))
			log.Info("Full message", zap.String("contents", string(w.Contents())))
		})

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
		fmt.Println("Too few arguments (pass after --)")
		os.Exit(1)
	}

	cmd, args := args[0], args[1:]
	switch cmd {
	case "http":
		httpServerProcess(args)
		os.Exit(0)
	case "echo":
		echoProcess(args)
		os.Exit(0)
	default:
		fmt.Printf("Unknown command %q\n", cmd)
		os.Exit(1)
	}
}

func httpServerProcess(args []string) {
	if len(args) != 1 {
		fmt.Println("httpserve command needs exactly one argument: the HTTP port to listen on")
		os.Exit(1)
	}

	addr := "localhost:" + args[0]
	fmt.Println("Starting HTTP server at " + addr)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello World!")
	})

	http.ListenAndServe(addr, handler)
}

func echoProcess(args []string) {
	fs := flag.NewFlagSet("echo", flag.ExitOnError)
	stdErr := fs.Bool("stderr", false, "print via std err")
	err := fs.Parse(args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	out := os.Stdout
	if *stdErr {
		out = os.Stderr
		fmt.Println("Printing via stderr")
	}

	for _, v := range args {
		fmt.Fprintln(out, v)
	}
}

// freePort asks the kernel for a free open port that is ready to be used.
func freePort(t GinkgoTInterface) string {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

// testProcessScript creates a command line that runs the test binary and
// executes only the "TestHelperProcess" test cases which delegates execution
// to one of the *Process functions to test sub process execution.
func testProcessScript(args ...string) string {
	return fmt.Sprintf("%s -test.run=^TestHelperProcess$ -- ", os.Args[0]) + strings.Join(args, " ")
}
