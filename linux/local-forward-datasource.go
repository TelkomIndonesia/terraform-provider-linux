package linux

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/phayes/freeport"
)

const (
	attrLocalForwardProviderOverride = "provider_override"
	attrLocalForwardRHost            = "remote_host"
	attrLocalForwardRPort            = "remote_port"
	attrLocalForwardLHost            = "local_host"
	attrLocalForwardLPort            = "local_port"
	attrLocalForwardHost             = "host"
	attrLocalForwardPort             = "port"
)

var schemaLocalForwardDataSource = map[string]*schema.Schema{
	attrLocalForwardProviderOverride: {
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: subSchemaProviderOverride,
		},
	},

	attrLocalForwardRHost: {
		Description: "The remote host",
		Type:        schema.TypeString,
		Required:    true,
	},
	attrLocalForwardRPort: {
		Description: "The remote port",
		Type:        schema.TypeInt,
		Required:    true,
	},

	attrLocalForwardLHost: {
		Description: "The local host",
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "0.0.0.0",
	},
	attrLocalForwardLPort: {
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "The local port",
		Default:     0,
	},

	attrLocalForwardHost: {
		Type:        schema.TypeString,
		Description: "The local host",
		Computed:    true,
	},
	attrLocalForwardPort: {
		Type:        schema.TypeInt,
		Description: "The local port",
		Computed:    true,
	},
}

type handlerLocalForwardDataSource struct {
}

func (h handlerLocalForwardDataSource) Read(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l, err := getLinux(meta.(*linuxPool), rd)
	if err != nil {
		return diag.FromErr(err)
	}

	host := rd.Get(attrLocalForwardLHost)
	port := rd.Get(attrLocalForwardLPort)
	if port == 0 {
		port, err = freeport.GetFreePort()
		if err != nil {
			return diag.FromErr(err)
		}
	}

	err = l.lforwardTCP(ctx,
		fmt.Sprintf("%s:%d", host, port),
		fmt.Sprintf("%s:%d", rd.Get(attrLocalForwardRHost), rd.Get(attrLocalForwardRPort)),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	rd.Set(attrLocalForwardHost, host)
	rd.Set(attrLocalForwardPort, port)
	rd.SetId("static")
	return
}

func localforwardDataSource() *schema.Resource {
	h := handlerLocalForwardDataSource{}
	return &schema.Resource{
		Schema:      schemaLocalForwardDataSource,
		ReadContext: h.Read,
	}
}
