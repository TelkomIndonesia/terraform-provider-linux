package linux

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/spf13/cast"
)

type haschange interface {
	HasChange(key string) bool
}

const (
	attrScriptLifecycleCommands      = "lifecycle_commands"
	attrScriptLifecycleCommandCreate = "create"
	attrScriptLifecycleCommandRead   = "read"
	attrScriptLifecycleCommandUpdate = "update"
	attrScriptLifecycleCommandDelete = "delete"
	attrScriptInterpreter            = "interpreter"

	attrScriptTriggers             = "triggers"
	attrScriptEnvironment          = "environment"
	attrScriptSensitiveEnvironment = "sensitive_environment"
	attrScriptWorkingDirectory     = "working_directory"

	attrScriptOutput = "output"

	attrScriptDirtyOutput  = "__dirty_output__"
	attrScriptFaultyOutput = "__faulty_output__"
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
	attrScriptInterpreter: {
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
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
	attrScriptWorkingDirectory: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  ".",
	},

	attrScriptOutput: {
		Type:     schema.TypeString,
		Computed: true,
	},

	attrScriptDirtyOutput: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  "",
		ValidateDiagFunc: func(i interface{}, c cty.Path) (d diag.Diagnostics) {
			return diag.Errorf("`%s` should not be set from configuration.", attrScriptDirtyOutput)
		},
	},
	attrScriptFaultyOutput: {
		Type:     schema.TypeString,
		Optional: true,
		Default:  "",
		ValidateDiagFunc: func(i interface{}, c cty.Path) (d diag.Diagnostics) {
			return diag.Errorf("`%s` should not be set from configuration.", attrScriptFaultyOutput)
		},
	},
}

type handlerScriptResource struct {
}

func (h handlerScriptResource) attrCommands() map[string]bool {
	return map[string]bool{
		attrScriptLifecycleCommands: true,
		attrScriptInterpreter:       true,
	}
}
func (h handlerScriptResource) attrInputs() map[string]bool {
	return map[string]bool{
		attrScriptEnvironment:          true,
		attrScriptSensitiveEnvironment: true,
		attrScriptWorkingDirectory:     true,
	}
}
func (h handlerScriptResource) attrOutputs() map[string]bool {
	return map[string]bool{
		attrScriptOutput: true,
	}
}
func (h handlerScriptResource) attrInternal() map[string]bool {
	return map[string]bool{
		attrScriptDirtyOutput:  true,
		attrScriptFaultyOutput: true,
	}
}
func (h handlerScriptResource) attrs(source ...map[string]bool) (m map[string]bool) {
	m = make(map[string]bool)
	if len(source) > 0 {
		for _, s := range source {
			for k, v := range s {
				m[k] = v
			}
		}
		return
	}
	for k := range h.attrCommands() {
		m[k] = true
	}
	for k := range h.attrInputs() {
		m[k] = true
	}
	for k := range h.attrOutputs() {
		m[k] = true
	}
	for k := range h.attrInternal() {
		m[k] = true
	}
	return
}

func (h handlerScriptResource) changed(rd haschange, attrs map[string]bool) (changed []string) {
	for k := range attrs {
		if rd.HasChange(k) {
			changed = append(changed, k)
		}
	}
	return
}
func (h handlerScriptResource) changedAttrInputs(rd haschange) (changed []string) {
	return h.changed(rd, h.attrInputs())
}
func (h handlerScriptResource) changedAttrCommands(rd haschange) (changed []string) {
	return h.changed(rd, h.attrCommands())
}
func (h handlerScriptResource) changedAttrInternal(rd haschange) (changed []string) {
	return h.changed(rd, h.attrInternal())
}

func (h handlerScriptResource) setNewComputed(rd *schema.ResourceDiff) (err error) {
	for k := range h.attrOutputs() {
		err = rd.SetNewComputed(k) // assume all computed value will changedAttrInputs
		if err != nil {
			return
		}
	}
	return
}
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
	_ = rd.Set(attrScriptDirtyOutput, "")
	_ = rd.Set(attrScriptFaultyOutput, "")
	old := cast.ToString(rd.Get(attrScriptOutput))
	defer func() { _ = rd.Set(attrScriptOutput, old) }() // never change output here, since it will be corrected by create or update.

	err := h.read(ctx, rd, meta.(*linux))
	if errExit := (*remote.ExitError)(nil); errors.As(err, &errExit) {
		_ = rd.Set(attrScriptFaultyOutput, fmt.Sprintf("Faulty output produced:\n\n%s", err))
		return
	}
	if err != nil {
		return diag.FromErr(err)
	}

	new := cast.ToString(rd.Get(attrScriptOutput))
	if old != new {
		_ = rd.Set(attrScriptDirtyOutput, fmt.Sprintf("Dirty output detected:\n\n%s", new))
	}

	return
}

func (h handlerScriptResource) Create(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	l := meta.(*linux)
	sc := h.newScript(rd, l, attrScriptLifecycleCommandCreate)
	if _, err := sc.exec(ctx); err != nil {
		return diag.FromErr(err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return diag.FromErr(err)
	}
	rd.SetId(id.String())

	if err := h.read(ctx, rd, l); err != nil {
		return diag.FromErr(err)
	}
	return
}

// WARN: see https://github.com/hashicorp/terraform-plugin-sdk/issues/476
func (h handlerScriptResource) restoreOldResourceData(rd *schema.ResourceData, except map[string]bool) (err error) {
	for k := range h.attrs() {
		if except != nil && except[k] {
			continue
		}
		o, _ := rd.GetChange(k)
		err = rd.Set(k, o)
		if err != nil {
			return
		}
	}
	return
}

func (h handlerScriptResource) UpdateCommands(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	if rd.HasChange(attrScriptLifecycleCommands + ".0." + attrScriptLifecycleCommandRead) {
		err := h.read(ctx, rd, meta.(*linux))
		if err != nil {
			_ = h.restoreOldResourceData(rd, nil)
			return diag.FromErr(err)
		}
	}

	return
}

func (h handlerScriptResource) Update(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	if len(h.changedAttrCommands(rd)) > 0 {
		return h.UpdateCommands(ctx, rd, meta)
	}

	l := meta.(*linux)
	sc := h.newScript(rd, l, attrScriptLifecycleCommandUpdate)
	oldOutput := cast.ToString(rd.Get(attrScriptOutput))
	sc.stdin = strings.NewReader(oldOutput)
	if _, err := sc.exec(ctx); err != nil {
		_ = h.restoreOldResourceData(rd, nil)
		return diag.FromErr(err)
	}

	if err := h.read(ctx, rd, l); err != nil {
		return diag.FromErr(err)
	}
	return
}

func (h handlerScriptResource) Delete(ctx context.Context, rd *schema.ResourceData, meta interface{}) (d diag.Diagnostics) {
	if cast.ToString(rd.Get(attrScriptFaultyOutput)) != "" {
		return
	}
	l := meta.(*linux)
	sc := h.newScript(rd, l, attrScriptLifecycleCommandDelete)
	if _, err := sc.exec(ctx); err != nil {
		return diag.FromErr(err)
	}
	return
}

func (h handlerScriptResource) CustomizeDiff(c context.Context, rd *schema.ResourceDiff, meta interface{}) (err error) {
	if rd.Id() == "" {
		return // no state
	}

	if cmd := h.changedAttrCommands(rd); len(cmd) > 0 {
		if fbd := h.changedAttrInputs(rd); len(fbd) > 0 {
			return fmt.Errorf("update to '%s' should not be combined with update to other arguments: %s",
				strings.Join(cmd, ","), strings.Join(fbd, ","))
		}

		if rd.HasChange(attrScriptLifecycleCommands + ".0." + attrScriptLifecycleCommandRead) {
			_ = h.setNewComputed(rd) // assume all computed will change
		}
		return // updated commands. let Update handle it.
	}

	if f, _ := rd.GetChange(attrScriptFaultyOutput); cast.ToString(f) != "" {
		_ = rd.ForceNew(attrScriptFaultyOutput) // faulty output, force recreation
		return
	}

	if _, ok := rd.GetOk(attrScriptLifecycleCommands + ".0." + attrScriptLifecycleCommandUpdate); ok {
		if len(h.changedAttrInputs(rd)) > 0 {
			_ = h.setNewComputed(rd) // assume all computed will change
		}
		return // updateable
	}

	for _, key := range h.changed(rd, h.attrs(h.attrInputs(), h.attrInternal())) {
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
