---
layout: "nutanix"
page_title: "NUTANIX: nutanix_virtual_machines"
sidebar_current: "docs-nutanix-datasource-virtual-machines"
description: |-
 Describes a List of Virtual Machines
---

# nutanix_virtual_machines

Describes a List of Virtual Machines

## Example Usage

```hcl
data "nutanix_clusters" "clusters" {
  metadata = {
    length = 2
  }
}

output "cluster" {
  value = "${data.nutanix_clusters.clusters.entities.0.metadata.uuid}"
}

resource "nutanix_virtual_machine" "vm1" {
  name = "test-dou-vm1"

  cluster_reference = {
    kind = "cluster"
    uuid = "${data.nutanix_clusters.clusters.entities.0.metadata.uuid}"
  }

  num_vcpus_per_socket = 1
  num_sockets          = 1
  memory_size_mib      = 2048
  power_state          = "ON"

  nic_list = [{
    subnet_reference = {
      kind = "subnet"
      uuid = "${nutanix_subnet.test.id}"
    }

    ip_endpoint_list = {
        ip = "192.168.0.10"
        type = "ASSIGNED"
    }
  }]
}

resource "nutanix_virtual_machine" "vm2" {
  name = "test-dou-vm2"

  cluster_reference = {
	  kind = "cluster"
	  uuid = "${data.nutanix_clusters.clusters.entities.0.metadata.uuid}"
  }

  num_vcpus_per_socket = 1
  num_sockets          = 1
  memory_size_mib      = 2048
  power_state          = "ON"

  nic_list = [{
    subnet_reference = {
      kind = "subnet"
      uuid = "${nutanix_subnet.test.id}"
    }

    ip_endpoint_list = {
        ip = "192.168.0.11"
        type = "ASSIGNED"
    }
  }]
}

resource "nutanix_subnet" "test" {
  name        = "dou_vlan0_test"

  cluster_reference = {
      kind = "cluster"
      uuid = "${data.nutanix_clusters.clusters.entities.0.metadata.uuid}"
  }

  vlan_id     = 201
  subnet_type = "VLAN"

  prefix_length      = 24
  default_gateway_ip = "192.168.0.1"
  subnet_ip          = "192.168.0.0"

  dhcp_options {
    boot_file_name   = "bootfile"
    tftp_server_name = "192.168.0.252"
    domain_name      = "nutanix"
  }

  dhcp_domain_name_server_list = ["8.8.8.8", "4.2.2.2"]
  dhcp_domain_search_list      = ["nutanix.com", "calm.io"]
}

data "nutanix_virtual_machines" "basic_web" {
    metadata = {
        length = 2
    }
}
```

## Argument Reference

The following arguments are supported:

* `metadata`: Represents virtual machine uuid

### Metadata Argument

The metadata attribute supports the following:

* `kind`: - The kind name.
* `sort_attribute`: The attribute to perform sort on.
* `filter`: - The filter in FIQL syntax used for the results.
* `length`: - The number of records to retrieve relative to the offset.
* `sort_order`: - The sort order in which results are returned
* `offset`: - Offset from the start of the entity list

## Attribute Reference

The following attributes are exported:

* `entities`: - A list of virtual machines.

### Entities Attribute

The entities attribute supports the following:

* `name`: - The name for the vm.
* `cluster_reference`: - The reference to a cluster.
* `cluster_name`: - The name of the reference to the cluster.
* `categories`: - Categories for the vm.
* `project_reference`: - The reference to a project.
* `owner_reference`: - The reference to a user.
* `availability_zone_reference`: - The reference to a availability_zone.
* `api_version`
* `description`: - A description for vm.
* `num_vnuma_nodes`: - Number of vNUMA nodes. 0 means vNUMA is disabled.
* `nic_list`: - NICs attached to the VM.
* `guest_os_id`: - Guest OS Identifier. For ESX, refer to VMware documentation [link](https://www.vmware.com/support/developer/converter-sdk/conv43_apireference/vim.vm.GuestOsDescriptor.GuestOsIdentifier.html) for the list of guest OS identifiers.
* `power_state`: - The current or desired power state of the VM. (Options : ON , OFF)
* `nutanix_guest_tools`: - Information regarding Nutanix Guest Tools.
* `num_vcpus_per_socket`: - Number of vCPUs per socket.
* `num_sockets`: - Number of vCPU sockets.
* `gpu_list`: - GPUs attached to the VM.
* `parent_referece`: - Reference to an entity that the VM cloned from.
* `memory_size_mib`: - Memory size in MiB.
* `boot_device_order_list`: - Indicates the order of device types in which VM should try to boot from. If boot device order is not provided the system will decide appropriate boot device order.
* `boot_device_disk_address`: - Address of disk to boot from.
* `boot_device_mac_address`: - MAC address of nic to boot from.
* `hardware_clock_timezone`: - VM's hardware clock timezone in IANA TZDB format (America/Los_Angeles).
* `guest_customization_cloud_init`: - VM guests may be customized at boot time using one of several different methods. Currently, cloud-init w/ ConfigDriveV2 (for Linux VMs) and Sysprep (for Windows VMs) are supported. Only ONE OF sysprep or cloud_init should be provided. Note that guest customization can currently only be set during VM creation. Attempting to change it after creation will result in an error. Additional properties can be specified. For example - in the context of VM template creation if \"override_script\" is set to \"True\" then the deployer can upload their own custom script.
* `guest_customization_is_overridable`: - Flag to allow override of customization by deployer.
* `guest_customization_cloud_init_custom_key_values`: - (Optional) Generic key value pair used for custom attributes in cloud init.
* `guest_customization_sysprep`: - VM guests may be customized at boot time using one of several different methods. Currently, cloud-init w/ ConfigDriveV2 (for Linux VMs) and Sysprep (for Windows VMs) are supported. Only ONE OF sysprep or cloud_init should be provided. Note that guest customization can currently only be set during VM creation. Attempting to change it after creation will result in an error. Additional properties can be specified. For example - in the context of VM template creation if \"override_script\" is set to \"True\" then the deployer can upload their own custom script.
* `guest_customization_sysrep_custom_key_values`: - (Optional) Generic key value pair used for custom attributes in sysrep.
* `should_fail_on_script_failure`: -  Extra configs related to power state transition. Indicates whether to abort ngt shutdown/reboot if script fails.
* `enable_script_exec`: - Extra configs related to power state transition. Indicates whether to execute set script before ngt shutdown/reboot.
* `power_state_mechanism`: - Indicates the mechanism guiding the VM power state transition. Currently used for the transition to \"OFF\" state. Power state mechanism (ACPI/GUEST/HARD).
* `vga_console_enabled`: - Indicates whether VGA console should be enabled or not.
* `disk_list` Disks attached to the VM.
* `metadata`: - The vm kind metadata.
* `state`: -
* `ip_address`: -
* `host_reference`: -
* `hypervisor_type`: - The hypervisor type for the hypervisor the VM is hosted on.

### Disk List

The disk_list attribute supports the following:

* `uuid`: - The device ID which is used to uniquely identify this particular disk.
* `disk_size_bytes` - Size of the disk in Bytes.
* `disk_size_mib` - Size of the disk in MiB. Must match the size specified in 'disk_size_bytes' - rounded up to the nearest MiB - when that field is present.
* `device_properties` - 
* `data_source_reference` - Reference
* `volume_group_reference` - Reference

### Device Properties

The device_properties attribute supports the following.

* `device_type`: - 
* `disk_address`: - 

### Sysprep

The guest_customization_sysprep attribute supports the following:

* `install_type`: - Whether the guest will be freshly installed using this unattend configuration, or whether this unattend configuration will be applied to a pre-prepared image. Default is \"PREPARED\".
* `unattend_xml`: - Generic key value pair used for custom attributes.

### Cloud Init

The guest_customization_cloud_init attribute supports the following:

* `meta_data`: - The contents of the meta_data configuration for cloud-init. This can be formatted as YAML or JSON. The value must be base64 encoded.
* `user_data`: - The contents of the user_data configuration for cloud-init. This can be formatted as YAML, JSON, or could be a shell script. The value must be base64 encoded.

### Disk Address

 The boot_device_disk_address attribute supports the following:

* `device_index`: -
* `adapter_type`: -

### GPU List

The gpu_list attribute supports the following:

* `frame_buffer_size_mib`: - GPU frame buffer size in MiB.
* `vendor`: - The vendor of the GPU.
* `uuid`: - UUID of the GPU.
* `name`: - Name of the GPU resource.
* `pci_address` - GPU {segment:bus:device:function} (sbdf) address if assigned.
* `fraction` - Fraction of the physical GPU assigned.
* `mode`: - The mode of this GPU.
* `num_virtual_display_heads`: - Number of supported virtual display heads.
* `guest_driver_version`: - Last determined guest driver version.
* `device_id`: - (Computed) The device ID of the GPU.

### Nutanix Guest Tools

The nutanix_guest_tools attribute supports the following:

* `available_version`: - Version of Nutanix Guest Tools available on the cluster.
* `iso_mount_state`: - (Optioinal) Desired mount state of Nutanix Guest Tools ISO.
* `state`: - Nutanix Guest Tools is enabled or not.
* `version`: - Version of Nutanix Guest Tools installed on the VM.
* `guest_os_version`: - Version of the operating system on the VM.
* `enabled_capability_list`: - Application names that are enabled.
* `vss_snapshot_capable`: - Whether the VM is configured to take VSS snapshots through NGT.
* `is_reachable`: - Communication from VM to CVM is active or not.
* `vm_mobility_drivers_installed`: - Whether VM mobility drivers are installed in the VM.

### NIC List

The nic_list attribute supports the following:

* `nic_type`: - The type of this NIC. Defaults to NORMAL_NIC. (Options : NORMAL_NIC , DIRECT_NIC , NETWORK_FUNCTION_NIC)
 .
* `uuid`: - The NIC's UUID, which is used to uniquely identify this particular NIC. This UUID may be used to refer to the NIC outside the context of the particular VM it is attached to.

* `floating_ip`: - 

* `model`: - The model of this NIC. (Options : VIRTIO , E1000).
* `network_function_nic_type`: - The type of this Network function NIC. Defaults to INGRESS. (Options : INGRESS , EGRESS , TAP).
* `mac_address`: - The MAC address for the adapter.
* `ip_endpoint_list`: - IP endpoints for the adapter. Currently, IPv4 addresses are supported.
* `network_function_chain_reference`: - The reference to a network_function_chain.
* `subnet_reference`: - The reference to a subnet.
* `subnet_reference_name`: - The name of the subnet reference to.

### ip_endpoint_list

The following attributes are exported:

* `ip`: - Address string.
* `type`: - Address type. It can only be "ASSIGNED" in the spec. If no type is specified in the spec, the default type is set to "ASSIGNED". (Options : ASSIGNED , LEARNED)

### Metadata

The metadata attribute exports the following:

* `last_update_time`: - UTC date and time in RFC-3339 format when vm was last updated.
* `uuid`: - vm uuid.
* `creation_time`: - UTC date and time in RFC-3339 format when vm was created.
* `spec_version`: - Version number of the latest spec.
* `spec_hash`: - Hash of the spec. This will be returned from server.
* `name`: - vm name.

### Categories

The categories attribute supports the following:

* `name`: - the key name.
* `value`: - value of the key.

### Reference

The `project_reference`, `owner_reference`, `availability_zone_reference`, `cluster_reference`, `network_function_chain_reference`, `subnet_reference`, `data_source_reference`, `volume_group_reference` attributes supports the following:

* `kind`: - The kind name (Default value: project).
* `name`: - the name.
* `uuid`: - the uuid.

Note: `cluster_reference`, `subnet_reference` does not support the attribute `name`

See detailed information in [Nutanix Virtual Machine](http://developer.nutanix.com/reference/prism_central/v3/#vms).
