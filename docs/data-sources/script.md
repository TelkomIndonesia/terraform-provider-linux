# linux_script

Read arbritrary resource by specifying commands that will be uploaded and executed remotely.

## Example Usage

```hcl
locals {
    package_name = "apache2"
}
resource "linux_script" "install_package" {
    lifecycle_commands {
        read = "apt-cache policy $PACKAGE_NAME | grep 'Installed:' | grep -v '(none)' | awk '{ print $2 }' | xargs | tr -d '\n'"
    }
    environment = {
        PACKAGE_NAME = local.package_name
        PACKAGE_VERSION = "2.4.18-2ubuntu3.4"
    }
}
```

## Argument Reference

The following arguments are supported:

- `lifecycle_commands` - (Required) see [lifecycle_commands](#lifecycle_commands).
- `interpreter` - (Optional, string list) Interpreter for running each `lifecycle_commands`. Default empty list.
- `working_directory` - (Optional, string) The working directory where each `lifecycle_commands` is executed. Default empty string.
- `environment` - (Optional, string map) A list of linux environment that will be available in each `lifecycle_commands`. Default empty map.
- `sensitive_environment` - (Optional, string map) Just like `environment` except they don't show up in log files. In case of duplication,  environment variables defined here will take precedence over the ones in `environment`. Default empty map.

### lifecycle_commands

Block that contains commands to be uploaded and remotely executed in Terraform.

- `read` - (Required, string) Commands that will be executed to obtain data regarding the arbritrary resource. Terraform will record the output of these commands inside `output` attributes.

## Attribute Reference

- `output` - (string) The raw output of `read` commands.
