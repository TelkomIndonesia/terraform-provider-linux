# linux_local_forward

Establish SSH local forwarding.

## Example Usage

```hcl
data "linux_local_forward" "remote" {
    remote_host = "127.0.0.1"
    remote_port = "3306"
    local_host = "0.0.0.0"
    local_port = "3306"
}
```

## Argument Reference

The following arguments are supported:

- `provider_override` - (Optional) see [provider_override](../#provider-override).
- `remote_host` - (Required, string) The remote host to forward the connection.
- `remote_port` - (Required, int) The remote port to forward the connection.
- `local_host` - (Optional, string) Local host address to receive the connection. Default `0.0.0.0`.
- `local_port` - (Optional, int) Local port to receive the connection. Default to `0` which will use random port.

## Attribute Reference

- `host` - (string) Local listen host. Will be equal to the value of `local_host`.
- `port` - (int) Local listen port. Will be equal to the value of `local_port` unless when it is `0` which will be a random port.
