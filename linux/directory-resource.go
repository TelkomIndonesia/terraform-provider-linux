package linux

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/spf13/cast"
)

const (
	attrDirectoryPath        = "path"
	attrDirectoryOwner       = "owner"
	attrDirectoryGroup       = "group"
	attrDirectoryMode        = "mode"
	attrDirectoryOverwrite   = "overwrite"
	attrDirectoryRecyclePath = "recycle_path"
)

var schemaDirectoryResource = map[string]*schema.Schema{
	attrDirectoryPath: {
		Type:     schema.TypeString,
		Required: true,
	},

	attrDirectoryOwner: {
		Type:     schema.TypeInt,
		Optional: true,
		Default:  0,
	},
	attrDirectoryGroup: {
		Type:     schema.TypeInt,
		Optional: true,
		Default:  0,
	},
	attrDirectoryMode: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  "755",
	},
	attrDirectoryOverwrite: {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "If true, existing directory on remote will be replaced on create or update. This doesn't affect the content of the directory.",
	},
	attrDirectoryRecyclePath: {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "Path to parent directory of a generated-unix-timestamp directory where the directory will be placed on destroy",
	},
}

type handlerDirectoryResource struct{}

func (handlerDirectoryResource) newDirectory(rd *schema.ResourceData) (d *directory) {
	if rd == nil {
		return
	}
	d = &directory{
		path: cast.ToString(rd.Get(attrDirectoryPath)),
		permission: permission{
			owner: cast.ToUint16(rd.Get(attrDirectoryOwner)),
			group: cast.ToUint16(rd.Get(attrDirectoryGroup)),
			mode:  cast.ToString(rd.Get(attrDirectoryMode)),
		},
		overwrite:   cast.ToBool(rd.Get(attrDirectoryOverwrite)),
		recyclePath: cast.ToString(rd.Get(attrDirectoryRecyclePath)),
	}
	return
}

func (handlerDirectoryResource) newDiffedDirectory(rd *schema.ResourceData) (old, new *directory) {
	if rd == nil {
		return
	}
	old, new = &directory{}, &directory{}

	o, n := rd.GetChange(attrDirectoryPath)
	old.path, new.path = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrDirectoryOwner)
	old.permission.owner, new.permission.owner = cast.ToUint16(o), cast.ToUint16(n)

	o, n = rd.GetChange(attrDirectoryGroup)
	old.permission.group, new.permission.group = cast.ToUint16(o), cast.ToUint16(n)

	o, n = rd.GetChange(attrDirectoryMode)
	old.permission.mode, new.permission.mode = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrDirectoryOverwrite)
	old.overwrite, new.overwrite = cast.ToBool(o), cast.ToBool(n)

	o, n = rd.GetChange(attrDirectoryRecyclePath)
	old.recyclePath, new.recyclePath = cast.ToString(o), cast.ToString(n)
	return
}

func (handlerDirectoryResource) updateResourceData(d *directory, rd *schema.ResourceData) (err error) {
	if d == nil {
		rd.SetId("")
		return
	}

	if err = rd.Set(attrDirectoryPath, d.path); err != nil {
		return
	}
	if err = rd.Set(attrDirectoryOwner, d.permission.owner); err != nil {
		return
	}
	if err = rd.Set(attrDirectoryGroup, d.permission.group); err != nil {
		return
	}
	if err = rd.Set(attrDirectoryMode, d.permission.mode); err != nil {
		return
	}
	if err = rd.Set(attrDirectoryOverwrite, d.overwrite); err != nil {
		return
	}
	if err = rd.Set(attrDirectoryRecyclePath, d.recyclePath); err != nil {
		return
	}
	return
}

func (h handlerDirectoryResource) Read(ctx context.Context, rd *schema.ResourceData, i interface{}) (dg diag.Diagnostics) {
	l := i.(linux)
	d, err := l.readDirectory(ctx, cast.ToString(rd.Get(attrDirectoryPath)))
	if err != nil && !errors.Is(err, errPathNotExist) {
		return diag.FromErr(err)
	}

	d.overwrite = cast.ToBool(rd.Get(attrDirectoryOverwrite))
	d.recyclePath = cast.ToString(rd.Get(attrDirectoryRecyclePath))
	if err = h.updateResourceData(d, rd); err != nil {
		diag.FromErr(err)
	}
	return
}

func (h handlerDirectoryResource) Create(ctx context.Context, rd *schema.ResourceData, i interface{}) (dg diag.Diagnostics) {
	l := i.(linux)
	d := h.newDirectory(rd)
	if err := l.createDirectory(ctx, d); err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return diag.FromErr(err)
	}

	rd.SetId(id.String())
	return h.Read(ctx, rd, i)
}

func (h handlerDirectoryResource) Update(ctx context.Context, rd *schema.ResourceData, i interface{}) (dg diag.Diagnostics) {
	l := i.(linux)
	old, new := h.newDiffedDirectory(rd)
	err := l.updateDirectory(ctx, old, new)
	if err != nil {
		_ = h.updateResourceData(old, rd) // WARN: see https://github.com/hashicorp/terraform-plugin-sdk/issues/476
		return diag.FromErr(err)
	}

	return h.Read(ctx, rd, i)
}

func (h handlerDirectoryResource) Delete(ctx context.Context, rd *schema.ResourceData, i interface{}) (d diag.Diagnostics) {
	l := i.(linux)
	if err := l.deleteDirectory(ctx, h.newDirectory(rd)); err != nil {
		return diag.FromErr(err)
	}
	return
}

func directoryResource() *schema.Resource {
	var hdr handlerDirectoryResource
	return &schema.Resource{
		Schema:        schemaDirectoryResource,
		CreateContext: hdr.Create,
		ReadContext:   hdr.Read,
		UpdateContext: hdr.Update,
		DeleteContext: hdr.Delete,
	}
}
