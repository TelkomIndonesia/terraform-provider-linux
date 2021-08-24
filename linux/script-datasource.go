package linux

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var schemaScriptDataSource = func() (m map[string]*schema.Schema) {
	m = make(map[string]*schema.Schema)
	for k, v := range schemaScriptResource {
		switch k {
		default:
			m[k] = v

		case attrScriptLifecycleCommands:
			m[k] = new(schema.Schema)
			*m[k] = *v
			m[k].Elem = &schema.Resource{
				Schema: map[string]*schema.Schema{
					attrScriptLifecycleCommandRead: v.Elem.(*schema.Resource).Schema[attrScriptLifecycleCommandRead],
				},
			}

		case attrScriptTriggers:
		case attrScriptDirtyOutput:
		case attrScriptFaultyOutput:
		}
	}
	return
}()

type handlerScriptDataSource struct {
	hsr handlerScriptResource
}

func (h handlerScriptDataSource) Read(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l, err := getLinux(meta.(*linuxPool), rd)
	if err != nil {
		return diag.FromErr(err)
	}

	err = h.hsr.read(ctx, rd, l)
	if err != nil {
		d = diag.FromErr(err)
	}
	rd.SetId("static")
	return
}

func scriptDataSource() *schema.Resource {
	h := handlerScriptDataSource{hsr: handlerScriptResource{}}
	return &schema.Resource{
		Schema:      schemaScriptDataSource,
		ReadContext: h.Read,
	}
}
