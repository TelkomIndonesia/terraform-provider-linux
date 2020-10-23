module github.com/TelkomIndonesia/terraform-provider-linuxbox

go 1.15

require (
	github.com/alessio/shellescape v1.3.0
	github.com/google/uuid v1.1.1
	github.com/hashicorp/terraform v0.13.4
	github.com/hashicorp/terraform-plugin-sdk v1.16.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.0.4
	github.com/spf13/cast v1.3.0
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/sys v0.0.0-20200615200032-f1bc736245b1 // indirect
)

replace github.com/hashicorp/terraform => ./workaround/hashicorp/terraform
