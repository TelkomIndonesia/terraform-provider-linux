package linuxbox

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/spf13/cast"
)

const (
	attrFilePath          = "path"
	attrFileContent       = "content"
	attrFileOwner         = "owner"
	attrFileGroup         = "group"
	attrFileMode          = "mode"
	attrFileIgnoreContent = "ignore_content"
	attrFileOverwrite     = "overwrite"
	attrFileRecyclePath   = "recycle_path"
)

var schemaFileResource = map[string]*schema.Schema{
	attrFilePath: {
		Type:     schema.TypeString,
		Required: true,
	},
	attrFileContent: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  "",
		DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
			return cast.ToBool(d.Get(attrFileIgnoreContent))
		},
	},
	attrFileOwner: {
		Type:     schema.TypeInt,
		Optional: true,
		Default:  0,
	},
	attrFileGroup: {
		Type:     schema.TypeInt,
		Optional: true,
		Default:  0,
	},
	attrFileMode: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  "755",
	},
	attrFileIgnoreContent: {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "If true, `content` will be ignored and won't be included in schema diff",
	},
	attrFileOverwrite: {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "If true, existing file on remote will be replaced on create or update",
	},
	attrFileRecyclePath: {
		Type:        schema.TypeString,
		Optional:    true,
		Default:     "",
		Description: "Path to parent directory of a generated-unix-timestamp folder where the file will be placed when destroyed",
	},
}

type fileResourceHandler struct{}

var frh fileResourceHandler

func (fileResourceHandler) Read(ctx context.Context, rd *schema.ResourceData, i interface{}) (d diag.Diagnostics) {
	l := i.(LinuxBox)
	f, err := l.ReadFile(ctx, cast.ToString(rd.Get(attrFilePath)), cast.ToBool(rd.Get(attrFileIgnoreContent)))
	if err != nil && !errors.Is(err, errPathNotExist) {
		return diag.FromErr(err)
	}

	f.overwrite = cast.ToBool(rd.Get(attrFileOverwrite))
	f.recyclePath = cast.ToString(rd.Get(attrFileRecyclePath))
	if err = f.setResourceData(rd); err != nil {
		diag.FromErr(err)
	}
	return
}

func (frh fileResourceHandler) Create(ctx context.Context, rd *schema.ResourceData, i interface{}) (d diag.Diagnostics) {
	l := i.(LinuxBox)
	f := newFileFromResourceData(rd)
	if err := l.CreateFile(ctx, f); err != nil {
		return diag.FromErr(err)
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return diag.FromErr(err)
	}

	rd.SetId(id.String())
	return frh.Read(ctx, rd, i)
}

func (frh fileResourceHandler) Update(ctx context.Context, rd *schema.ResourceData, i interface{}) (d diag.Diagnostics) {
	l := i.(LinuxBox)
	old, new := newDiffedFileFromResourceData(rd)
	if err := l.UpdateFile(ctx, old, new); err != nil {
		_ = old.setResourceData(rd) // revert state
		return diag.FromErr(err)
	}

	return frh.Read(ctx, rd, i)
}

func (fileResourceHandler) Delete(ctx context.Context, rd *schema.ResourceData, i interface{}) (d diag.Diagnostics) {
	l := i.(LinuxBox)
	if err := l.DeleteFile(ctx, newFileFromResourceData(rd)); err != nil {
		return diag.FromErr(err)
	}
	return
}

func fileResource() *schema.Resource {
	return &schema.Resource{
		Schema:        schemaFileResource,
		CreateContext: frh.Create,
		ReadContext:   frh.Read,
		UpdateContext: frh.Update,
		DeleteContext: frh.Delete,
	}
}
