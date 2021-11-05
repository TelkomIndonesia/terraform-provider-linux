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
			attrLocalForwardRHost: `"172.67.201.247"`,
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
		    {{- .Provider.Serialize | nindent 4 }}
		}

		data "linux_local_forward" "mockbin" {	
			{{ .LocalForward.Serialize | nindent 4 }}
		}

		resource "null_resource" "output" {
			provisioner "local-exec" {
				command = "curl -f -H 'host: mockbin.org' 'http://${ data.linux_local_forward.mockbin.remote_host }:${ data.linux_local_forward.mockbin.remote_port }/request'"
			}
			provisioner "local-exec" {
				command = "curl -f -H 'host: mockbin.org' 'http://${ data.linux_local_forward.mockbin.host }:${ data.linux_local_forward.mockbin.port }/request'"
			}
		}
	`)

	s, err := conf.compile(tf)
	t.Log(s)
	require.NoError(t, err, "compile template failed")
	return
}
