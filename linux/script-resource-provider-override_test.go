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

func TestAccLinuxScriptProviderOverrideBasic(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
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
	conf2 := conf1.Copy(func(tc *tfConf) {
		tc.Script.Environment.With("FILE", fmt.Sprintf(`"/tmp/linux/%s"`, acctest.RandString(16)))
	})

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptProviderOverrideBasicConfig(t, conf1),
			},
			{
				Config: testAccLinuxScriptProviderOverrideBasicConfig(t, conf2),
			},
		},
	})
}

func testAccLinuxScriptProviderOverrideBasicConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "destroy_validator" {
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
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
			
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }

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
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
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

func TestAccLinuxScriptProviderOverrideNoUpdate(t *testing.T) {
	conf1 := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
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
	conf2 := conf1.Copy(func(tc *tfConf) {
		tc.Extra.With("Version", "2.0.0")
	})
	conf3 := conf2.Copy(func(tc *tfConf) {
		tc.Script.Environment.
			With("FILE", fmt.Sprintf(`"/tmp/linux1/%s"`, acctest.RandString(16)))
	})
	conf4 := conf3.Copy(func(tc *tfConf) {
		tc.Script.Triggers.
			With("HELLO", `"world"`)
	})
	conf5 := conf2.Copy(func(tc *tfConf) {
		tc.Extra.With("Taint", `\n`)
	})

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptProviderOverrideNoUpdateConfig(t, conf1),
			},
			{
				Config: testAccLinuxScriptProviderOverrideNoUpdateConfig(t, conf2),
			},
			{
				Config: testAccLinuxScriptProviderOverrideNoUpdateConfig(t, conf3),
			},
			{
				Config: testAccLinuxScriptProviderOverrideNoUpdateConfig(t, conf4),
			},
			{
				Config:             testAccLinuxScriptProviderOverrideNoUpdateConfig(t, conf5),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccLinuxScriptProviderOverrideNoUpdateConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			{{- .Provider.Serialize | nindent 4 }}
		}

		resource "null_resource" "destroy_validator" {
			connection {
				type = "ssh"
				{{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
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
				{{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
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
		    
			provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
			
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
				{{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
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
				{{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
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

func TestAccLinuxScriptProviderOverrideUpdatedScript(t *testing.T) {
	echo := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo -n"`,
				attrScriptLifecycleCommandRead:   `"echo -n"`,
				attrScriptLifecycleCommandUpdate: `"echo -n"`,
				attrScriptLifecycleCommandDelete: `"echo -n"`,
			},
		},
	}
	readUpdated := echo.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.With(attrScriptLifecycleCommandRead, `"echo -n true"`)
	})
	createDeleteUpdated := readUpdated.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.
			With(attrScriptLifecycleCommandCreate, `"echo -n true"`).
			With(attrScriptLifecycleCommandDelete, `"echo -n true"`)
	})
	updateUpdated := createDeleteUpdated.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.
			With(attrScriptLifecycleCommandUpdate, `"echo -n true"`)
	})
	createFileButNotAllowed := updateUpdated.Copy(func(tc *tfConf) {
		// expect to run create only
		tc.Script.LifecycleCommands.With(attrScriptLifecycleCommandCreate, `"echo -n $CONTENT > $FILE"`)
		tc.Script.LifecycleCommands.With(attrScriptLifecycleCommandRead, `"cat $FILE"`)
		tc.Script.LifecycleCommands.Without(attrScriptLifecycleCommandUpdate)
		tc.Script.Environment.With("FILE", `"/tmp/hello"`)
		tc.Script.Environment.With("CONTENT", `"world"`)
	})
	createFilePt1 := createFileButNotAllowed.Copy(func(tc *tfConf) {
		tc.Script.Environment.Without("FILE", "CONTENT")
	})
	createFilePt2 := createFilePt1.Copy(func(tc *tfConf) {
		tc.Script.Environment = createFileButNotAllowed.Script.Environment.Copy()
	})
	UpdateFilePt1 := createFilePt2.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.With(attrScriptLifecycleCommandUpdate, `"echo -n '\n'$CONTENT >> $FILE"`)
		tc.Script.LifecycleCommands.With(attrScriptLifecycleCommandDelete, `"rm $FILE"`)
	})
	UpdateFilePt2 := UpdateFilePt1.Copy(func(tc *tfConf) {
		tc.Script.Environment = createFilePt2.Script.Environment.Copy().With("CONTENT", `"world1"`)
	})
	interpreterUpdated := UpdateFilePt2.Copy(func(tc *tfConf) {
		tc.Script.Interpreter = tfList{`"/bin/sh"`}
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  testAccPreCheckConnection(t),
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, echo),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", ""),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, readUpdated),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "true"),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, createDeleteUpdated),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "true"),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, updateUpdated),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "true"),
			},
			{
				Config:      testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, createFileButNotAllowed),
				ExpectError: regexp.MustCompile(`should not be combined with update to other arguments`),
			},
			{
				Config:      testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, createFileButNotAllowed),
				ExpectError: regexp.MustCompile(`should not be combined with update to other arguments`),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, createFilePt1),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", ""),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, createFilePt2),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "world"),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, UpdateFilePt1),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "world"),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, UpdateFilePt2),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "world\nworld1"),
			},
			{
				Config: testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t, interpreterUpdated),
				Check:  resource.TestCheckResourceAttr("linux_script.script", "output", "world\nworld1"),
			},
		},
	})
}

func testAccLinuxScriptProviderOverrideUpdatedScriptConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			{{- .Provider.Serialize | nindent 4 }}
		}
		resource "linux_script" "script" {
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
			{{- .Script.Serialize | nindent 4 }}
		}
	`)
	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}

func TestAccLinuxScriptProviderOverrideFailedRead(t *testing.T) {
	createFile := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo -n $CONTENT > $FILE"`,
				attrScriptLifecycleCommandRead:   `"cat $FILE"`,
				attrScriptLifecycleCommandUpdate: `"echo -n '\n'$CONTENT >> $FILE"`,
				attrScriptLifecycleCommandDelete: `"rm $FILE"`,
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
	scriptUnchanged := createFile.Copy(func(tc *tfConf) {
		tc.Extra.Without("ShouldDelete")
	})

	scriptUpdatedWithOtherArguments := createFile.Copy(func(tc *tfConf) {
		tc.Extra.Without("ShouldDelete")
		tc.Script.LifecycleCommands.
			With(attrScriptLifecycleCommandRead, `"cat $FILE || echo"`)
		tc.Script.Environment.With("CONTENT", `"test2"`)
	})
	scriptUpdated := scriptUpdatedWithOtherArguments.Copy(func(tc *tfConf) {
		tc.Script.Environment = createFile.Script.Environment
	})
	contentUpdated := scriptUpdated.Copy(func(tc *tfConf) {
		tc.Script.Environment = scriptUpdatedWithOtherArguments.Script.Environment
	})

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:             testAccLinuxScriptProviderOverrideFailedReadConfig(t, createFile),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccLinuxScriptProviderOverrideFailedReadConfig(t, scriptUnchanged),
				Check:  resource.TestCheckResourceAttr("linux_script.create_file", "output", "test"),
			},
		},
	})
	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:             testAccLinuxScriptProviderOverrideFailedReadConfig(t, createFile),
				ExpectNonEmptyPlan: true,
			},
			{
				Config:      testAccLinuxScriptProviderOverrideFailedReadConfig(t, scriptUpdatedWithOtherArguments),
				ExpectError: regexp.MustCompile(`should not be combined with update to other arguments`),
			},
			{
				Config: testAccLinuxScriptProviderOverrideFailedReadConfig(t, scriptUpdated),
			},
			{
				Config: testAccLinuxScriptProviderOverrideFailedReadConfig(t, contentUpdated),
				Check:  resource.TestCheckResourceAttr("linux_script.create_file", "output", "\ntest2"),
			},
		},
	})
}

func testAccLinuxScriptProviderOverrideFailedReadConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}
		resource "linux_script" "create_file" {
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    
		    {{- .Script.Serialize | nindent 4 }}

		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		
		    {{ if .Extra.ShouldDelete -}}
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                rm ${self.environment.FILE}
		            EOF
		        ]
		    }
		    {{- else -}}
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                [ "$(cat ${self.environment.FILE})" == "${self.environment.CONTENT}" ] || exit 100
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

func TestAccLinuxScriptProviderOverrideComputedDependent(t *testing.T) {
	createFile := tfConf{
		Provider:         testAccOverridenProvider,
		ProviderOverride: testAccProvider.Copy().With("id", `"someid"`),
		Script: tfScript{
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandCreate: `"echo -n $CONTENT > $FILE"`,
				attrScriptLifecycleCommandRead:   `"echo $CONTENT"`,
				attrScriptLifecycleCommandUpdate: `"echo -n $CONTENT > $FILE"`,
				attrScriptLifecycleCommandDelete: `"rm $FILE"`,
			},
			Environment: tfmap{
				"FILE":    `"/tmp/linux-test"`,
				"CONTENT": `"test"`,
			},
		},
	}

	updateContentAndAddDependent := createFile.Copy(func(tc *tfConf) {
		tc.Script.Environment.With("CONTENT", `"test1"`)
		tc.Extra.With("OutputDependent", "true")
	})

	updateCommandsRemoveDependent := updateContentAndAddDependent.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.With("read", `"echo -n $CONTENT"`)
		tc.Extra.Without("OutputDependent")
	})

	reAddDependent := updateCommandsRemoveDependent.Copy(func(tc *tfConf) {
		tc.Extra.With("OutputDependent", "true")
	})

	updateCommands := reAddDependent.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.With("read", `"echo $CONTENT.wrong"`)
	})
	reupdateButErrorCommands := updateCommands.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.With("read", `"cat $FILE && exit 1"`)
	})
	reupdateCommands := updateCommands.Copy(func(tc *tfConf) {
		tc.Script.LifecycleCommands.With("read", `"cat $FILE"`)
	})

	taintResource := reupdateCommands.Copy(func(tc *tfConf) {
		tc.Extra.With("Tainter", "true").
			Without("OutputDependent")
	})
	taintedWithDependentResource := taintResource.Copy(func(tc *tfConf) {
		tc.Extra.With("OutputDependent", "true")
	})

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, createFile),
			},
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, updateContentAndAddDependent),
			},
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, updateCommandsRemoveDependent),
			},
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, reAddDependent),
			},
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, updateCommands),
			},
			{
				Config:      testAccLinuxScriptProviderOverrideComputedConfig(t, reupdateButErrorCommands),
				ExpectError: regexp.MustCompile(""),
				Check:       resource.TestCheckResourceAttr("linux_script.linux_script", "lifecycle_commands.0.read", "echo $CONTENT.wrong"),
			},
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, reupdateCommands),
				Check:  resource.TestCheckResourceAttr("null_resource.output", "triggers.output", "test1"),
			},
			{
				Config:             testAccLinuxScriptProviderOverrideComputedConfig(t, taintResource),
				Check:              resource.TestCheckResourceAttr("linux_script.linux_script", "output", "test1"),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccLinuxScriptProviderOverrideComputedConfig(t, taintedWithDependentResource),
				Check:  resource.TestCheckResourceAttr("null_resource.output", "triggers.output", "test1"),
			},
		},
	})
}

func testAccLinuxScriptProviderOverrideComputedConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "linux_script" "linux_script" {
		    provider_override {
		        {{- .ProviderOverride.Serialize | nindent 8 }}
		    }
		    {{- .Script.Serialize | nindent 4 }}
		}

		{{ if .Extra.OutputDependent -}}
		resource "null_resource" "output" {
		    triggers = {
		       output = linux_script.linux_script.output
		    }
		}
		{{- end }}
		
		{{ if .Extra.Tainter -}}
		resource "null_resource" "tainter" {
		    connection {
		        type = "ssh"
		        {{- ((.ProviderOverride.Copy).Without "id").Serialize | nindent 8 }}
		    }
		
		    
		    provisioner "remote-exec" {
		        inline = [
		            <<-EOF
		                echo "taint" > ${linux_script.linux_script.environment.FILE}
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
