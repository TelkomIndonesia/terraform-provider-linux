package linuxbox

import (
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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

				path_previous = "%s"
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
						[ ! -f "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 14
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

	return provider + destroyChecker + linuxbox + createChecker
}

func TestAccLinuxBoxFileOverride(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxBoxFileOverrideConfig(path, path+".neverexist", false),
				ExpectError: regexp.MustCompile("is already exist"),
			},
			{
				Config: testAccLinuxBoxFileOverrideConfig(path, path+".neverexist", true),
			},
			{
				Config:      testAccLinuxBoxFileOverrideConfig(path+".new", path, false),
				ExpectError: regexp.MustCompile("is already exist"),
			},
			{
				Config: testAccLinuxBoxFileOverrideConfig(path+".new", path, true),
			},
		},
	})
}

func testAccLinuxBoxFileOverrideConfig(path, pathPrev string, overwrite bool) string {
	provider := heredoc.Docf(`
		provider "linuxbox" {
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

	linuxbox := heredoc.Docf(`
		resource "linuxbox_file" "overwrite" {
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
				path = linuxbox_file.overwrite.path
				content = linuxbox_file.overwrite.content
				owner = linuxbox_file.overwrite.owner
				group = linuxbox_file.overwrite.group
				mode = linuxbox_file.overwrite.mode

				path_previous = "%s"
				path_compare = "${linuxbox_file.overwrite.path}.compare"
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
						[ ! -f "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 14
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

	return provider + existing + linuxbox + createChecker
}

func TestAccLinuxBoxFileIgnoreContent(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxBoxFileIgnoreContentConfig(path, path+".neverexist", "first"),
			},
			{
				Config: testAccLinuxBoxFileIgnoreContentConfig(path, path+".neverexist", "second"),
			},
		},
	})
}

func testAccLinuxBoxFileIgnoreContentConfig(path, pathPrev, content string) string {
	provider := heredoc.Docf(`
		provider "linuxbox" {
			host = "127.0.0.1"
			port = 2222
			user = "root"
			password = "root"
		}

		locals {
			new_content = "new content"
		}
	`)

	linuxbox := heredoc.Docf(`
		resource "linuxbox_file" "ignore_content" {
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
				path = linuxbox_file.ignore_content.path
				owner = linuxbox_file.ignore_content.owner
				group = linuxbox_file.ignore_content.group
				mode = linuxbox_file.ignore_content.mode
				content = linuxbox_file.ignore_content.content

				path_previous = "%s"
				path_compare = "${linuxbox_file.ignore_content.path}.compare"
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
						[ ! -f "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 14
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

	return provider + linuxbox + createChecker
}

func TestAccLinuxBoxFileRecyclePath(t *testing.T) {
	path := "/tmp/linuxbox/" + acctest.RandString(8)
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		PreCheck:  func() {},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxBoxFileRecyclePathConfig(path, path+".neverexist"),
			},
		},
	})
}

func testAccLinuxBoxFileRecyclePathConfig(path, pathPrev string) string {
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
						find /tmp/recycle -name "$(basename "$FILE")" | grep . || exit 17
						rm -rf /tmp/recycle || exit 18
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
			recycle_path = "/tmp/recycle"
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

				path_previous = "%s"
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
						[ ! -f "${self.triggers.path_previous}"  ] || exit 12
						cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 13
						[ "$( stat -c %%u '${self.triggers.path}' )" == "${self.triggers.owner}" ] || exit 14
						[ "$( stat -c %%g '${self.triggers.path}' )" == "${self.triggers.group}" ] || exit 14
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

	return provider + destroyChecker + linuxbox + createChecker
}
