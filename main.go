package main

import (
	"github.com/TelkomIndonesia/terraform-provider-linuxbox/linuxbox"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: linuxbox.Provider,
	})
}
