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
						FILE='%s'
						[ -e "$FILE".update ] || exit 10
						rm -rf "$FILE".update
						[ ! -e "$FILE" ] || exit 11
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
				update = <<-EOF
					OLD_FILE="$(cat)"
					touch "$FILE".update
					mv "$OLD_FILE" $FILE
				EOF
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

func TestAccLinuxBoxScriptNoUpdate(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxBoxScriptNoUpdateConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxBoxScriptNoUpdateConfig(path+".new", path),
			},
		},
	})
}

func testAccLinuxBoxScriptNoUpdateConfig(path, pathPrev string) string {
	provider := heredoc.Docf(`
		provider "linuxbox" {
			host = "127.0.0.1"
			port = 2222
			user = "root"
			password = "root"
		}
	`)

	locals := heredoc.Docf(`
		locals {
			path  = "%s"
			content = "test"
			path_previous = "%s"
		}
	`, path, pathPrev)

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
					"[ ! -e '%s' ] || exit 10"
				]
			}
		}
	`, path)

	directory := heredoc.Docf(`
		resource "null_resource" "directory" {
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}

			triggers = {
				directory = dirname(local.path)
			}

			provisioner "remote-exec" {
				inline = [
					"mkdir -p '${self.triggers.directory}'"
				]
			}
			provisioner "remote-exec" {
				when = destroy
				inline = [
					"rm -rf '${self.triggers.directory}'"
				]
			}
		}
	`)

	linuxbox := heredoc.Doc(`
		resource "linuxbox_script" "no_update" {
			depends_on = [ null_resource.destroy_checker, null_resource.directory ]
			lifecycle_commands {
				create = "echo -n '${local.content}' > $FILE"
				read = "echo $FILE"
				delete = "rm $FILE"
			}
			environment = {
				FILE = local.path
			}
		}
	`)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			depends_on = [ linuxbox_script.no_update ]
			triggers = {
				path = local.path
				path_previous = local.path_previous
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
	`)

	return provider + locals + destroyChecker + directory + linuxbox + createChecker
}
