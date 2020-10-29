package linux

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLinuxDirectoryBasic(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDirectoryBasicConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxDirectoryBasicConfig(path+".new", path),
			},
		},
	})
}

func testAccLinuxDirectoryBasicConfig(path, pathPrev string) string {
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
				inline = [
					<<-EOF
						rm -rf '%s' || true
					EOF
				]
			}

			provisioner "remote-exec" {
				when = destroy
				inline = [
					<<-EOF
						[ ! -d '%s' ] || exit 11
					EOF
				]
			}
		}
	`, path, path)

	linux := heredoc.Docf(`
		resource "linux_directory" "basic" {
			depends_on = [ null_resource.destroy_checker]
			path = "%s"
			owner = 1001
			group = 1001
			mode = 755
		}
	`, path)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linux_directory.basic.path
				owner = linux_directory.basic.owner
				group = linux_directory.basic.group
				mode = linux_directory.basic.mode

				path_previous = "%s"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "remote-exec" {
				inline 		= [
					<<-EOF
						[ ! -e '${self.triggers.path_previous}' ] || exit 12
						[ -d '${self.triggers.path}' ] || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
					EOF
				]
			}
		}
	`, pathPrev)

	return provider + destroyChecker + linux + createChecker
}

func TestAccLinuxDirectoryOverride(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxDirectoryeOverrideConfig(path, path+".neverexist", false),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxDirectoryeOverrideConfig(path, path+".neverexist", true),
			},
			{
				Config:      testAccLinuxDirectoryeOverrideConfig(path+".new", path, false),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxDirectoryeOverrideConfig(path+".new", path, true),
			},
		},
	})
}

func testAccLinuxDirectoryeOverrideConfig(path, pathPrev string, overwrite bool) string {
	provider := heredoc.Docf(`
		provider "linux" {
			host = "127.0.0.1"
			port = 2222
			user = "root"
			password = "root"
		}
	`)

	existing := heredoc.Docf(`
		resource "null_resource" "existing" {
			triggers = {
				path = "%s"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "remote-exec" {
				inline 	= [ "mkdir -p '${self.triggers.path}'" ]
			}
		}
	`, path)

	linux := heredoc.Docf(`
		resource "linux_directory" "overwrite" {
			depends_on = [ null_resource.existing]
			path = "%s"
			owner = 1001
			group = 1001
			mode = 644
			overwrite = %t
		}
	`, path, overwrite)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linux_directory.overwrite.path
				owner = linux_directory.overwrite.owner
				group = linux_directory.overwrite.group
				mode = linux_directory.overwrite.mode

				path_previous = "%s"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "remote-exec" {
				inline 		= [
					<<-EOF
						[ ! -e '${self.triggers.path_previous}' ] || exit 12
						[ -d '${self.triggers.path}' ] || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
					EOF
				]
			}
			
		}
	`, pathPrev)

	return provider + existing + linux + createChecker
}

func TestAccLinuxDirectoryRecyclePath(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDirectoryRecyclePathConfig(path, path+".neverexist"),
			},
		},
	})
}

func testAccLinuxDirectoryRecyclePathConfig(path, pathPrev string) string {
	provider := heredoc.Docf(`
		provider "linux" {
			host = "127.0.0.1"
			port = 2222
			user = "root"
			password = "root"
		}
	`)

	destroyChecker := heredoc.Docf(`
		locals {
			recycle_path = "/tmp/recycle"
		}
		resource "null_resource" "destroy_checker" {
			triggers = {
				recycle_path = local.recycle_path
			}
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
						DIRECTORY="%s" 
						[ ! -e '%s' ] || exit 10
						find ${self.triggers.recycle_path} -name "$(basename "$DIRECTORY")" | grep . || exit 17
						rm -rf ${self.triggers.recycle_path} || exit 18
					EOF
				]
			}
		}
	`, path)

	linux := heredoc.Docf(`
		resource "linux_directory" "basic" {
			depends_on = [ null_resource.destroy_checker]
			path = "%s"
			owner = 1001
			group = 1001
			mode = 644
			recycle_path = local.recycle_path
		}
	`, path)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linux_directory.basic.path
				owner = linux_directory.basic.owner
				group = linux_directory.basic.group
				mode = linux_directory.basic.mode
				recycle_path = linux_directory.basic.recycle_path

				path_previous = "%s"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "remote-exec" {
				inline 		= [
					<<-EOF
						! -e '${self.triggers.path_previous}' ] || exit 12
						[ -d '${self.triggers.path}' ] || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
					EOF
				]
			}
		}
	`, pathPrev)

	return provider + destroyChecker + linux + createChecker
}
