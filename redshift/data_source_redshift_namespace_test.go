package redshift

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedshiftNamespace(t *testing.T) {
	ctx, cancel := testContext(t)
	defer cancel()

	config := `
data "redshift_namespace" "namespace" {

}
`
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(ctx, t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestMatchResourceAttr("data.redshift_namespace.namespace", "id", uuidRegex),
			},
		},
	})
}
