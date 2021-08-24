# linux_file

Manage linux file with support for Terraform update lifecycle.

## Example Usage

```hcl
resource "linux_file" "file" {
    path = "/tmp/linux/file"
    content = <<-EOF
        hello world
    EOF
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
- `path` - (Required, string) Absolute path of the file. Parent directory will be prepared as needed.
- `content` - (Optional, string) Content of the file to create. Default to empty string.
- `owner` - (Optional, int) User ID of the folder. Default `0`.
- `group` - (Optional, int) Group ID of the folder. Default `0`.
- `mode` - (Optional, string) File mode. Default `644`.
- `ignore_content` - (Optional, bool) If true, `content` will be ignored and won't be included in schema diff. Default `false`.
- `overwrite` - (Optional, bool) If `true`, existing file on remote will be replaced on Create or Update. Default `false`.
- `recycle_path` - (Optional, string) Absolute path to a parent directory of a generated-unix-timestamp directory where the file will be placed on destroy. Default to empty string which will make the file becomes deleted on destroy.

## Attribute Reference

None
