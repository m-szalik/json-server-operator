/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/uuid"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"github.com/m-szalik/json-server-operator/test/utils"
)

const testNamespace = "json-serv-operator-test"
const operatorSystemNamespace = "json-server-operator-system"

var operatorImage string

var _ = Describe("JsonServer", Ordered, func() {
	BeforeAll(func() {
		operatorImage = fmt.Sprintf("ttl.sh/%s:1h", uuid.NewUUID())
		By("installing the cert-manager")
		Expect(utils.InstallCertManager()).To(Succeed())
		By("creating test testNamespace")
		cmd := exec.Command("kubectl", "create", "ns", testNamespace)
		_, _ = utils.Run(cmd)
	})

	AfterAll(func() {
		By("uninstalling the cert-manager bundle")
		utils.UninstallCertManager()
		By("removing test testNamespace")
		cmd := exec.Command("kubectl", "delete", "ns", testNamespace)
		_, _ = utils.Run(cmd)
		By("removing manager testNamespace")
		cmd = exec.Command("kubectl", "delete", "ns", operatorSystemNamespace)
		_, _ = utils.Run(cmd)
	})

	Context("Should create resources", func() {
		It("Should create resources", func() {
			var controllerPodName string
			var err error
			projectDir, _ := utils.GetProjectDir()

			By("building the manager(Operator) image")
			cmd := exec.Command("docker", "build", "-t", operatorImage, "--build-arg", fmt.Sprintf("IMG=%s", operatorImage), ".")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("push the image")
			cmd = exec.Command("docker", "push", operatorImage)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", operatorImage))
			_, err = utils.Run(cmd)
			time.Sleep(20 * time.Second)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				cmd = exec.Command("kubectl", "get", "pods", "-l", "control-plane=controller-manager", "-o", "name", "-n", operatorSystemNamespace)
				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				cmd = exec.Command("kubectl", "get", controllerPodName, "-o", "jsonpath={.status.phase}", "-n", operatorSystemNamespace)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

			var jsonServerName string
			By("creating an instance of the JsonServer")
			EventuallyWithOffset(1, func() error {
				cmd = exec.Command("kubectl", "apply", "-f", filepath.Join(projectDir, "config/samples/example.com_v1_jsonserver.yaml"), "-n", testNamespace, "-o", "name")
				buff, err := utils.Run(cmd)
				jsonServerName = strings.TrimSpace(strings.Split(string(buff), "/")[1])
				return err
			}, time.Minute, time.Second).Should(Succeed())

			// validate if all resources have been created
			By("validating that configmap was created")
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "configmap", jsonServerName, "-n", testNamespace)
				_, err = utils.Run(cmd)
				return err
			}).Should(Succeed())
			By("validating that deployment was created")
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "deployment", jsonServerName, "-n", testNamespace)
				_, err = utils.Run(cmd)
				return err
			}).Should(Succeed())
			By("validating that service was created")
			Eventually(func() error {
				cmd = exec.Command("kubectl", "get", "service", jsonServerName, "-n", testNamespace)
				_, err = utils.Run(cmd)
				return err
			}).Should(Succeed())

			// TODO add more tests
		})
	})
})
