# Packer Plugin Bhyve

This plugin can be used with HashiCorp [Packer](https://www.packer.io) to
create custom images.   It borrows from
[packer-plugin-qemu](https://github.com/hashicorp/packer-plugin-qemu) with the
aim of being able to reuse Packer machine templates as much as possible.

The current target platform is illumos, and while there are currently some
hardcoded dependencies on that platform (such as VNICs), the long-term goal is
that it should be cross-platform.

## Differences

Many packer-plugin-qemu variables are unsupported as they simply do not make
sense in a bhyve context.

Variables introduced for bhyve support are:

* `host_nic`: The host NIC on which we create a VNIC for the virtual machine to
  use, as well as listen on for the Packer HTTP server.
