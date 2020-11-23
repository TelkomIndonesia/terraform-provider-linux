module github.com/TelkomIndonesia/terraform-provider-linux

go 1.15

require (
	github.com/MakeNowJust/heredoc v1.0.0
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/alessio/shellescape v1.3.1
	github.com/google/uuid v1.1.2
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/terraform v0.13.5
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.3.0
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/spf13/cast v1.3.1
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
)

replace github.com/hashicorp/terraform => ./internal/workaround/hashicorp/terraform
