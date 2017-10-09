package prox

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Server", func() {
	Describe("Tail", func() {
		It("should return the output of a process to the Client", func() {
			t := GinkgoT()
			_, client, executor, done := TestNewServerAndClient(t, GinkgoWriter)
			defer done()

			p1 := &TestProcess{name: "p1"}
			p2 := &TestProcess{name: "p2"}

			go executor.Run(p1, p2)

			ctx := context.Background()
			output := NewBuffer()

			Eventually(p1.HasBeenStarted).Should(BeTrue())
			Eventually(p2.HasBeenStarted).Should(BeTrue())

			sync := make(chan bool)
			go func() {
				defer GinkgoRecover()
				sync <- true
				err := client.Tail(ctx, []string{"p2"}, output)
				Expect(err).NotTo(HaveOccurred())
			}()

			<-sync
			time.Sleep(time.Millisecond) // some more time for client to establish the tail
			// TODO: the sleep above can be removed if we can tail from the beginning

			p1.ShouldSay(t, "A message from p1\n")
			p2.ShouldSay(t, "A message from p2\n")
			p1.ShouldSay(t, "Another message from p1\n")
			p2.ShouldSay(t, "And another message from p2\n")

			Consistently(output).ShouldNot(Say("A message from p1"))
			Consistently(output).ShouldNot(Say("Another message from p1"))
			Eventually(output).Should(Say("A message from p2"))
			Eventually(output).Should(Say("And another message from p2"))
		})
	})

	Describe("List", func() {
		It("should return a list of all currently running processes to the Client", func() {
			t := GinkgoT()
			_, client, executor, done := TestNewServerAndClient(t, GinkgoWriter)
			defer done()

			p1 := &TestProcess{name: "p1", PID: 101}
			p2 := &TestProcess{name: "p2", PID: 102}

			go executor.Run(p1, p2)

			ctx := context.Background()
			output := NewBuffer()

			Eventually(p1.HasBeenStarted).Should(BeTrue())
			Eventually(p2.HasBeenStarted).Should(BeTrue())

			go func() {
				defer GinkgoRecover()
				err := client.List(ctx, output)
				Expect(err).NotTo(HaveOccurred())
			}()

			Eventually(output).Should(Say("NAME    PID"))
			Eventually(output).Should(Say("p1      101"))
			Eventually(output).Should(Say("p2      102"))
		})
	})
})
