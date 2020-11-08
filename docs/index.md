# Linux Provider

Provider for managing linux machine through SSH connection.

## Example Usage

```hcl
provider "linux" {
    host = "127.0.0.1"
    port = 22
    user = "root"
    password = "root"
}

resource "linux_directory" "directory" {
    path = "/tmp/linux/dir"
    owner = 1000
    group = 1000
    mode = "755"
    overwrite = true
    recycle_path = "/tmp/recycle"
}

resource "linux_file" "file" {
    path = "/tmp/linux/file"
    content = <<-EOF
        hello world
    EOF
    owner = 1000
    group = 1000
    mode = "644"
    overwrite = true
    recycle_path = "/tmp/recycle"
}

resource "linux_script" "install_package" {
    lifecycle_commands {
        create = "apt update && apt install -y $PACKAGE_NAME=$PACKAGE_VERSION"
        read = "apt-cache policy $PACKAGE_NAME | grep 'Installed:' | grep -v '(none)' | awk '{ print $2 }' | xargs | tr -d '\n'"
        update = "apt update && apt install -y $PACKAGE_NAME=$PACKAGE_VERSION"
        delete = "apt remove -y $PACKAGE_NAME"
    }
    environment = {
        PACKAGE_NAME = "apache2"
        PACKAGE_VERSION = "2.4.18-2ubuntu3.4"
    }
}
```

## Argument Reference

- `user` - The user that we should use for the connection. Defaults to `root`.
- `password` - The password we should use for the connection.
- `host` - (Required) The address of the resource to connect to.
- `port` - The port to connect to. Defaults to `22`.
- `timeout` - The timeout to wait for the connection to become available. Should be provided as a string like `30s` or `5m`. Defaults to 5 minutes.
- `script_path` - The path used to copy scripts meant for remote execution.
- `private_key` - The contents of an SSH key to use for the connection. These can be loaded from a file on disk using [the file function](https://www.terraform.io/docs/configuration/functions/file.html). This takes preference over the `password` if provided.
- `certificate` - The contents of a signed CA Certificate. The certificate argument must be used in conjunction with a `private_key`. These can be loaded from a file on disk using the [the file function](https://www.terraform.io/docs/configuration/functions/file.html).
- `agent` - Set to `false` to disable using `ssh-agent` to authenticate. On Windows the only supported SSH authentication agent is [Pageant](http://the.earth.li/~sgtatham/putty/0.66/htmldoc/Chapter9.html#pageant).
- `agent_identity` - The preferred identity from the ssh agent for authentication.
- `host_key` - The public key from the remote host or the signing CA, used to verify the connection.
- `bastion_host` - Setting this enables the bastion Host connection. This host will be connected to first, and then the host connection will be made from there.
- `bastion_host_key` - The public key from the remote host or the signing CA, used to verify the host connection.
- `bastion_port` - The port to use connect to the bastion host. Defaults to the value of the `port` field.
- `bastion_user` - The user for the connection to the bastion host. Defaults to the value of the `user` field.
- `bastion_password` - The password we should use for the bastion host. Defaults to the value of the password field.
- `bastion_private_key` - The contents of an SSH key file to use for the bastion host. These can be loaded from a file on disk using the file function. Defaults to the value of the `private_key` field.
- `bastion_certificate` - The contents of a signed CA Certificate. The certificate argument must be used in conjunction with a `bastion_private_key`. These can be loaded from a file on disk using the [the file function](https://www.terraform.io/docs/configuration/functions/file.html).

## Lazy SSH Connection Setup

SSH connection are only made when Terraform enters Create|Read|Update|Delete phase of this provider's resources. Thus specifying it's arguments with value that only known after apply should be possible.
