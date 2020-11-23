package linux

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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
		Type:         schema.TypeString,
		Optional:     true,
		Default:      "644",
		ValidateFunc: validation.StringMatch(regexp.MustCompile("[0-7]{3}"), "Invalid linux permission"),
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
		Description: "Path to parent directory of a generated-unix-timestamp directory where the file will be placed on destroy",
	},
}

type handlerFileResource struct{}

func (handlerFileResource) newFile(rd *schema.ResourceData) (f *file) {
	if rd == nil {
		return
	}
	f = &file{
		path:    cast.ToString(rd.Get(attrFilePath)),
		content: cast.ToString(rd.Get(attrFileContent)),
		permission: permission{
			owner: cast.ToUint16(rd.Get(attrFileOwner)),
			group: cast.ToUint16(rd.Get(attrFileGroup)),
			mode:  cast.ToString(rd.Get(attrFileMode)),
		},
		ignoreContent: cast.ToBool(rd.Get(attrFileIgnoreContent)),
		overwrite:     cast.ToBool(rd.Get(attrFileOverwrite)),
		recyclePath:   cast.ToString(rd.Get(attrFileRecyclePath)),
	}
	return
}

func (handlerFileResource) newDiffedFile(rd *schema.ResourceData) (old, new *file) {
	if rd == nil {
		return
	}
	old, new = &file{}, &file{}

	o, n := rd.GetChange(attrFilePath)
	old.path, new.path = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrFileContent)
	old.content, new.content = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrFileOwner)
	old.permission.owner, new.permission.owner = cast.ToUint16(o), cast.ToUint16(n)

	o, n = rd.GetChange(attrFileGroup)
	old.permission.group, new.permission.group = cast.ToUint16(o), cast.ToUint16(n)

	o, n = rd.GetChange(attrFileMode)
	old.permission.mode, new.permission.mode = cast.ToString(o), cast.ToString(n)

	o, n = rd.GetChange(attrFileIgnoreContent)
	old.ignoreContent, new.ignoreContent = cast.ToBool(o), cast.ToBool(n)

	o, n = rd.GetChange(attrFileOverwrite)
	old.overwrite, new.overwrite = cast.ToBool(o), cast.ToBool(n)

	o, n = rd.GetChange(attrFileRecyclePath)
	old.recyclePath, new.recyclePath = cast.ToString(o), cast.ToString(n)
	return
}

func (handlerFileResource) updateResourceData(f *file, rd *schema.ResourceData) (err error) {
	if f == nil {
		rd.SetId("")
		return
	}

	if err = rd.Set(attrFilePath, f.path); err != nil {
		return
	}
	if err = rd.Set(attrFileOwner, f.permission.owner); err != nil {
		return
	}
	if err = rd.Set(attrFileGroup, f.permission.group); err != nil {
		return
	}
	if err = rd.Set(attrFileMode, f.permission.mode); err != nil {
		return
	}
	if err = rd.Set(attrFileOverwrite, f.overwrite); err != nil {
		return
	}
	if err = rd.Set(attrFileRecyclePath, f.recyclePath); err != nil {
		return
	}

	if err = rd.Set(attrFileIgnoreContent, f.ignoreContent); err != nil {
		return
	}
	if f.ignoreContent {
		return
	}
	if err = rd.Set(attrFileContent, f.content); err != nil {
		return
	}
	return
}

func (h handlerFileResource) Read(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	f, err := l.readFile(ctx, cast.ToString(rd.Get(attrFilePath)), cast.ToBool(rd.Get(attrFileIgnoreContent)))
	if err != nil && !errors.Is(err, errPathNotExist) {
		return diag.FromErr(err)
	}

	f.overwrite = cast.ToBool(rd.Get(attrFileOverwrite))
	f.recyclePath = cast.ToString(rd.Get(attrFileRecyclePath))
	if err = h.updateResourceData(f, rd); err != nil {
		diag.FromErr(err)
	}
	return
}

func (h handlerFileResource) Create(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	f := h.newFile(rd)
	if err := l.createFile(ctx, f); err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return diag.FromErr(err)
	}

	rd.SetId(id.String())
	return h.Read(ctx, rd, meta)
}

func (h handlerFileResource) Update(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	old, new := h.newDiffedFile(rd)
	err := l.updateFile(ctx, old, new)
	if err != nil {
		_ = h.updateResourceData(old, rd) // WARN: see https://github.com/hashicorp/terraform-plugin-sdk/issues/476
		return diag.FromErr(err)
	}

	return h.Read(ctx, rd, meta)
}

func (h handlerFileResource) Delete(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	if err := l.deleteFile(ctx, h.newFile(rd)); err != nil {
		return diag.FromErr(err)
	}
	return
}

func fileResource() *schema.Resource {
	var hfr handlerFileResource
	return &schema.Resource{
		Schema:        schemaFileResource,
		CreateContext: hfr.Create,
		ReadContext:   hfr.Read,
		UpdateContext: hfr.Update,
		DeleteContext: hfr.Delete,
	}
}
