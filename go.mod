module github.com/TelkomIndonesia/terraform-provider-linux

go 1.15

require (
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/alessio/shellescape v1.3.0
	github.com/google/uuid v1.1.2
	github.com/hashicorp/terraform v0.13.5
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.1.0
	github.com/spf13/cast v1.3.1
	golang.org/x/net v0.0.0-20201027133719-8eef5233e2a1
)

replace github.com/hashicorp/terraform => ./internal/workaround/hashicorp/terraform
