package linuxbox

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/communicator/ssh"
	"github.com/hashicorp/terraform/terraform"
	"github.com/spf13/cast"
)

type LinuxBox struct {
	communicator *ssh.Communicator
	connInfo     map[string]string
}

func newLinuxBoxFromSchema(d *schema.ResourceData) (p LinuxBox, err error) {
	connInfo := map[string]string{
		"type": "ssh",

		attrProviderHost:    cast.ToString(d.Get(attrProviderHost)),
		attrProviderPort:    cast.ToString(d.Get(attrProviderPort)),
		attrProviderHostKey: cast.ToString(d.Get(attrProviderHostKey)),

		attrProviderUser:        cast.ToString(d.Get(attrProviderUser)),
		attrProviderPassword:    cast.ToString(d.Get(attrProviderPassword)),
		attrProviderPrivateKey:  cast.ToString(d.Get(attrProviderPrivateKey)),
		attrProviderCertificate: cast.ToString(d.Get(attrProviderCertificate)),

		attrProviderAgent:         cast.ToString(d.Get(attrProviderAgent)),
		attrProviderAgentIdentity: cast.ToString(d.Get(attrProviderAgentIdentity)),

		attrProviderBastionHost:        cast.ToString(d.Get(attrProviderBastionHost)),
		attrProviderBastionPort:        cast.ToString(d.Get(attrProviderBastionPort)),
		attrProviderBastionHostKey:     cast.ToString(d.Get(attrProviderBastionHostKey)),
		attrProviderBastionUser:        cast.ToString(d.Get(attrProviderBastionUser)),
		attrProviderBastionPassword:    cast.ToString(d.Get(attrProviderBastionPassword)),
		attrProviderBastionPrivateKey:  cast.ToString(d.Get(attrProviderBastionPrivateKey)),
		attrProviderBastionCertificate: cast.ToString(d.Get(attrProviderBastionCertificate)),

		attrProviderScriptPath: cast.ToString(d.Get(attrProviderScriptPath)),
		attrProviderTimeout:    cast.ToString(d.Get(attrProviderTimeout)),
	}
	c, err := ssh.New(&terraform.InstanceState{Ephemeral: terraform.EphemeralState{
		ConnInfo: connInfo,
	}})
	if err != nil {
		return
	}
	if err = c.Connect(nil); err != nil {
		return
	}
	return LinuxBox{communicator: c, connInfo: connInfo}, nil
}

func (p LinuxBox) exec(cmd *remote.Cmd) (err error) {
	if err = p.communicator.Start(cmd); err != nil {
		return
	}
	return cmd.Wait()
}
