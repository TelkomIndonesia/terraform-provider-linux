package linuxbox

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccLinuxBoxFileBasic(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxBoxFileBasicConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxBoxFileBasicConfig(path+".new", path),
			},
		},
		CheckDestroy: func(t *terraform.State) (err error) {
			_, err = ioutil.ReadFile(path + "1")
			if err == nil {
				return fmt.Errorf("file at %s should no longer exist", path)
			}
			return nil
		},
	})
}

func testAccLinuxBoxFileBasicConfig(path, pathPrev string) string {
	provider := heredoc.Docf(`
		provider "linuxbox" {
			host = "127.0.0.1"
			port = 2222
			user = "root"
			password = "root"
		}
	`)

	destroyChecker := heredoc.Docf(`
		resource "null_resource" "destroy_checker" {
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}

			provisioner "remote-exec" {
				when = destroy
				inline = [
					<<-EOF
						FILE="%s" 
						mkdir -p $FILE || exit 10
						rm -rf $FILE || exit 11
					EOF
				]
			}
		}
	`, path)

	linuxbox := heredoc.Docf(`
		resource "linuxbox_file" "basic" {
			depends_on = [ null_resource.destroy_checker]
			path = "%s"
			content = <<-EOF
				this test file should be
				present in remote
			EOF
			owner = 1001
			group = 1001
			mode = 644
		}
	`, path)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linuxbox_file.basic.path
				content = linuxbox_file.basic.content
				owner = linuxbox_file.basic.owner
				group = linuxbox_file.basic.group
				mode = linuxbox_file.basic.mode

				previous_path = "%s"
				path_compare = "${linuxbox_file.basic.path}.compare"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "file" {
				content      	= self.triggers.content
				destination 	= self.triggers.path_compare
			}
			provisioner "remote-exec" {
				inline 		= [
					<<-EOF
						[ ! -f "${self.triggers.previous_path}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 14
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
					EOF
				]
			}
		}
	`, pathPrev)

	return provider + destroyChecker + linuxbox + createChecker
}
