package linux

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

var testAccProviders map[string]*schema.Provider

var provider = tfmap{
	"host":     `"127.0.0.1"`,
	"port":     `"2222"`,
	"user":     `"root"`,
	"password": `"root"`,
}

func init() {
	testAccProviders = map[string]*schema.Provider{
		"linux": Provider(),
	}
}

func TestAccLinuxProviderUnknownValue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: testAccProviders,
		PreCheck: func() {

		},
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
		provider, provider.Copy().Without("host"),
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
		        create = "echo -n"
		        read = "echo -n 'hi'"
		        delete = "echo -n"
		    }
		}
	`)
	s, err := tCompileTemplate(conf, data)
	require.NoError(t, err)
	t.Log(s)
	return
}
