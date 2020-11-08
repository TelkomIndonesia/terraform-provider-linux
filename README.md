Terraform Provider Linux
========================

[![Build Status](https://cloud.drone.io/api/badges/TelkomIndonesia/terraform-provider-linux/status.svg?branch=master)](https://cloud.drone.io/TelkomIndonesia/terraform-provider-linux)
[![Go Report Card](https://goreportcard.com/badge/github.com/TelkomIndonesia/terraform-provider-linux)](https://goreportcard.com/report/github.com/TelkomIndonesia/terraform-provider-linux)

- Website: <https://registry.terraform.io/namespaces/TelkomIndonesia>

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.12.x
- [Go](https://golang.org/doc/install) 1.12 (to build the provider plugin)

Usage
-----

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

Developing The Provider
-----------------------

In order to build the provider run `make build`:

```sh
make build
```

In order to test the provider, you can simply run `make test`.

```sh
make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* A linux machine with ssh connection is required to run Acceptance tests. Connection information need to be specified through [Environment variables](linux/linux_test.go#L34-L48)  for the test code. This repo includes [SSH server inside docker](build/docker/docker-compose.yml) that can be used for running the tests.

```sh
make testacc
```
