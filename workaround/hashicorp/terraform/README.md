# Workaround

A partial copy of github.com/hashicorp/terraform to avoid `gob: registering duplicate types for "*tfdiags.rpcFriendlyDiag": *tfdiags.rpcFriendlyDiag != *tfdiags.rpcFriendlyDiag`. See [hashicorp/terraform-plugin-sdk#268](https://github.com/hashicorp/terraform-plugin-sdk/issues/268) and [hashicorp/terraform#23725](https://github.com/hashicorp/terraform/issues/23725)
