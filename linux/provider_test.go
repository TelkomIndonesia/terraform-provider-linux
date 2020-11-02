package linux

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var provider = tfmap{
	"host":     `"127.0.0.1"`,
	"port":     `"2222"`,
	"user":     `"root"`,
	"password": `"root"`,
}

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"linux": testAccProvider,
	}
}
