package internal_test

import (
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	"github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`resource "freebox_virtual_machine_power" { ... }`, func() {
	const diskType = "qcow2"

	var (
		vmResourceName    string
		powerResourceName string
		initialConfig     string
		powerState        string
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		vmResourceName = strings.Join(splitName[:len(splitName)-1], "-") + "-vm"
		powerResourceName = strings.Join(splitName[:len(splitName)-1], "-") + "-power"
		powerState = types.RunningStatus
	})

	JustBeforeEach(func(ctx SpecContext) {
		initialConfig = providerBlock + `
			resource "freebox_virtual_machine" "` + vmResourceName + `" {
				vcpus     = 1
				memory    = 300
				name      = "` + vmResourceName + `"
				disk_type = "` + diskType + `"
				disk_path = "` + existingDisk.filepath + `"
				status    = "stopped"

				enable_cloudinit   = false
				cloudinit_hostname = null
				cloudinit_userdata = null

				timeouts = {
					kill       = "500ms"
					networking = "0s"
				}
			}

			resource "freebox_virtual_machine_power" "` + powerResourceName + `" {
				vm_id       = freebox_virtual_machine.` + vmResourceName + `.id
				power_state = "` + powerState + `"

				timeouts = {
					kill = "500ms"
				}
			}
		`
	})

	vmIDFromState := func(s *terraform.State) int64 {
		id, err := strconv.ParseInt(s.RootModule().Resources["freebox_virtual_machine."+vmResourceName].Primary.Attributes["id"], 10, 64)
		Expect(err).To(BeNil())
		return id
	}

	expectVMStatus := func(ctx SpecContext, vmID int64, status string) {
		Eventually(func(g Gomega) string {
			vm, err := freeboxClient.GetVirtualMachine(ctx, vmID)
			g.Expect(err).To(BeNil())
			return vm.Status
		}, "2m", "2s").Should(Equal(status))
	}

	Context("create and delete", func() {
		It("should start a stopped virtual machine", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine."+vmResourceName, "status", types.StoppedStatus),
							resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "power_state", types.RunningStatus),
							resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "timeouts.kill", "500ms"),
							func(s *terraform.State) error {
								vmID := vmIDFromState(s)
								Expect(s.RootModule().Resources["freebox_virtual_machine_power."+powerResourceName].Primary.Attributes["id"]).To(Equal(strconv.FormatInt(vmID, 10)))
								expectVMStatus(ctx, vmID, types.RunningStatus)
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					vmID, err := strconv.ParseInt(s.RootModule().Resources["freebox_virtual_machine."+vmResourceName].Primary.Attributes["id"], 10, 64)
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, vmID)
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound), "virtual machine %d should not exist", vmID)

					return nil
				},
			})
		})

		Context("when the desired power state is stopped", func() {
			BeforeEach(func(ctx SpecContext) {
				powerState = types.StoppedStatus
			})

			It("should keep the virtual machine stopped", func(ctx SpecContext) {
				resource.UnitTest(GinkgoT(), resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: initialConfig,
							Check: resource.ComposeAggregateTestCheckFunc(
								resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "power_state", types.StoppedStatus),
								func(s *terraform.State) error {
									expectVMStatus(ctx, vmIDFromState(s), types.StoppedStatus)
									return nil
								},
							),
						},
					},
					CheckDestroy: func(s *terraform.State) error {
						vmID, err := strconv.ParseInt(s.RootModule().Resources["freebox_virtual_machine."+vmResourceName].Primary.Attributes["id"], 10, 64)
						Expect(err).To(BeNil())

						_, err = freeboxClient.GetVirtualMachine(ctx, vmID)
						Expect(err).To(MatchError(client.ErrVirtualMachineNotFound))

						return nil
					},
				})
			})
		})
	})

	Context("create, update and delete", func() {
		It("should stop a running virtual machine", func(ctx SpecContext) {
			stoppedConfig := terraformConfigWithAttribute("power_state", types.StoppedStatus)(initialConfig)

			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "power_state", types.RunningStatus),
							func(s *terraform.State) error {
								expectVMStatus(ctx, vmIDFromState(s), types.RunningStatus)
								return nil
							},
						),
					},
					{
						Config: stoppedConfig,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("freebox_virtual_machine_power."+powerResourceName, plancheck.ResourceActionUpdate),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "power_state", types.StoppedStatus),
							func(s *terraform.State) error {
								expectVMStatus(ctx, vmIDFromState(s), types.StoppedStatus)
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					vmID, err := strconv.ParseInt(s.RootModule().Resources["freebox_virtual_machine."+vmResourceName].Primary.Attributes["id"], 10, 64)
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, vmID)
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound))

					return nil
				},
			})
		})
	})

	Context("import and delete", func() {
		It("should import from the virtual machine identifier", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: initialConfig,
					},
					{
						Config:        initialConfig,
						ResourceName:  "freebox_virtual_machine_power." + powerResourceName,
						ImportState:   true,
						ImportStateIdFunc: func(s *terraform.State) (string, error) {
							return s.RootModule().Resources["freebox_virtual_machine."+vmResourceName].Primary.Attributes["id"], nil
						},
						ImportStatePersist: true,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "power_state", types.RunningStatus),
							resource.TestCheckResourceAttr("freebox_virtual_machine_power."+powerResourceName, "timeouts.kill", "30s"),
							func(s *terraform.State) error {
								vmID := vmIDFromState(s)
								Expect(s.RootModule().Resources["freebox_virtual_machine_power."+powerResourceName].Primary.Attributes["id"]).To(Equal(strconv.FormatInt(vmID, 10)))
								Expect(s.RootModule().Resources["freebox_virtual_machine_power."+powerResourceName].Primary.Attributes["vm_id"]).To(Equal(strconv.FormatInt(vmID, 10)))
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					vmID, err := strconv.ParseInt(s.RootModule().Resources["freebox_virtual_machine."+vmResourceName].Primary.Attributes["id"], 10, 64)
					Expect(err).To(BeNil())

					_, err = freeboxClient.GetVirtualMachine(ctx, vmID)
					Expect(err).To(MatchError(client.ErrVirtualMachineNotFound))

					return nil
				},
			})
		})
	})
})
