package linuxbox

import (
	"context"
	"errors"

	"github.com/google/uuid"
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
	attrFileReplace       = "replace"
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
	attrFileReplace: {
		Type:        schema.TypeBool,
		Optional:    true,
		Default:     false,
		Description: "If true, existing file on remote will be replaced on create or update (when path changed)",
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

func (fileResourceHandler) Read(rd *schema.ResourceData, i interface{}) (err error) {
	l := i.(LinuxBox)
	f, err := l.ReadFile(context.Background(), cast.ToString(rd.Get(attrFilePath)))
	if err != nil && !errors.Is(err, errPathNotExist) {
		return
	}
	return f.setResourceData(rd)
}

func (frh fileResourceHandler) Create(rd *schema.ResourceData, i interface{}) (err error) {
	l := i.(LinuxBox)
	f := newFileFromResourceData(rd)
	err = l.CreateFile(context.Background(), f)
	if err != nil {
		return
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return
	}
	rd.SetId(id.String())
	rd.SetConnInfo(l.connInfo)

	return frh.Read(rd, i)
}

func (frh fileResourceHandler) Update(rd *schema.ResourceData, i interface{}) (err error) {
	l := i.(LinuxBox)
	old, new := newDiffedFileFromResourceData(rd)
	if err = l.UpdateFile(context.Background(), old, new); err != nil {
		return
	}

	return frh.Read(rd, i)
}

func (fileResourceHandler) Delete(rd *schema.ResourceData, i interface{}) (err error) {
	l := i.(LinuxBox)
	if err = l.DeleteFile(context.Background(), newFileFromResourceData(rd)); err != nil {
		return
	}
	var z *File
	return z.setResourceData(rd)
}

func fileResource() *schema.Resource {
	return &schema.Resource{
		Schema: schemaFileResource,
		Create: frh.Create,
		Read:   frh.Read,
		Update: frh.Update,
		Delete: frh.Delete,
	}
}
