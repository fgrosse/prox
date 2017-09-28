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

			p1.ShouldSay(t, "A message from p1")
			p2.ShouldSay(t, "A message from p2")
			p1.ShouldSay(t, "Another message from p1")
			p2.ShouldSay(t, "And another message from p2")

			Consistently(output).ShouldNot(Say("A message from p1"))
			Consistently(output).ShouldNot(Say("Another message from p1"))
			Eventually(output).Should(Say("A message from p2"))
			Eventually(output).Should(Say("And another message from p2"))

			//GinkgoWriter.Write(output.Contents())
		})
	})
})
