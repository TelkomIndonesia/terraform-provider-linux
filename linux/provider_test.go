package linux

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxProviderUnknownValue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		PreCheck:  testAccPreCheckConnection(t),
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxProviderUnknownValueConf(t),
			},
		},
	},
	)
}

func testAccLinuxProviderUnknownValueConf(t *testing.T) (s string) {
	data := struct {
		Provider1, Provider2 tfmap
	}{
		testAccProvider, testAccProvider.Copy().Without("host"),
	}

	conf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider1.Serialize | nindent 4 }}
		}
		
		resource "linux_script" "script" {
		    lifecycle_commands {
		        create = "echo -n"
		        read = <<-EOF
		            echo -n {{ .Provider1.host }}
		        EOF
		        delete = "echo -n"
		    }
		}

		provider "linux" {
		    alias = "two"

		    host = linux_script.script.output
            {{- .Provider2.Serialize | nindent 4 }}
		}
		resource "linux_script" "script_two" {
		    provider = linux.two
		    
		    lifecycle_commands {
		        create = "echo -n 'hi' > /tmp/test"
		        read = "cat /tmp/test"
		        delete = "rm /tmp/test"
		    }
		    
		    connection {
		        type = "ssh"
		        {{- .Provider1.Serialize | nindent 8 }}
		    }
		    
		    provisioner "remote-exec" {
		        inline = [ 
		            <<-EOF
		                [ "$(cat /tmp/test)" ==  "hi" ]
		            EOF
		        ]
		    }
		}
	`)
	s, err := tCompileTemplate(conf, data)
	require.NoError(t, err)
	t.Log(s)
	return
}

func TestAccLinuxProviderParallel(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		PreCheck:  testAccPreCheckConnection(t),
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxProviderParallelConf(t),
			},
		},
	},
	)
}

func testAccLinuxProviderParallelConf(t *testing.T) (s string) {
	data := struct {
		Provider1, Provider2 tfmap
	}{
		testAccProvider, testAccProvider.Copy().Without("host"),
	}

	conf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider1.Serialize | nindent 4 }}
		}
		
		resource "linux_file" "files" {
		    count       = 20
		    path        = "/tmp/linux/file-${count.index}"
		    content     = "file-${count.index}"
		}
		resource "linux_directory" "directories" {
		    count       = 20
		    path        = "/tmp/linux/dir-${count.index}"
		}
		resource "linux_script" "script" {
		    count       = 20
		    lifecycle_commands {
				create = "mkdir -p /tmp/linux/script-dir"
				read = "ls -lhd /tmp/linux/script-dir"
				delete = "rm -rf /tmp/linux/script-dir"
			}
		}
	`)
	s, err := tCompileTemplate(conf, data)
	require.NoError(t, err)
	t.Log(s)
	return
}
