package linux

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestAccLinuxLocalForwardDatasourceBasic(t *testing.T) {
	conf := tfConf{
		Provider: testAccProvider,
		LocalForward: tfmap{
			attrLocalForwardRHost: `"34.160.111.145"`,
			attrLocalForwardRPort: `"80"`,
		},
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{"null": {}},
		PreCheck:          testAccPreCheckConnection(t),
		Providers:         testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccLinuxLocalForwardDatasourceBasic(t, conf),
			},
		},
	})
}

func testAccLinuxLocalForwardDatasourceBasic(t *testing.T, conf tfConf) (s string) {
	tf := heredoc.Doc(`
		provider "linux" {
			alias = "test"
		    {{- .Provider.Serialize | nindent 4 }}
		}

		data "linux_local_forward" "test" {	
			provider = "linux.test"
			{{ .LocalForward.Serialize | nindent 4 }}
		}

		resource "null_resource" "output" {
			provisioner "local-exec" {
				command = "curl -f -H 'host: ifconfig.me' 'http://${ data.linux_local_forward.test.remote_host }:${ data.linux_local_forward.test.remote_port }'"
			}
			provisioner "local-exec" {
				command = "curl -f -H 'host: ifconfig.me' 'http://${ data.linux_local_forward.test.host }:${ data.linux_local_forward.test.port }'"
			}
		}
	`)

	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
