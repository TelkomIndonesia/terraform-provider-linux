package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return &schema.Provider{
				Schema:             map[string]*schema.Schema{},
				ResourcesMap:       map[string]*schema.Resource{},
				DataSourcesMap:     map[string]*schema.Resource{},
				ProviderMetaSchema: map[string]*schema.Schema{},
				ConfigureFunc: func(*schema.ResourceData) (interface{}, error) {
					return nil, nil
				},
				ConfigureContextFunc: func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
					return nil, nil
				},
				TerraformVersion: "",
			}
		},
	})
}
