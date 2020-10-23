package linuxbox

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccLinuxBoxFileBasic(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxBoxFileBasicConfig(path),
			},
		},
		CheckDestroy: func(t *terraform.State) (err error) { return },
	})
}

func testAccLinuxBoxFileBasicConfig(path string) string {
	return fmt.Sprintf(`
		provider "linuxbox" {
			host = "127.0.0.1"
			port = 2222
			user = "linuxbox"
			password = "password"
		}

		resource "linuxbox_file" "basic" {
			path = "%s"
			content = <<-EOF
				test
				file
			EOF
			owner = 1001
			group = 1001
		}
	`, path)
}
