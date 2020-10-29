package linux

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccLinuxFileBasic(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileBasicConfig(path, path+".neverexist"),
			},
			{
				Config: testAccLinuxFileBasicConfig(path+".new", path),
			},
		},
	})
}

func testAccLinuxFileBasicConfig(path, pathPrev string) string {
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
						[ ! -e '%s' ] || exit 10
					EOF
				]
			}
		}
	`, path)

	linux := heredoc.Docf(`
		resource "linux_file" "basic" {
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
				path = linux_file.basic.path
				content = linux_file.basic.content
				owner = linux_file.basic.owner
				group = linux_file.basic.group
				mode = linux_file.basic.mode

				path_previous = "%s"
				path_compare = "${linux_file.basic.path}.compare"
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
						[ ! -e "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
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

func TestAccLinuxFileOverride(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxFileOverrideConfig(path, path+".neverexist", false),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxFileOverrideConfig(path, path+".neverexist", true),
			},
			{
				Config:      testAccLinuxFileOverrideConfig(path+".new", path, false),
				ExpectError: regexp.MustCompile(" exist"),
			},
			{
				Config: testAccLinuxFileOverrideConfig(path+".new", path, true),
			},
		},
	})
}

func testAccLinuxFileOverrideConfig(path, pathPrev string, overwrite bool) string {
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
			provisioner "file" {
				content      	= "existing"
				destination 	= self.triggers.path
			}	
		}
	`, path)

	linux := heredoc.Docf(`
		resource "linux_file" "overwrite" {
			depends_on = [ null_resource.existing]
			path = "%s"
			content = <<-EOF
				this test file should be
				present in remote
			EOF
			owner = 1001
			group = 1001
			mode = 644
			overwrite = %t
		}
	`, path, overwrite)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linux_file.overwrite.path
				content = linux_file.overwrite.content
				owner = linux_file.overwrite.owner
				group = linux_file.overwrite.group
				mode = linux_file.overwrite.mode

				path_previous = "%s"
				path_compare = "${linux_file.overwrite.path}.compare"
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
						[ ! -e "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
					EOF
				]
			}
			provisioner "remote-exec" {
				when	= destroy
				inline 	= [ "rm -f '${self.triggers.path_compare}'" ]
			}
		}
	`, pathPrev)

	return provider + existing + linux + createChecker
}

func TestAccLinuxFileIgnoreContent(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileIgnoreContentConfig(path, path+".neverexist", "first"),
			},
			{
				Config: testAccLinuxFileIgnoreContentConfig(path, path+".neverexist", "second"),
			},
		},
	})
}

func testAccLinuxFileIgnoreContentConfig(path, pathPrev, content string) string {
	provider := heredoc.Docf(`
		provider "linux" {
			host = "127.0.0.1"
			port = 2222
			user = "root"
			password = "root"
		}
	`)

	linux := heredoc.Docf(`
		locals {
			new_content = "new content"
		}

		resource "linux_file" "ignore_content" {
			path = "%s"
			content = "%s"
			owner = 1001
			group = 1001
			mode = 644
			ignore_content = true

			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "remote-exec" {
				inline 		= [ "echo -n '${local.new_content}' > ${self.path}" ]
			}	
		}
	`, path, content)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linux_file.ignore_content.path
				owner = linux_file.ignore_content.owner
				group = linux_file.ignore_content.group
				mode = linux_file.ignore_content.mode
				content = linux_file.ignore_content.content

				path_previous = "%s"
				path_compare = "${linux_file.ignore_content.path}.compare"
			}
			connection {
				type     = "ssh"
				host     = "127.0.0.1"
				port 	 = 2222
				user     = "root"
				password = "root" 
			}
			provisioner "file" {
				content      	= local.new_content
				destination 	= self.triggers.path_compare
			}
			provisioner "remote-exec" {
				inline 		= [
					<<-EOF
						[ ! -e "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
					EOF
				]
			}
			provisioner "remote-exec" {
				when	= destroy
				inline 	= [ "rm -f '${self.triggers.path_compare}'" ]
			}
		}
	`, pathPrev)

	return provider + linux + createChecker
}

func TestAccLinuxFileRecyclePath(t *testing.T) {
	path := "/tmp/linux/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxFileRecyclePathConfig(path, path+".neverexist"),
			},
		},
	})
}

func testAccLinuxFileRecyclePathConfig(path, pathPrev string) string {
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
						FILE="%s" 
						[ ! -e '%s' ] || exit 10
						find ${self.triggers.recycle_path} -name "$(basename "$FILE")" | grep . || exit 17
						rm -rf ${self.triggers.recycle_path} || exit 18
					EOF
				]
			}
		}
	`, path)

	linux := heredoc.Docf(`
		resource "linux_file" "basic" {
			depends_on = [ null_resource.destroy_checker]
			path = "%s"
			content = <<-EOF
				this test file should be
				present in remote
			EOF
			owner = 1001
			group = 1001
			mode = 644
			recycle_path = local.recycle_path
		}
	`, path)

	createChecker := heredoc.Docf(`
		resource "null_resource" "create_checker" {
			triggers = {
				path = linux_file.basic.path
				content = linux_file.basic.content
				owner = linux_file.basic.owner
				group = linux_file.basic.group
				mode = linux_file.basic.mode
				recycle_path = linux_file.basic.recycle_path

				path_previous = "%s"
				path_compare = "${linux_file.basic.path}.compare"
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
						[ ! -e "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 15
						[ "$( stat -c %%a '${self.triggers.path}' )" == "${self.triggers.mode}" ] || exit 16
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
