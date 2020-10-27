package linuxbox

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLinuxBoxScriptBasic(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxBoxScriptBasicConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxBoxScriptBasicConfig(path+".new", path),
			},
		},
	})
}

func testAccLinuxBoxScriptBasicConfig(path, pathPrev string) string {
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
						[ ! -e '%s' ] || exit 10
					EOF
				]
			}
		}
	`, path)

	linuxbox := heredoc.Docf(`
		locals {
			path  = "%s"
			content = "test"
		} 
		resource "linuxbox_script" "basic" {
			depends_on = [ null_resource.destroy_checker ]
			lifecycle_commands {
				create = "mkdir -p $(dirname $FILE) && echo -n '${local.content}' > $FILE"
				read = "echo $FILE"
				update = "echo $FILE $FILE"
				delete = "rm $FILE"
			}
			environment = {
				FILE = local.path
			}
		}
	`, path)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			depends_on = [ linuxbox_script.basic ]
			triggers = {
				path = local.path
				path_previous = "%s"
				path_compare = "${local.path}.compare"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "file" {
				content      	= local.content
				destination 	= self.triggers.path_compare
			}
			provisioner "remote-exec" {
				inline 		= [
					<<-EOF
						[ ! -e "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
					EOF
				]
			}
			provisioner "remote-exec" {
				when	= destroy
				inline 	= [ "rm -f '${self.triggers.path_compare}'" ]
			}
		}
	`, pathPrev)

	return provider + destroyChecker + linuxbox + createChecker
}
