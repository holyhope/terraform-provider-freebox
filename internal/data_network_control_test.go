package internal_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DataNetworkControl", func() {
	var (
		config         string
		resourceName   string
		networkControl freeboxTypes.NetworkControlInfo
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")

		networkControls, err := freeboxClient.ListNetworkControl(ctx)
		Expect(err).To(BeNil())
		Expect(networkControls).ToNot(BeEmpty())

		networkControl = networkControls[0]
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			data "freebox_network_control" "` + resourceName + `" {
				profile_id = ` + strconv.FormatInt(networkControl.ProfileID, 10) + `
			}
		`
	})

	It("should fetch network control information", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_network_control."+resourceName].Primary.Attributes

							Expect(state["profile_id"]).To(Equal(strconv.FormatInt(networkControl.ProfileID, 10)))
							Expect(state["override_mode"]).To(Equal(string(networkControl.OverrideMode)))
							Expect(state["current_mode"]).To(Equal(string(networkControl.CurrentMode)))
							Expect(state["rule_mode"]).To(Equal(string(networkControl.RuleMode)))
							Expect(state["override_until"]).To(Equal(strconv.Itoa(networkControl.OverrideUntil)))
							Expect(state["override"]).To(Equal(strconv.FormatBool(networkControl.Override)))
							Expect(state["resolution"]).To(Equal(strconv.Itoa(networkControl.Resolution)))
							Expect(state["macs.#"]).To(Equal(strconv.Itoa(len(networkControl.Macs))))

							for i, mac := range networkControl.Macs {
								Expect(state[fmt.Sprintf("macs.%d", i)]).To(Equal(mac))
							}

							return nil
						},
					),
				},
			},
		})
	})
})
