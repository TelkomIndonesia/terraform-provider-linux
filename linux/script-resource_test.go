package linux

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLinuxScriptBasic(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptBasicConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxScriptBasicConfig(path+".new", path),
			},
		},
	})
}

func testAccLinuxScriptBasicConfig(path, pathPrev string) string {
	provider := heredoc.Docf(`
		provider "linux" {
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

	linux := heredoc.Docf(`
		locals {
			path  = "%s"
			content = "test"
		} 
		resource "linux_script" "basic" {
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
			depends_on = [ linux_script.basic ]
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

	return provider + destroyChecker + linux + createChecker
}

func TestAccLinuxScriptNoUpdate(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptNoUpdateConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxScriptNoUpdateConfig(path+".new", path),
			},
		},
	})
}

func testAccLinuxScriptNoUpdateConfig(path, pathPrev string) string {
	provider := heredoc.Docf(`
		provider "linux" {
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

	linux := heredoc.Doc(`
		resource "linux_script" "no_update" {
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
			depends_on = [ linux_script.no_update ]
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

	return provider + locals + destroyChecker + directory + linux + createChecker
}
