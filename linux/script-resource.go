package linux

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/spf13/cast"
)

const (
	attrScriptLifecycleCommands      = "lifecycle_commands"
	attrScriptLifecycleCommandCreate = "create"
	attrScriptLifecycleCommandRead   = "read"
	attrScriptLifecycleCommandUpdate = "update"
	attrScriptLifecycleCommandDelete = "delete"
	attrScriptTriggers               = "triggers"
	attrScriptEnvironment            = "environment"
	attrScriptSensitiveEnvironment   = "sensitive_environment"
	attrScriptInterpreter            = "interpreter"
	attrScriptWorkingDirectory       = "working_directory"
	attrScriptDirty                  = "dirty"
	attrScriptReadFailed             = "read_failed"
	attrScriptOutput                 = "output"
)

var schemaScriptResource = map[string]*schema.Schema{
	attrScriptLifecycleCommands: {
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				attrScriptLifecycleCommandCreate: {
					Type:     schema.TypeString,
					Required: true,
				},
				attrScriptLifecycleCommandUpdate: {
					Type:     schema.TypeString,
					Optional: true,
				},
				attrScriptLifecycleCommandRead: {
					Type:     schema.TypeString,
					Required: true,
				},
				attrScriptLifecycleCommandDelete: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	},
	attrScriptTriggers: {
		Type:     schema.TypeMap,
		Optional: true,
		ForceNew: true,
	},
	attrScriptEnvironment: {
		Type:     schema.TypeMap,
		Optional: true,
		Elem:     schema.TypeString,
	},
	attrScriptSensitiveEnvironment: {
		Type:      schema.TypeMap,
		Optional:  true,
		Elem:      schema.TypeString,
		Sensitive: true,
	},
	attrScriptInterpreter: {
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	},
	attrScriptWorkingDirectory: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  ".",
	},
	attrScriptDirty: {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
	},
	attrScriptReadFailed: {
		Type:     schema.TypeBool,
		Optional: true,
		Default:  false,
	},
	attrScriptOutput: {
		Type:     schema.TypeString,
		Computed: true,
	},
}

type handlerScriptResource struct{}

func (h handlerScriptResource) newScript(rd *schema.ResourceData, l *linux, attrLifeCycle string) (s *script) {
	if rd == nil {
		return
	}

	lc := cast.ToSlice(rd.Get(attrScriptLifecycleCommands))[0]
	s = &script{
		l: l,

		workdir:     cast.ToString(rd.Get(attrScriptWorkingDirectory)),
		env:         cast.ToStringMapString(rd.Get(attrScriptEnvironment)),
		interpreter: cast.ToStringSlice(rd.Get(attrScriptInterpreter)),
		body:        cast.ToStringMapString(lc)[attrLifeCycle],
	}
	for k, v := range cast.ToStringMapString(rd.Get(attrScriptSensitiveEnvironment)) {
		s.env[k] = v
	}
	return
}

func (h handlerScriptResource) read(ctx context.Context, rd *schema.ResourceData, l *linux) (err error) {
	sc := h.newScript(rd, l, attrScriptLifecycleCommandRead)
	res, err := sc.exec(ctx)
	if err != nil {
		return
	}
	if err = rd.Set(attrScriptOutput, res); err != nil {
		return
	}
	return
}

func (h handlerScriptResource) Read(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	old := cast.ToString(rd.Get(attrScriptOutput))

	err := h.read(ctx, rd, meta.(*linux))
	var errExit *remote.ExitError
	if errors.As(err, &errExit) {
		_ = rd.Set(attrScriptReadFailed, true)
		return
	}

	new := cast.ToString(rd.Get(attrScriptOutput))
	if new == "" {
		rd.SetId("")
		return
	}
	if err := rd.Set(attrScriptDirty, old != new); err != nil {
		return diag.FromErr(err)
	}
	if err := rd.Set(attrScriptReadFailed, false); err != nil {
		return diag.FromErr(err)
	}
	return
}

func (h handlerScriptResource) Create(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	sc := h.newScript(rd, l, attrScriptLifecycleCommandCreate)
	if _, err := sc.exec(ctx); err != nil {
		return diag.FromErr(err)
	}

	if err := h.read(ctx, rd, l); err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return diag.FromErr(err)
	}
	rd.SetId(id.String())
	return
}

// WARN: see https://github.com/hashicorp/terraform-plugin-sdk/issues/476
func (h handlerScriptResource) restoreOldResourceData(rd *schema.ResourceData) (err error) {
	for _, k := range []string{
		attrScriptLifecycleCommands,
		attrScriptEnvironment,
		attrScriptSensitiveEnvironment,
		attrScriptInterpreter,
		attrScriptWorkingDirectory,
		attrScriptOutput,
	} {
		o, _ := rd.GetChange(k)
		err = rd.Set(k, o)
		if err != nil {
			return
		}
	}
	return
}

func (h handlerScriptResource) getChangedKeys(rd *schema.ResourceData) (ks []string) {
	if rd == nil || rd.State() == nil {
		return
	}
	for k := range rd.State().Attributes {
		if rd.HasChange(k) {
			ks = append(ks, k)
		}
	}
	return
}

func (h handlerScriptResource) shouldIgnoreUpdate(rd *schema.ResourceData) bool {
	for _, k := range h.getChangedKeys(rd) {
		if k != attrScriptLifecycleCommands+".0."+attrScriptLifecycleCommandCreate &&
			k != attrScriptLifecycleCommands+".0."+attrScriptLifecycleCommandDelete {
			return false
		}
	}
	return true
}

func (h handlerScriptResource) Update(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	if h.shouldIgnoreUpdate(rd) {
		return
	}

	if rd.HasChange(attrScriptLifecycleCommands + ".0." + attrScriptLifecycleCommandRead) {
		scNew := rd.Get(attrScriptLifecycleCommands)
		_ = h.restoreOldResourceData(rd)

		scOld := rd.Get(attrScriptLifecycleCommands)
		_ = rd.Set(attrScriptLifecycleCommands, scNew)

		err := h.read(ctx, rd, meta.(*linux))
		if err != nil {
			_ = rd.Set(attrScriptLifecycleCommands, scOld)
			d = diag.FromErr(err)
		}
		return
	}

	l := meta.(*linux)
	sc := h.newScript(rd, l, attrScriptLifecycleCommandUpdate)
	oldOutput := cast.ToString(rd.Get(attrScriptOutput))
	sc.stdin = strings.NewReader(oldOutput)
	if _, err := sc.exec(ctx); err != nil {
		_ = h.restoreOldResourceData(rd)
		return diag.FromErr(err)
	}

	if err := h.read(ctx, rd, l); err != nil {
		return diag.FromErr(err)
	}
	return
}

func (h handlerScriptResource) Delete(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	sc := h.newScript(rd, l, attrScriptLifecycleCommandDelete)
	if _, err := sc.exec(ctx); err != nil {
		return diag.FromErr(err)
	}
	return
}

func (h handlerScriptResource) CustomizeDiff(c context.Context, rd *schema.ResourceDiff, meta interface{}) (err error) {
	if rd.Id() == "" {
		return
	}

	if f, _ := rd.GetChange(attrScriptReadFailed); cast.ToBool(f) &&
		!rd.HasChange(attrScriptLifecycleCommands+".0."+attrScriptLifecycleCommandRead) {
		_ = rd.ForceNew(attrScriptReadFailed)
	}
	if _, ok := rd.GetOk(attrScriptLifecycleCommands + ".0." + attrScriptLifecycleCommandUpdate); ok {
		return
	}
	for _, key := range rd.GetChangedKeysPrefix("") {
		if key == attrScriptLifecycleCommands+".0."+attrScriptLifecycleCommandRead ||
			key == attrScriptLifecycleCommands+".0."+attrScriptLifecycleCommandDelete {
			continue // should not trigger force new
		}
		if strings.HasPrefix(key, attrScriptTriggers) {
			continue // already force new
		}

		// need to remove index from map and list
		switch {
		case strings.HasPrefix(key, attrScriptEnvironment):
			key = strings.Split(key, ".")[0]
		case strings.HasPrefix(key, attrScriptSensitiveEnvironment):
			key = strings.Split(key, ".")[0]
		case strings.HasPrefix(key, attrScriptInterpreter):
			key = strings.Split(key, ".")[0]
		}

		err = rd.ForceNew(key)
		if err != nil {
			return
		}
	}
	return
}

func scriptResource() *schema.Resource {
	var h handlerScriptResource
	return &schema.Resource{
		Schema:        schemaScriptResource,
		CreateContext: h.Create,
		ReadContext:   h.Read,
		UpdateContext: h.Update,
		DeleteContext: h.Delete,
		CustomizeDiff: h.CustomizeDiff,
	}
}
