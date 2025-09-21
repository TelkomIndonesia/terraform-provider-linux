package linux

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxDataScriptBasic(t *testing.T) {
	file := fmt.Sprintf(`"/tmp/linux-%s.yml"`, acctest.RandString(16))
	conf := tfConf{
		Provider: testAccProvider,
		Script: tfScript{
			Interpreter: tfList{`"sh"`, `"-c"`},
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandRead:   `"cat $FILE"`,
				attrScriptLifecycleCommandCreate: `"echo -n $CONTENT > $FILE"`,
				attrScriptLifecycleCommandUpdate: `"echo -n $CONTENT > $FILE"`,
				attrScriptLifecycleCommandDelete: `"rm $FILE"`,
			},
			Environment: tfmap{
				"FILE":    file,
				"CONTENT": `"helloworld"`,
			},
		},
		DataScript: tfScript{
			Interpreter: tfList{`"sh"`, `"-c"`},
			LifecycleCommands: tfmap{
				attrScriptLifecycleCommandRead: `"cat $FILE"`,
			},
			Environment: tfmap{
				"FILE": file,
			},
		},
	}
	failedScripte := conf.Copy(func(tc *tfConf) {
		tc.DataScript.LifecycleCommands.With(attrScriptLifecycleCommandRead, `"cat $FILE.notexist"`)
	})
	updatedContent := conf.Copy(func(tc *tfConf) {
		tc.Script.Environment.With("CONTENT", `"helloworld1"`)
	})

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxDataScriptBasicConfig(t, conf),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("null_resource.output", "triggers.output", strings.Trim(conf.Script.Environment["CONTENT"], `"`)),
				),
			},
			{
				Config:      testAccLinuxDataScriptBasicConfig(t, failedScripte),
				ExpectError: regexp.MustCompile("No such file or directory"),
			},
			{
				Config: testAccLinuxDataScriptBasicConfig(t, updatedContent),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("null_resource.output", "triggers.output", strings.Trim(updatedContent.Script.Environment["CONTENT"], `"`)),
				),
			},
		},
	})
}

func testAccLinuxDataScriptBasicConfig(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		resource "linux_script" "linux_script" {
			provider = linux.test
		    {{ .Script.Serialize | nindent 4 }}
		}

		data "linux_script" "linux_script" {	
			provider = linux.test
			depends_on = [ linux_script.linux_script ]	
			{{ .DataScript.Serialize | nindent 4 }}
		}

		resource "null_resource" "output" {
			triggers = {
				output = data.linux_script.linux_script.output
			}
		}
	`)

	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
