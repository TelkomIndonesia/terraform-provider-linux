package linux

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxScriptBasic(t *testing.T) {
	conf1 := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			Interpreter: tfList{`"sh"`, `"-c"`},
			Environment: tfmap{
				"FILE": fmt.Sprintf(`"/tmp/linux/%s.yml"`, acctest.RandString(16)),
				"CONTENT": heredoc.Doc(`
					key:
						- key1: "val1"
						  key2: 'val'
						- key1: "val2"
				`),
			},
		},
	}
	conf2 := tfConf{
		Provider: testAccProvider,
		Script:   conf1.Script.Copy(),
	}
	conf2.Script.Environment = conf2.Script.Environment.With("FILE", fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)))

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
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
		        path = linux_script.script.environment["FILE"]
		        content = linux_script.script.environment["CONTENT"]
		
		        path_compare = "${linux_script.script.environment["FILE"]}.compare"
		        path_previous = {{ .Extra.path_previous | default "0"}}
		    }
		    connection {
		        type = "ssh"
		        {{- .Provider.Serialize | nindent 8 }}
		    }
		    provisioner "file" {
		        content = self.triggers["content"]
		        destination = self.triggers["path_compare"]
		    }
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ ! -e "${self.triggers["path_previous"]}"  ] || exit 102
		
		                cmp -s "${self.triggers["path"]}" "${self.triggers["path_compare"]}" || exit 103
		                [ "$( stat -c %u '${self.triggers["path"]}' )" == "0" ] || exit 104
		                [ "$( stat -c %g '${self.triggers["path"]}' )" == "0" ] || exit 105
		                [ "$( stat -c %a '${self.triggers["path"]}' )" == "644" ] || exit 106
		            EOF
		        ]
		    }
		    provisioner "remote-exec" {
		        when = destroy
		        inline = [ "rm -f '${self.triggers["path_compare"]}'" ]
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
		Provider: testAccProvider,
		Script: tfScript{
			Environment: tfmap{
				"FILE":    fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)),
				"CONTENT": `"test"`,
			},
		},
		Extra: tfmap{
			"Version": "1.0.0",
		},
	}
	conf2 := tfConf{
		Provider: testAccProvider,
		Script:   conf1.Script.Copy(),
		Extra:    conf1.Extra.Copy().With("Version", "2.0.0"),
	}
	conf3 := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			Environment: conf2.Script.Environment.Copy().With("FILE", fmt.Sprintf(`"/tmp/linux1/%s"`, acctest.RandString(16))),
		},
		Extra: conf2.Extra.Copy(),
	}
	conf4 := tfConf{
		Provider: testAccProvider,
		Script:   conf2.Script.Copy(),
		Extra:    conf2.Extra.Copy().With("Taint", `\n`),
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptNoUpdateConfig(t, conf1),
			},
			{
				Config: testAccLinuxScriptNoUpdateConfig(t, conf2),
			},
			{
				Config: testAccLinuxScriptNoUpdateConfig(t, conf3),
			},
			{
				Config:             testAccLinuxScriptNoUpdateConfig(t, conf4),
				ExpectNonEmptyPlan: true,
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
			                [ ! -e {{ .Script.Environment.FILE }} ] || exit 100
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
			            "mkdir -p '${self.triggers["directory"]}'"
			        ]
			    }
			    provisioner "remote-exec" {
			        when = destroy
			        inline = [
			            "rm -rf '${self.triggers["directory"]}'"
			        ]
			    }
			}
	
			resource "linux_script" "script" {
			    depends_on = [ null_resource.destroy_validator, null_resource.directory ]  
			    lifecycle_commands {
			        create = <<-EOF
			            echo {{ .Extra.Version }} > /dev/null
			            echo -n "$CONTENT" > "$FILE"
			        EOF
			        read = <<-EOF
			            echo {{ .Extra.Version }} > /dev/null
			            cat "$FILE"
			        EOF
			        delete = <<-EOF
			            echo {{ .Extra.Version }} > /dev/null
			            rm "$FILE"
			        EOF
			    }
			    
			    {{- .Script.Serialize | nindent 4 }}
			}

			resource "null_resource" "create_validator" {
			    triggers = {
			        path = linux_script.script.environment["FILE"]
			        content = linux_script.script.environment["CONTENT"]
		
			        path_compare = "${linux_script.script.environment["FILE"]}.compare"
			        path_previous = {{ .Extra.path_previous | default "0"}}
			    }
			    connection {
			        type = "ssh"
			        {{- .Provider.Serialize | nindent 8 }}
			    }
			    provisioner "file" {
			        content = self.triggers["content"]
			        destination = self.triggers["path_compare"]
			    }
			    provisioner "remote-exec" {
			        inline = [
			            <<-EOF
			                [ ! -e "${self.triggers["path_previous"]}"  ] || exit 101
			
			                cmp -s "${self.triggers["path"]}" "${self.triggers["path_compare"]}" || exit 102
			                [ "$( stat -c %u '${self.triggers["path"]}' )" == "0" ] || exit 103
			                [ "$( stat -c %g '${self.triggers["path"]}' )" == "0" ] || exit 104
			                [ "$( stat -c %a '${self.triggers["path"]}' )" == "644" ] || exit 105
			            EOF
			        ]
			    }
			    provisioner "remote-exec" {
			        when = destroy
			        inline = [ "rm -f '${self.triggers["path_compare"]}'" ]
			    }
			}

			{{ if .Extra.Taint -}}
			resource "null_resource" "taint" {
			    depends_on = [ null_resource.create_validator ]
			    connection {
			        type = "ssh"
			        {{- .Provider.Serialize | nindent 8 }}
			    }
			    provisioner "remote-exec" {
			        inline = [
			            <<-EOF
			                echo -n "{{ .Extra.Taint }}" >> '${ linux_script.script.environment["FILE"] }'
			            EOF
			        ]
			    }
			}
			{{- end }}
		`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxScriptUpdatedScript(t *testing.T) {
	failedCreateDueToReadError := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo"`,
				attrScriptLifecycleCommandRead:   `"cat /nonexist"`,
				attrScriptLifecycleCommandUpdate: `"echo"`,
				attrScriptLifecycleCommandDelete: `"echo"`,
			},
		},
	}
	fixedRead := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo"`,
				attrScriptLifecycleCommandRead:   `"echo '/nonexist'"`,
				attrScriptLifecycleCommandUpdate: `"echo"`,
				attrScriptLifecycleCommandDelete: `"echo"`,
			},
		},
	}
	updatedReadButError := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo"`,
				attrScriptLifecycleCommandRead:   `"cat '/nonexist'"`,
				attrScriptLifecycleCommandUpdate: `"echo 'is now exist' >> '/nonexist' "`,
				attrScriptLifecycleCommandDelete: `"rm '/nonexist'"`,
			},
		},
	}
	restoredRead := tfConf{
		Provider: testAccProvider,
		Script: updatedReadButError.Script.Copy(func(sc *tfScript) {
			sc.LifecycleCommands.
				With(attrScriptLifecycleCommandRead, fixedRead.Script.LifecycleCommands[attrScriptLifecycleCommandRead])
		}),
	}
	onlyUpdateCreate := tfConf{
		Provider: testAccProvider,
		Script: restoredRead.Script.Copy(func(sc *tfScript) {
			sc.LifecycleCommands.With(attrScriptLifecycleCommandCreate, `"echo               "`)
		}),
	}
	onlyUpdateDelete := tfConf{
		Provider: testAccProvider,
		Script: restoredRead.Script.Copy(func(sc *tfScript) {
			sc.LifecycleCommands.With(attrScriptLifecycleCommandDelete, `"rm       '/nonexist'"`)
		}),
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccLinuxScriptUpdatedScriptConfig(t, failedCreateDueToReadError),
				ExpectError: regexp.MustCompile(`cat: can't open`),
			},
			{
				Config: testAccLinuxScriptUpdatedScriptConfig(t, fixedRead),
			},
			{
				Config:      testAccLinuxScriptUpdatedScriptConfig(t, updatedReadButError),
				ExpectError: regexp.MustCompile(`cat: can't open`),
			},
			{
				Config: testAccLinuxScriptUpdatedScriptConfig(t, restoredRead),
			},
			{
				Config: testAccLinuxScriptUpdatedScriptConfig(t, onlyUpdateCreate),
			},
			{
				Config: testAccLinuxScriptUpdatedScriptConfig(t, onlyUpdateDelete),
			},
		},
	})
}

func testAccLinuxScriptUpdatedScriptConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			{{- .Provider.Serialize | nindent 4 }}
		}
		resource "linux_script" "script" {
			{{- .Script.Serialize | nindent 4 }}
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxScriptFailedRead(t *testing.T) {
	// TODO: a proper provisioner to check
	createFile := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo $CONTENT > $FILE"`,
				attrScriptLifecycleCommandRead:   `"cat $FILE"`,
				attrScriptLifecycleCommandUpdate: `"echo $CONTENT > $FILE"`,
				attrScriptLifecycleCommandDelete: `"rm $FILE || true"`,
			},
			Environment: tfmap{
				"FILE":    `"/tmp/linux-test"`,
				"CONTENT": `"test"`,
			},
		},
		Extra: tfmap{
			"ShouldDelete": "true",
		},
	}
	recreateFile := tfConf{
		Provider: testAccProvider,
		Script:   createFile.Script.Copy(),
	}
	fixedReadUpdatedContentScript := tfConf{
		Provider: testAccProvider,
		Script: createFile.Script.Copy(
			func(sc *tfScript) {
				sc.LifecycleCommands.
					With(attrScriptLifecycleCommandRead, `"cat $FILE || echo"`).
					With(attrScriptLifecycleCommandDelete, `"rm $FILE"`)
			},
			func(sc *tfScript) {
				sc.Environment.With("CONTENT", `"test2"`)
			},
		),
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:             testAccLinuxScriptFailedReadConfig(t, createFile),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccLinuxScriptFailedReadConfig(t, recreateFile),
				Check:  resource.TestCheckResourceAttr("linux_script.create_file", "read_failed", "false"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:             testAccLinuxScriptFailedReadConfig(t, createFile),
				ExpectNonEmptyPlan: true,
			},
			{
				Config:             testAccLinuxScriptFailedReadConfig(t, fixedReadUpdatedContentScript),
				Check:              resource.TestCheckResourceAttr("linux_script.create_file", "read_failed", "false"),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccLinuxScriptFailedReadConfig(t, fixedReadUpdatedContentScript),
				Check:  resource.TestCheckResourceAttr("linux_script.create_file", "read_failed", "false"),
			},
		},
	})
}

func testAccLinuxScriptFailedReadConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			{{- .Provider.Serialize | nindent 4 }}
		}
		resource "linux_script" "create_file" {
			{{- .Script.Serialize | nindent 4 }}

			{{ if .Extra.ShouldDelete -}}
			connection {
				type = "ssh"
				{{- .Provider.Serialize | nindent 8 }}
			}
			provisioner "remote-exec" {
				inline = [
					<<-EOF
						rm ${self.environment.FILE}
					EOF
				]
			}
			{{- end }}
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
