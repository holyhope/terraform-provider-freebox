package internal_test

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe(`data "freebox_profiles" { ... }`, func() {
	var (
		config   string
		resName  string
		profiles []freeboxTypes.Profile
	)

	BeforeEach(func(ctx SpecContext) {
		splitName := strings.Split(("test-" + uuid.New().String())[:30], "-")
		resName = strings.Join(splitName[:len(splitName)-1], "-")

		var err error
		profiles, err = freeboxClient.ListProfiles(ctx)
		Expect(err).To(BeNil())
		Expect(profiles).ToNot(BeEmpty())
	})

	JustBeforeEach(func() {
		config = providerBlock + `
			data "freebox_profiles" "` + resName + `" {
			}
		`
	})

	It("should list all profiles", func(ctx SpecContext) {
		resource.UnitTest(GinkgoT(), resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr(
							"data.freebox_profiles."+resName,
							"profiles.#",
							fmt.Sprintf("%d", len(profiles)),
						),
						func(s *terraform.State) error {
							state := s.RootModule().Resources["data.freebox_profiles."+resName].Primary.Attributes

							for i, profile := range profiles {
								Expect(state[fmt.Sprintf("profiles.%d.id", i)]).To(Equal(fmt.Sprintf("%d", profile.ID)))
								Expect(state[fmt.Sprintf("profiles.%d.name", i)]).To(Equal(profile.Name))
								Expect(state[fmt.Sprintf("profiles.%d.icon", i)]).To(Equal(profile.Icon))
							}

							return nil
						},
					),
				},
			},
		})
	})
})
