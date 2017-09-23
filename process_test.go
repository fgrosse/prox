package prox

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/fgrosse/zaptest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	"go.uber.org/zap"
)

var _ = Describe("process", func() {
	Describe("Name", func() {
		It("should return the process.name", func() {
			p := NewProcess("foo", "echo foo", Environment{})
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
			p := &process{
				name:   "test",
				script: testProcessScript("http", port),
				env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
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

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, nil, log.Named("http_server"))
			}()

			Eventually(httpRequest).Should(Equal(http.StatusOK), "should eventually answer with status code 200")
		})

		It("should send its stdout to the configured writer", func() {
			w := NewBuffer()
			p := &process{
				name:   "test",
				script: testProcessScript("echo", "hello", "world"),
				env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, w, log.Named("process"))
			}()

			Eventually(w).Should(Say(`hello`))
			Eventually(w).Should(Say(`world`))
		})

		It("should send its stderr to the configured writer", func() {
			w := NewBuffer()
			p := &process{
				name:   "test",
				script: testProcessScript("echo", "-stderr", "hello", "world"),
				env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, w, log.Named("process"))
			}()

			Eventually(w).Should(Say(`hello`))
			Eventually(w).Should(Say(`world`))
			log.Info("Full message", zap.String("contents", string(w.Contents())))
		})

		It("should use the given environment", func() {
			w := NewBuffer()
			p := &process{
				name:   "test",
				script: testProcessScript("echo", "-env"),
				env: NewEnv([]string{
					"GO_WANT_HELPER_PROCESS=1",
					"FOO=bar",
					"baz=BLUP",
				}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, w, log.Named("process"))
			}()

			Eventually(w).Should(Say(`FOO=bar`))
			Eventually(w).Should(Say(`baz=BLUP`))
		})

		It("should replace environment variables in the script", func() {
			w := NewBuffer()
			p := &process{
				name:   "test",
				script: testProcessScript("echo", "$FOO"),
				env: NewEnv([]string{
					"GO_WANT_HELPER_PROCESS=1",
					"FOO=it_worked!",
				}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, w, log.Named("process"))
			}()

			Eventually(w).Should(Say(`it_worked!`))
		})

		It("should parse and use environment variables at the beginning of the script", func() {
			w := NewBuffer()
			p := &process{
				name:   "test",
				script: "FOO=nice BAR=cool " + testProcessScript("echo", "-env"),
				env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, w, log.Named("process"))
			}()

			Eventually(w).Should(Say(`BAR=cool`))
			Eventually(w).Should(Say(`FOO=nice`))
		})

		PIt("should pass env variables with spaces at the beginning of the script", func() {
			w := NewBuffer()
			p := &process{
				name:   "test",
				script: `FOO="Hello World" ` + testProcessScript("echo", "-env"),
				env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				defer GinkgoRecover()
				p.Run(ctx, w, log.Named("process"))
			}()

			Eventually(w).Should(Say(`FOO=Hello World`))
		})

		Describe("canceling process", func() {
			It("should send SIGINT if the given context is canceled", func() {
				p := &process{
					name:   "test",
					script: testProcessScript("echo", "-block"),
					env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
				}

				ctx, cancel := context.WithCancel(context.Background())
				sync := make(chan bool)
				go func() {
					defer GinkgoRecover()
					sync <- true
					p.Run(ctx, nil, log.Named("process"))
					sync <- true
				}()

				Eventually(sync).Should(Receive(), "wait for goroutine to start")
				cancel()
				Eventually(sync).Should(Receive(), "wait for goroutine to finish")
			})

			It("should kill process if it does not respond to SIGINT", func() {
				p := &process{
					name:   "test",
					script: testProcessScript("echo", "-block", "-ignoreSIGINT"),
					env:    NewEnv([]string{"GO_WANT_HELPER_PROCESS=1"}),
				}

				ctx, cancel := context.WithCancel(context.Background())
				sync := make(chan bool)
				go func() {
					defer GinkgoRecover()
					sync <- true
					p.Run(ctx, nil, log.Named("process"))
					sync <- true
				}()

				Eventually(sync).Should(Receive(), "wait for goroutine to start")
				cancel()
				Eventually(sync).Should(Receive(), "wait for goroutine to finish")
			})
		})
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
	printEnv := fs.Bool("env", false, "print all environment variables")
	blocking := fs.Bool("block", false, "do not return after printing")
	noSigInt := fs.Bool("no-sigint", false, "ignore SIGINT when blocking")

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

	if *printEnv {
		fmt.Println("Printing all environment variables")
		all := os.Environ()
		sort.Strings(all) // for stable tests

		for _, e := range all {
			fmt.Fprintln(out, e)
		}
	}

	for _, v := range args {
		fmt.Fprintln(out, v)
	}

	if *blocking {
		fmt.Println("Blocking..")
		c := make(chan os.Signal, 1)
		signal.Reset(os.Interrupt, os.Kill) // take control from test runner
		signal.Notify(c)

		for {
			fmt.Println("Waiting for os signal")
			sig := <-c
			fmt.Printf("Received signal %v\n", sig)

			if sig == syscall.SIGINT && *noSigInt {
				// TODO: this does not actually work and the program terminates
				// anyway on sigint. I suspect this is automatically and
				// unconditionally done by the test binary itself somehow.
				continue
			} else {
				break
			}
		}
	}

	os.Exit(0)
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
