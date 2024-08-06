# linux_directory

Manage linux directory with support for Terraform update lifecycle.

## Example Usage

```hcl
resource "linux_directory" "file" {
    path = "/tmp/linux/directory"
    owner = 1000
    group = 1000
    mode = "755"
    overwrite = true
    recycle_path = "/tmp/recycle"
}
```

## Argument Reference

The following arguments are supported:

- `provider_override` - (Optional) see [provider_override](../#provider-override).
- `path` - (Required, string) Absolute path of the directory. Parent directory will be prepared as needed. Changing this will move all contents under the current directory to the new directory.
- `owner` - (Optional, int) User ID of the folder. Default `0`.
- `group` - (Optional, int) Group ID of the folder. Default `0`.
- `mode` - (Optional, string) File mode. Default `755`.
- `overwrite` - (Optional, bool) If `true`, existing directory on remote will be replaced on Create or Update. This doesn't affect the content of the directory. Default `false`.
- `recycle_path` - (Optional, string) Absolute path to a parent directory of a generated-unix-timestamp directory where the directory will be placed on destroy. Default to empty string which will make the directory becomes deleted on destroy.

## Attribute Reference

None
