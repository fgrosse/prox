package prox

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Executor", func() {
	var executor *TestExecutor

	BeforeEach(func() {
		executor = TestNewExecutor(GinkgoWriter)
	})

	It("should run all processes and block until they have finished", func() {
		p1 := &TestProcess{name: "p1"}
		p2 := &TestProcess{name: "p2"}

		go executor.Run(p1, p2)

		Expect(executor.IsDone()).To(BeFalse())
		Eventually(p1.HasBeenStarted).Should(BeTrue(), "it should start p1")
		Eventually(p2.HasBeenStarted).Should(BeTrue(), "it should start p2")
		Expect(executor.IsDone()).To(BeFalse(), "it should not return immediately after all processes have been started")

		p1.Finish()
		Expect(executor.IsDone()).To(BeFalse(), "it should block until all processes have finished")

		p2.Finish()
		Eventually(executor.IsDone).Should(BeTrue(), "it should return once all processes are done")
	})

	Context("when a process fails", func() {
		It("should interrupt all other processes", func() {
			p1 := &TestProcess{name: "p1"}
			p2 := &TestProcess{name: "p2"}

			go executor.Run(p1, p2)
			EventuallyAllProcessesShouldHaveStarted(p1, p2)

			p1.Fail()

			Eventually(p2.HasBeenInterrupted).Should(BeTrue(), "p2 should be interrupted")
			Consistently(p1.HasBeenInterrupted).Should(BeFalse(), "p1 should not interrupt p1 (it failed already)")
			Eventually(executor.IsDone).Should(BeTrue(), "executor should return")
		})

		It("should interrupt all processes concurrently", func() {
			p1 := &TestProcess{name: "p1"}
			p2 := &TestProcess{name: "p2"}
			p3 := &TestProcess{name: "p3"}

			go executor.Run(p1, p2, p3)
			EventuallyAllProcessesShouldHaveStarted(p1, p2, p3)

			p2.ShouldBlockOnInterrupt()
			p1.Fail()

			Consistently(p2.HasBeenInterrupted).Should(BeFalse())
			Eventually(p3.HasBeenInterrupted).Should(BeTrue(), "it should interrupt p3 while waiting for p2")

			p2.FinishInterrupt()
			Eventually(p2.HasBeenInterrupted).Should(BeTrue())
			Eventually(executor.IsDone).Should(BeTrue())
		})

		It("should wait for all processes to finish their interruption", func() {
			p1 := &TestProcess{name: "p1"}
			p2 := &TestProcess{name: "p2"}
			p3 := &TestProcess{name: "p3"}

			go executor.Run(p1, p2, p3)
			EventuallyAllProcessesShouldHaveStarted(p1, p2, p3)

			p2.ShouldBlockOnInterrupt()
			p1.Fail()
			Consistently(executor.IsDone).Should(BeFalse(), "executor should wait for p2")

			p2.FinishInterrupt()
			Eventually(executor.IsDone).Should(BeTrue(), "executor should return when p2 is done")
		})
	})
})

func EventuallyAllProcessesShouldHaveStarted(pp ...*TestProcess) {
	for _, p := range pp {
		Eventually(p.HasBeenStarted).Should(BeTrue(), "it should start process %q", p)
	}
}
