package linux

import (
	"context"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/spf13/cast"
)

const (
	attrProviderID = "id"

	attrProviderHost    = "host"
	attrProviderPort    = "port"
	attrProviderHostKey = "host_key"

	attrProviderUser        = "user"
	attrProviderPassword    = "password"
	attrProviderPrivateKey  = "private_key"
	attrProviderCertificate = "certificate"

	attrProviderAgent         = "agent"
	attrProviderAgentIdentity = "agent_identity"

	attrProviderBastionHost        = "bastion_host"
	attrProviderBastionPort        = "bastion_port"
	attrProviderBastionHostKey     = "bastion_host_key"
	attrProviderBastionUser        = "bastion_user"
	attrProviderBastionPassword    = "bastion_password"
	attrProviderBastionPrivateKey  = "bastion_private_key"
	attrProviderBastionCertificate = "bastion_certificate"

	attrProviderScriptPath = "script_path"
	attrProviderTimeout    = "timeout"
)

var schemaProvider = map[string]*schema.Schema{

	attrProviderHost: {
		Type:        schema.TypeString,
		Description: "The address of the resource to connect to.",
		Optional:    true,
		Default:     "127.0.0.1",
	},
	attrProviderPort: {
		Type:        schema.TypeInt,
		Default:     "22",
		Optional:    true,
		Description: "The port to connect to. Defaults to `22`.",
	},
	attrProviderHostKey: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The public key from the remote host or the signing CA, used to verify the connection.",
	},

	attrProviderUser: {
		Type:        schema.TypeString,
		Default:     "root",
		Optional:    true,
		Description: "The user that we should use for the connection. Defaults to `root`.",
	},
	attrProviderPassword: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The password we should use for the connection.",
		Sensitive:   true,
	},
	attrProviderPrivateKey: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The contents of an SSH key to use for the connection. These can be loaded from a file on disk using the `file` function. This takes preference over the `password` if provided.",
		Sensitive:   true,
	},
	attrProviderCertificate: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The contents of a signed CA Certificate. The certificate argument must be used in conjunction with a `private_key`. These can be loaded from a file on disk using the the `file` function.",
	},

	attrProviderAgent: {
		Type:        schema.TypeBool,
		Optional:    true,
		Description: "Set to `false` to disable using ssh-agent to authenticate.",
	},
	attrProviderAgentIdentity: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The preferred identity from the ssh agent for authentication.",
	},

	attrProviderBastionHost: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "Setting this enables the bastion Host connection. This host will be connected to first, and then the `host` connection will be made from there.",
	},
	attrProviderBastionPort: {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The port to use connect to the bastion host. Defaults to the value of the `port` field.",
	},
	attrProviderBastionHostKey: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The public key from the remote host or the signing CA, used to verify the host connection.",
	},
	attrProviderBastionUser: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The user for the connection to the bastion host. Defaults to the value of the `user` field.",
	},
	attrProviderBastionPassword: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The password we should use for the bastion host. Defaults to the value of the `password` field.",
		Sensitive:   true,
	},
	attrProviderBastionPrivateKey: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The contents of an SSH key file to use for the bastion host. These can be loaded from a file on disk using the `file` function. Defaults to the value of the `private_key` field.",
		Sensitive:   true,
	},
	attrProviderBastionCertificate: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The contents of a signed CA Certificate. The certificate argument must be used in conjunction with a `bastion_private_key`. These can be loaded from a file on disk using the the `file` function.",
	},

	attrProviderScriptPath: {
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The path used to copy scripts meant for remote execution.",
		Default:     "/tmp/linux-%RAND%.sh",
	},
	attrProviderTimeout: {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "5m",
		Description: " The timeout to wait for the connection to become available. Should be provided as a string like `30s` or `5m`. Defaults to 5 minutes.",
	},
}

var subSchemaProviderOverride = func() (m map[string]*schema.Schema) {
	m = map[string]*schema.Schema{
		attrProviderID: {
			Type:        schema.TypeString,
			Description: "Connection instance ID for locking purpose when defined in resource/datasource",
			Required:    true,
		},
	}
	for k, v := range schemaProvider {
		m[k] = v
	}
	return
}()

func newLinuxFromSchema(d *schema.ResourceData) (l *linux, err error) {
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
	return &linux{connInfo: connInfo, commOnce: sync.Once{}}, nil
}

func getLinux(lp *linuxPool, d *schema.ResourceData) (l *linux, err error) {
	var con map[string]string

	pro := cast.ToSlice(d.Get(attrScriptProviderOverride))
	if len(pro) <= 0 {
		return lp.def, nil
	}

	con = cast.ToStringMapString(pro[0])
	return lp.getOrSet(con[attrProviderID], &linux{connInfo: con, commOnce: sync.Once{}})
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: schemaProvider,
		ConfigureContextFunc: func(ctx context.Context, d *schema.ResourceData) (lp interface{}, diags diag.Diagnostics) {
			l, err := newLinuxFromSchema(d)
			if err != nil {
				return nil, diag.FromErr(err)
			}
			lp = &linuxPool{def: l, pool: make(map[string]*linux)}
			return
		},
		DataSourcesMap: map[string]*schema.Resource{
			"linux_script":        scriptDataSource(),
			"linux_local_forward": localforwardDataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"linux_file":      fileResource(),
			"linux_directory": directoryResource(),
			"linux_script":    scriptResource(),
		},
	}
}
