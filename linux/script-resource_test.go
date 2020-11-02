package linux

import (
	"fmt"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxScriptBasic(t *testing.T) {
	conf1 := tfConf{
		Provider: provider,
		Script: tfScript{
			Environment: tfmap{
				"FILE":    fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)),
				"CONTENT": `"test"`,
			},
		},
	}
	conf2 := tfConf{
		Provider: provider,
		Script:   conf1.Script.Copy(),
	}
	conf2.Script.Environment = conf2.Script.Environment.With("FILE", fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)))

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptBasicConfig(t, conf1),
			},
			{
				Config: testAccLinuxScriptBasicConfig(t, conf2),
			},
		},
	})
}

func testAccLinuxScriptBasicConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "destroy_validator" {
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [
		            <<-EOF
		                [ ! -e {{ .Script.Environment.FILE }} ] || exit 100
		                [ -e {{ .Script.Environment.FILE }}.updated ] || exit 101
		                rm -rf {{ .Script.Environment.FILE }}.updated
		            EOF
		        ]
		    }
		}

		resource "linux_script" "script" {
		    depends_on = [ null_resource.destroy_validator ]  
		
		    lifecycle_commands {
		        create = <<-EOF
		            mkdir -p "$(dirname "$FILE")" && echo -n "$CONTENT" > "$FILE"
		        EOF
		        read = <<-EOF
		            echo "$FILE"
		        EOF
		        update = <<-EOF
		            touch "$FILE".updated
		            mv "$(cat)" "$FILE"
		        EOF
		        delete = <<-EOF
		            rm "$FILE"
		        EOF
		    }
		    
		    {{ .Script.Serialize | nindent 4 }}
		}

		resource "null_resource" "create_validator" {
		    triggers = {
		        path = {{ .Script.Environment.FILE }}
		        content = {{ .Script.Environment.CONTENT }}
		
		        path_compare = format("%s.compare", {{ .Script.Environment.FILE }})
		        path_previous = {{ .Extra.path_previous | default "0"}}
		    }
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers.content
		        destination = self.triggers.path_compare
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e "${self.triggers.path_previous}"  ] || exit 102
		
		                cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 103
		                [ "$( stat -c %u '${self.triggers.path}' )" == "0" ] || exit 104
		                [ "$( stat -c %g '${self.triggers.path}' )" == "0" ] || exit 105
		                [ "$( stat -c %a '${self.triggers.path}' )" == "644" ] || exit 106
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [ "rm -f '${self.triggers.path_compare}'" ]
		    }
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxScriptNoUpdate(t *testing.T) {
	conf1 := tfConf{
		Provider: provider,
		Script: tfScript{
			Environment: tfmap{
				"FILE":    fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)),
				"CONTENT": `"test"`,
			},
		},
	}
	conf2 := tfConf{
		Provider: provider,
		Script:   conf1.Script.Copy(),
	}
	conf2.Script.Environment = conf2.Script.Environment.With("FILE", fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)))
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {},
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptNoUpdateConfig(t, conf1),
			},
			{
				Config: testAccLinuxScriptNoUpdateConfig(t, conf2),
			},
		},
	})
}

func testAccLinuxScriptNoUpdateConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
			provider "linux" {
			    {{- .Provider.Serialize | nindent 4 }}
			}
	
			resource "null_resource" "destroy_validator" {
			    connection {
			        type = "ssh"
			        {{- .Provider.Serialize | nindent 8 }}
			    }
			    provisioner "remote-exec" {
			        when = destroy
			        inline = [
			            <<-EOF
			                [ ! -e "{{ .Script.Environment.FILE }}" ] || exit 100
			            EOF
			        ]
			    }
			}

			resource "null_resource" "directory" {
			    connection {
			        type = "ssh"
			        {{- .Provider.Serialize | nindent 8 }}
			    }
	
			    triggers = {
			        directory = dirname( {{ .Script.Environment.FILE }} )
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
	
			resource "linux_script" "script" {
			    depends_on = [ null_resource.destroy_validator, null_resource.directory ]  
			    lifecycle_commands {
			        create = <<-EOF
			            echo -n "$CONTENT" > "$FILE"
			        EOF
			        read = <<-EOF
			            echo "$FILE"
			        EOF
			        delete = <<-EOF
			            rm "$FILE"
			        EOF
			    }
			    
			    {{- .Script.Serialize | nindent 4 }}
			}

			resource "null_resource" "create_validator" {
			    triggers = {
			        path = {{ .Script.Environment.FILE }}
			        content = {{ .Script.Environment.CONTENT }}
		
			        path_compare = format("%s.compare", {{ .Script.Environment.FILE }})
			        path_previous = {{ .Extra.path_previous | default "0"}}
			    }
			    connection {
			        type = "ssh"
			        {{- .Provider.Serialize | nindent 8 }}
			    }
			    provisioner "file" {
			        content = self.triggers.content
			        destination = self.triggers.path_compare
			    }
			    provisioner "remote-exec" {
			        inline = [
			            <<-EOF
			                [ ! -e "${self.triggers.path_previous}"  ] || exit 102
			
			                cmp -s "${self.triggers.path}" "${self.triggers.path_compare}" || exit 103
			                [ "$( stat -c %u '${self.triggers.path}' )" == "0" ] || exit 104
			                [ "$( stat -c %g '${self.triggers.path}' )" == "0" ] || exit 105
			                [ "$( stat -c %a '${self.triggers.path}' )" == "644" ] || exit 106
			            EOF
			        ]
			    }
			    provisioner "remote-exec" {
			        when = destroy
			        inline = [ "rm -f '${self.triggers.path_compare}'" ]
			    }
			}
		`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
