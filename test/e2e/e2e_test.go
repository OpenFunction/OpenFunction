package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	openfunction "github.com/openfunction/apis/core/v1alpha2"
)

var _ = Describe("E2E Tests", func() {

	var (
		err error
		fn  *openfunction.Function
	)

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			printTestFailureDebugInfo(fn)
		}

		if fn != nil {
			if err := cl.Delete(context.Background(), fn); err != nil {
				logf("delete fn %s/%s error, %s", fn.Name, fn.Namespace, err.Error())
			}
			fn = nil
		}
	})

	Context("Sync Function with build and serving test",
		func() {

			It("should be successfully", func() {
				fn, err = createFunction("../../config/samples/function-sample.yaml")
				Expect(err).NotTo(HaveOccurred())

				Expect(cl.Create(context.Background(), fn)).NotTo(HaveOccurred())

				Eventually(func() bool {
					var res bool
					res, err = checkFunction(fn)
					if err != nil {
						return true
					}

					return res
				}, time.Minute*10, time.Second*5).Should(BeTrue())
				Expect(err).NotTo(HaveOccurred())

				err = accessFunction(fn)
				if err != nil {
					logf("access function %s/%s error, %s", fn.Name, fn.Namespace, err.Error())
				}
				Expect(err).NotTo(HaveOccurred())
			})
		})

	Context("Sync Function with build only test",
		func() {

			It("should be successfully", func() {
				fn, err = createFunction("../../config/samples/function-sample-build-only.yaml")
				Expect(err).NotTo(HaveOccurred())

				Expect(cl.Create(context.Background(), fn)).NotTo(HaveOccurred())

				Eventually(func() bool {
					var res bool
					res, err = checkFunction(fn)
					if err != nil {
						return true
					}

					return res
				}, time.Minute*10, time.Second*5).Should(BeTrue())
				Expect(err).NotTo(HaveOccurred())
			})
		})

	Context("Sync Function with serving only test",
		func() {

			It("should be successfully", func() {
				fn, err = createFunction("../../config/samples/function-sample-serving-only.yaml")
				Expect(err).NotTo(HaveOccurred())

				Expect(cl.Create(context.Background(), fn)).NotTo(HaveOccurred())

				Eventually(func() bool {
					var res bool
					res, err = checkFunction(fn)
					if err != nil {
						return true
					}

					return res
				}, time.Minute*10, time.Second*5).Should(BeTrue())
				Expect(err).NotTo(HaveOccurred())

				err = accessFunction(fn)
				if err != nil {
					logf("access function %s/%s error, %s", fn.Name, fn.Namespace, err.Error())
				}
				Expect(err).NotTo(HaveOccurred())
			})
		})

	Context("Async function with build and serving test", func() {

		It("should be successfully", func() {
			fn, err = createFunction("../../config/samples/function-producer-sample.yaml")
			Expect(err).NotTo(HaveOccurred())

			Expect(cl.Create(context.Background(), fn)).NotTo(HaveOccurred())

			Eventually(func() bool {
				var res bool
				res, err = checkFunction(fn)
				if err != nil {
					return true
				}

				return res
			}, time.Minute*10, time.Second*5).Should(BeTrue())
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
