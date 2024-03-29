# linux_script

Manage arbritrary resource by specifying commands that will be uploaded and executed remotely.

## Example Usage

```hcl
locals {
    package_name = "apache2"
}
resource "linux_script" "install_package" {
    lifecycle_commands {
        create = "apt update && apt install -y $PACKAGE_NAME=$PACKAGE_VERSION"
        read = "apt-cache policy $PACKAGE_NAME | grep 'Installed:' | grep -v '(none)' | awk '{ print $2 }' | xargs | tr -d '\n'"
        update = "apt update && apt install -y $PACKAGE_NAME=$PACKAGE_VERSION"
        delete = "apt remove -y $PACKAGE_NAME"
    }
    environment = {
        PACKAGE_NAME = local.package_name
        PACKAGE_VERSION = "2.4.18-2ubuntu3.4"
    }
    triggers = {
        PACKAGE_NAME = local.package_name"
    }
}
```

## Argument Reference

The following arguments are supported:

- `provider_override` - (Optional) see [provider_override](../#provider-override).
- `lifecycle_commands` - (Required) see [lifecycle_commands](#lifecycle_commands).
- `interpreter` - (Optional, string list) Interpreter for running each `lifecycle_commands`. Default empty list.
- `working_directory` - (Optional, string) The working directory where each `lifecycle_commands` is executed. Default empty string.
- `environment` - (Optional, string map) A list of linux environment that will be available in each `lifecycle_commands`. Default empty map.
- `sensitive_environment` - (Optional, string map) Just like `environment` except they don't show up in log files. In case of duplication,  environment variables defined here will take precedence over the ones in `environment`. Default empty map.
- `triggers` - (Optional, string map) Attribute that will trigger resource recreation on changes just like the one in [null_resource](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource#triggers). Default empty map.

### lifecycle_commands

Block that contains commands to be uploaded and remotely executed respective to the terraform's [**Create**, **Read**, **Update**, and **Delete** phase](https://learn.hashicorp.com/tutorials/terraform/provider-use?in=terraform/providers). For complex commands, use [the file function](https://www.terraform.io/docs/configuration/functions/file.html). The following arguments are supported:

- `create` - (Required, string) Commands that will be executed in **Create** phase.
- `read` - (Required, string) Commands that will be executed in **Read** phase and after execution of `create` or `update` commands in the respective phase. Terraform will record the output of these commands inside `output` attributes and trigger update/recreate when it changes in **Read** phase. If the result of running these commands instead produce an error, then it will give a signal to recreate the resource. In this scenario, user have three options before applying the changes: (1) do nothing and apply the changes since the resource has indeed become absent, (2) manually modifying the linux machine so no error will be produced in the next run, or (3) update the commands. If (1) is choosen then `delete` script will not be executed in **Delete** phases. It is recommended that this operations does not do any kind of 'write' operation or at least safe to be retried.
- `update` - (Optional, string) Commands that will be executed in **Update** phase. The previous `output` are accessible from stdin. Note that to produce a consistent plan especially when `output` becomes a dependency for other objects, the commands should affect the value of `output` the same way as if executing the `create` commands on non-existent resource. Omiting this will instead tell terraform to recreate the resource each time it detect changes.
- `delete` - (Required, string) Commands that will be executed in **Delete** phase.

### Updating Resource

This resource is somewhat different from regular terraform resource because it does not only define the information about the actual resource, but also the instructions to CRUD the resource. Among these arguments, `lifecycle_commands` and `interpreter` are considered as instructions while the rest are considered as the actual data. A special course of actions must be taken when these arguments are updated, or else user would get undesired behavior such as `update` command being executed when updating only the `delete` commands.

As such, if `lifecycle_commands` and/or `interpreter` are updated, it will first execute the current `read` commands with the existing `interpreter` (since this is unavoidable) and then either the new `read` commands if its changes or no commands at all. At the same time, no changes to other arguments are allowed, or else an error will be thrown. When successfully updated through `terraform apply`, the next terraform execution will use these new instructions and update to other arguments are allowed.

## Attribute Reference

- `output` - (string) The raw output of `read` commands.
