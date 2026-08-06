package internal_test

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/nikolalohinski/free-go/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`resource "freebox_vpn_user" { ... }`, func() {
	var (
		resourceName string
		login        string
		password     string
		config       string
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resourceName = strings.Join(splitName[:len(splitName)-1], "-")
		login = resourceName
		password = uuid.New().String()
	})

	JustBeforeEach(func(ctx SpecContext) {
		config = providerBlock + `
			resource "freebox_vpn_user" "` + resourceName + `" {
				login    = "` + login + `"
				password = "` + password + `"
			}
		`
	})

	Context("create and delete", func() {
		It("should create and delete the VPN user", func(ctx SpecContext) {
			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("freebox_vpn_user."+resourceName, "login", login),
							resource.TestCheckResourceAttrSet("freebox_vpn_user."+resourceName, "ovpn_config"),
							func(s *terraform.State) error {
								_, err := freeboxClient.GetVPNUser(ctx, login)
								Expect(err).To(BeNil())
								return nil
							},
						),
					},
				},
				CheckDestroy: func(s *terraform.State) error {
					_, err := freeboxClient.GetVPNUser(ctx, login)
					Expect(errors.Is(err, client.ErrVPNUserNotFound)).To(BeTrue(), "expected VPN user to be deleted, got: %v", err)
					return nil
				},
			})
		})
	})

	Context("refreshing state", func() {
		It("should not report ovpn_config as changed across a refresh", func(ctx SpecContext) {
			// The Freebox API embeds a freshly generated certificate in the
			// response every time GetVPNUserClientConfig is called, even when
			// nothing about the VPN user changed. Read must not blindly
			// re-fetch and overwrite ovpn_config on every refresh, or this
			// resource (and anything consuming its ovpn_config output, e.g. a
			// downstream OpenVPN client profile) would be forced to replace
			// on every single terraform apply.
			var firstOVPNConfig string

			resource.UnitTest(GinkgoT(), resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: func(s *terraform.State) error {
							firstOVPNConfig = s.RootModule().Resources["freebox_vpn_user."+resourceName].Primary.Attributes["ovpn_config"]
							Expect(firstOVPNConfig).ToNot(BeEmpty())
							return nil
						},
					},
					{
						RefreshState: true,
						Check: func(s *terraform.State) error {
							Expect(s.RootModule().Resources["freebox_vpn_user."+resourceName].Primary.Attributes["ovpn_config"]).To(Equal(firstOVPNConfig))
							return nil
						},
					},
					{
						Config:             config,
						PlanOnly:           true,
						ExpectNonEmptyPlan: false,
					},
				},
			})
		})
	})
})
