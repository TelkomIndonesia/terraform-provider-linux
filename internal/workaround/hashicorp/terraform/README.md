# Workaround

A partial copy of [github.com/hashicorp/terraform](https://github.com/hashicorp/terraform) to avoid `gob: registering duplicate types for "*tfdiags.rpcFriendlyDiag": *tfdiags.rpcFriendlyDiag != *tfdiags.rpcFriendlyDiag`. See [hashicorp/terraform-plugin-sdk#268](https://github.com/hashicorp/terraform-plugin-sdk/issues/268) and [hashicorp/terraform#23725](https://github.com/hashicorp/terraform/issues/23725)

## Extension

An [extention]((communicator/ssh/communicator-ext.go)) to [github.com/hashicorp/terraform/communicator/ssh.Communicator](communicator/ssh/communicator.go#L50) is also done to add/expose additional function and methods.
