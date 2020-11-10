# linux_script

Manage arbritrary resource by specifying commands that will be executed remotely.

## Example Usage

```hcl
resource "linux_script" "install_package" {
    lifecycle_commands {
        create = "apt install -y $PACKAGE_NAME=$PACKAGE_VERSION"
        read = "apt-cache policy $PACKAGE_NAME | grep 'Installed:' | grep -v '(none)' | awk '{ print $2 }' | xargs | tr -d '\n'"
        update = "apt install -y $PACKAGE_NAME=$PACKAGE_VERSION"
        delete = "apt remove -y $PACKAGE_NAME"
    }
    environment = {
        PACKAGE_NAME = "apache2"
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
- `triggers` - (Optional, string map) Attribute that will trigger resource recreation on changes just like the one in [null_resource](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource#triggers). Default empty map.

### lifecycle_commands

Block that contains commands to be remotely executed respective to terraform's **Create**,**Read**,**Update**, and **Delete** phase. For complex commands, use [the file function](https://www.terraform.io/docs/configuration/functions/file.html). The following arguments are supported:

- `create` - (Required, string) Commands that will be executed in **Create** phase.
- `read` - (Required, string) Commands that will be executed in **Read** phase and after execution of `create` or `update` commands. Terraform will record the output of these commands inside `output` attributes and trigger update/recreation when it changes (in **Read** phase only). If the result of running these commands produce an error, then it will give a signal for resource recreation. In this scenario, user have three options  before applying the changes: (1) do nothing since the resource has indeed become absent, (2) manually modifying the linux machine so no error will be produced in the next run, (3) update the commands. It is recommended that this operations does not do any kind of 'write' operation.
- `update` - (Optional, string) Commands that will be executed in **Update** phase. Previous `output` are accessible from stdin. Omiting this will trigger resource recreation (**Delete** -> **Create**) each time terraform detect changes.
- `delete` - (Required, string) Commands that will be executed in **Delete** phase.

### Resource Update

When any of the `lifecycle_commands` and/or `interpreter` are updated, then nothing will be executed (except for the current `read` commands with existing `interpreter` since it will always be executed before changes are detected). This is to mimic the behavior of an updated provider's instructions, where no previous instructions are executed. Changes to these two arguments must not be followed with changes to other arguments at the same time, or else an error will be thrown.

## Attribute Reference

- `output` - (string) The raw output of `read` commands.
