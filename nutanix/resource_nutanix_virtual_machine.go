package nutanix

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/terraform-providers/terraform-provider-nutanix/client/v3"

	"github.com/hashicorp/terraform/helper/schema"
)

// var statusCodeFilter map[int]bool
// var statusMap map[int]bool
// var version int64
// var powerON = vmconfig.PowerON
// var powerOFF = vmconfig.PowerOFF

// func init() {
// 	statusMap = map[int]bool{
// 		200: true,
// 		201: true,
// 		202: true,
// 		203: true,
// 		204: true,
// 		205: true,
// 		206: true,
// 		207: true,
// 		208: true,
// 	}
// 	statusCodeFilter = statusMap
// }

// func resourceNutanixVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
// 	client := meta.(*V3Client)
// 	APIInstance := setAPIInstance(client)
// 	VMIntentResponse, APIResponse, err := APIInstance.VmsUuidGet(d.Id())
// 	log.Printf("[DEBUG] Syncing the remote Virtual Machine instance with local instance: %s, %s", VMIntentResponse.Spec.Name, d.Id())
// 	if err != nil {
// 		return err
// 	}
// 	machine := vmconfig.SetMachineConfig(d)

// 	err = checkAPIResponse(*APIResponse)
// 	if err != nil {
// 		return err
// 	}

// 	VMIntentResponse.Spec.Resources = vmconfig.GetVMResources(VMIntentResponse.Status.Resources)

// 	machineTemp := nutanixV3.VmIntentInput{
// 		ApiVersion: "3.0",
// 		Spec:       VMIntentResponse.Spec,
// 		Metadata:   VMIntentResponse.Metadata,
// 	}

// 	if len(machineTemp.Spec.Resources.DiskList) == len(machine.Spec.Resources.DiskList) {
// 		machineTemp.Spec.Resources.DiskList = machine.Spec.Resources.DiskList
// 	}
// 	if len(machineTemp.Spec.Resources.NicList) == len(machine.Spec.Resources.NicList) {
// 		machineTemp.Spec.Resources.NicList = machine.Spec.Resources.NicList
// 	}
// 	machineTemp.Metadata.OwnerReference = machine.Metadata.OwnerReference
// 	machineTemp.Metadata.Uuid = machine.Metadata.Uuid
// 	machineTemp.Metadata.Name = machine.Metadata.Name

// 	if !reflect.DeepEqual(machineTemp, machine) {
// 		err = vmconfig.UpdateTerraformState(d, VMIntentResponse.Metadata, VMIntentResponse.Spec)
// 		if err != nil {
// 			return err
// 		}
// 		d.Set("ip_address", "")
// 		if len(VMIntentResponse.Spec.Resources.NicList) > 0 && VMIntentResponse.Spec.Resources.PowerState == powerON {
// 			err = client.WaitForIP(d.Id(), d)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 		version = VMIntentResponse.Metadata.SpecVersion

// 	}

// 	return nil
// }

// func resourceNutanixVirtualMachineDelete(d *schema.ResourceData, m interface{}) error {
// 	client := m.(*V3Client)
// 	log.Printf("[DEBUG] Deleting Virtual Machine: %s", d.Id())
// 	APIInstance := setAPIInstance(client)
// 	uuid := d.Id()

// 	APIResponse, err := APIInstance.VmsUuidDelete(uuid)
// 	if err != nil {
// 		return err
// 	}
// 	err = checkAPIResponse(*APIResponse)
// 	if err != nil {
// 		return err
// 	}

// 	d.SetId("")
// 	return nil
// }

// // MachineExists function returns the uuid of the machine with given name
// func resourceNutanixVirtualMachineExists(d *schema.ResourceData, m interface{}) (bool, error) {
// 	log.Printf("[DEBUG] Checking Virtual Machine Existance: %s", d.Id())
// 	client := m.(*V3Client)
// 	APIInstance := setAPIInstance(client)

// 	getEntitiesRequest := nutanixV3.VmListMetadata{} // VmListMetadata
// 	VMListIntentResponse, APIResponse, err := APIInstance.VmsListPost(getEntitiesRequest)
// 	if err != nil {
// 		return false, err
// 	}
// 	err = checkAPIResponse(*APIResponse)
// 	if err != nil {
// 		return false, err
// 	}

// 	for i := range VMListIntentResponse.Entities {
// 		if VMListIntentResponse.Entities[i].Metadata.Uuid == d.Id() {
// 			return true, nil
// 		}
// 	}
// 	return false, nil
// }
// }

func resourceNutanixVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	// Get client connection
	conn := meta.(*NutanixClient).API

	var version string
	if v, ok := d.GetOk("api_version"); ok {
		version = v.(string)
	} else {
		version = Version
	}

	// Prepare request
	request := v3.VMIntentInput{
		APIVersion: version,
	}

	// Read Arguments
	spec := d.Get("spec").(map[string]interface{})
	metadata := d.Get("metadata").(map[string]interface{})

	vmMetadata, err := setVMMetadata(metadata)
	if err != nil {
		return fmt.Errorf("error reading metadata info %s", err)
	}

	vmSpec, err := setVMSpec(spec)
	if err != nil {
		return fmt.Errorf("error reading spec info %s", err)
	}

	request.Spec = vmSpec
	request.Metadata = vmMetadata

	// Make request to the API
	resp, err := conn.V3.CreateVM(request)
	if err != nil {
		return err
	}

	uuid := resp.Metadata.UUID

	// Wait for the VM to be available
	status, err := waitForVMProcess(conn, uuid)
	for status != true {
		return err
	}

	// Set terraform state id
	d.SetId(uuid)
	d.Partial(true)
	d.Set("ip_address", "")
	d.Partial(false)

	// Read the ip
	if resp.Spec.Resources.NicList != nil && resp.Spec.Resources.PowerState == "ON" {
		log.Printf("[DEBUG] Polling for IP\n")
		err := waitForIP(conn, uuid, d)
		if err != nil {
			return err
		}
	}

	return resourceNutanixVirtualMachineRead(d, meta)
}

func resourceNutanixVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	// Get client connection
	conn := meta.(*NutanixClient).API

	// Make request to the API
	resp, err := conn.V3.GetVM(d.Id())
	if err != nil {
		return err
	}

	// Set vm values
	status := make(map[string]interface{})

	// Simple first
	status["name"] = resp.Status.Name
	status["state"] = resp.Status.State
	status["description"] = resp.Status.Description

	// Complext after
	availabilityZoneReference := make(map[string]interface{})
	availabilityZoneReference["kind"] = resp.Status.AvailabilityZoneReference.Kind
	availabilityZoneReference["name"] = resp.Status.AvailabilityZoneReference.Name
	availabilityZoneReference["uuid"] = resp.Status.AvailabilityZoneReference.UUID

	status["availability_zone_reference"] = availabilityZoneReference

	messages := make([]map[string]interface{}, len(resp.Status.MessageList))

	for k, v := range resp.Status.MessageList {
		message := make(map[string]interface{})

		message["message"] = v.Message
		message["reason"] = v.Reason
		message["details"] = v.Details

		messages[k] = message
	}

	status["message_list"] = messages

	clusterReference := make(map[string]interface{})
	clusterReference["kind"] = resp.Status.ClusterReference.Kind
	clusterReference["name"] = resp.Status.ClusterReference.Name
	clusterReference["uuid"] = resp.Status.ClusterReference.UUID

	status["cluster_reference"] = clusterReference

	resouces := make(map[string]interface{})

	vnumaConfig := make(map[string]interface{})
	vnumaConfig["num_vnuma_nodes"] = resp.Status.Resources.VnumaConfig.NumVnumaNodes

	resouces["vnuma_config"] = vnumaConfig

	nics := resp.Status.Resources.NicList

	nicLists := make([]map[string]interface{}, len(nics))
	for k, v := range nics {
		nic := make(map[string]interface{})
		// simple firts
		nic["nic_type"] = v.NicType
		nic["uuid"] = v.UUID
		nic["floating_ip"] = v.FloatingIP
		nic["network_function_nic_type"] = v.NetworkFunctionNicType
		nic["mac_address"] = v.MacAddress
		nic["model"] = v.Model

		ipEndpointList := make([]map[string]interface{}, len(v.IPEndpointList))
		for k1, v1 := range v.IPEndpointList {
			ipEndpoint := make(map[string]interface{})
			ipEndpoint["ip"] = v1.IP
			ipEndpoint["type"] = v1.Type
			ipEndpointList[k1] = ipEndpoint
		}
		nic["ip_endpoint_list"] = ipEndpointList

		netFnChainRef := make(map[string]interface{})
		netFnChainRef["kind"] = v.NetworkFunctionChainReference.Kind
		netFnChainRef["name"] = v.NetworkFunctionChainReference.Name
		netFnChainRef["uuid"] = v.NetworkFunctionChainReference.UUID

		nic["network_function_chain_reference"] = netFnChainRef

		subtnetRef := make(map[string]interface{})
		subtnetRef["kind"] = v.SubnetReference.Kind
		subtnetRef["name"] = v.SubnetReference.Name
		subtnetRef["uuid"] = v.SubnetReference.UUID

		nic["subnet_reference"] = subtnetRef

		nicLists[k] = nic
	}

	resouces["nic_list"] = nicLists
	hostRef := make(map[string]interface{})
	hostRef["kind"] = resp.Status.Resources.HostReference.Kind
	hostRef["name"] = resp.Status.Resources.HostReference.Name
	hostRef["uuid"] = resp.Status.Resources.HostReference.UUID

	resouces["host_reference"] = hostRef

	guestTools := make(map[string]interface{})

	tools := resp.Status.Resources.GuestTools.NutanixGuestTools
	nutanixGuestTools := make(map[string]interface{})
	nutanixGuestTools["available_version"] = tools.AvailableVersion
	nutanixGuestTools["iso_mount_state"] = tools.IsoMountState
	nutanixGuestTools["state"] = tools.State
	nutanixGuestTools["version"] = tools.Version
	nutanixGuestTools["guest_os_version"] = tools.GuestOsVersion

	capList := make([]string, len(tools.EnabledCapabilityList))
	for k, v := range tools.EnabledCapabilityList {
		capList[k] = v
	}
	nutanixGuestTools["enabled_capability_list"] = capList
	nutanixGuestTools["vss_snapshot_capable"] = tools.VSSSnapshotCapable
	nutanixGuestTools["is_reachable"] = tools.IsReachable
	nutanixGuestTools["vm_mobility_drivers_installed"] = tools.VMMobilityDriversInstalled

	guestTools["nutanix_guest_tools"] = nutanixGuestTools

	resouces["guest_tools"] = guestTools

	gpuList := make([]map[string]interface{}, len(resp.Status.Resources.GpuList))
	for k, v := range resp.Status.Resources.GpuList {
		gpu := make(map[string]interface{})
		gpu["frame_buffer_size_mib"] = v.FrameBufferSizeMib
		gpu["vendor"] = v.Vendor
		gpu["uuid"] = v.UUID
		gpu["name"] = v.Name
		gpu["pci_address"] = v.PCIAddress
		gpu["fraction"] = v.Fraction
		gpu["mode"] = v.Mode
		gpu["num_virtual_display_heads"] = v.NumVirtualDisplayHeads
		gpu["guest_driver_version"] = v.GuestDriverVersion
		gpu["device_id"] = v.DeviceID

		gpuList[k] = gpu
	}

	resouces["gpu_list"] = gpuList

	parentRef := make(map[string]interface{})
	parentRef["kind"] = resp.Status.Resources.ParentReference.Kind
	parentRef["name"] = resp.Status.Resources.ParentReference.Name
	parentRef["uuid"] = resp.Status.Resources.ParentReference.UUID

	resouces["parent_reference"] = parentRef

	bootConfig := make(map[string]interface{})
	boots := make([]string, len(resp.Status.Resources.BootConfig.BootDeviceOrderList))
	for k, v := range resp.Status.Resources.BootConfig.BootDeviceOrderList {
		boots[k] = v
	}
	bootDevice := make(map[string]interface{})
	diskAddress := make(map[string]interface{})
	diskAddress["device_index"] = resp.Status.Resources.BootConfig.BootDevice.DiskAddress.DeviceIndex
	diskAddress["adapter_type"] = resp.Status.Resources.BootConfig.BootDevice.DiskAddress.AdapterType
	bootDevice["disk_address"] = diskAddress
	bootDevice["mac_address"] = resp.Status.Resources.BootConfig.BootDevice.MacAddress

	bootConfig["boot_device"] = bootDevice
	bootConfig["boot_device_order_list"] = boots

	resouces["boot_config"] = bootConfig

	guestCustom := make(map[string]interface{})
	cloudInit := make(map[string]interface{})
	cloudInit["meta_data"] = resp.Status.Resources.GuestCustomization.CloudInit.MetaData
	cloudInit["user_data"] = resp.Status.Resources.GuestCustomization.CloudInit.UserData
	cloudInit["custom_key_values"] = resp.Status.Resources.GuestCustomization.CloudInit.CustomKeyValues

	guestCustom["cloud_init"] = cloudInit
	guestCustom["is_overridable"] = resp.Status.Resources.GuestCustomization.IsOverridable

	sysprep := make(map[string]interface{})
	sysprep["install_type"] = resp.Status.Resources.GuestCustomization.Sysprep.InstallType
	sysprep["unattend_xml"] = resp.Status.Resources.GuestCustomization.Sysprep.UnattendXML
	sysprep["custom_key_values"] = resp.Status.Resources.GuestCustomization.Sysprep.CustomKeyValues

	guestCustom["sysprep"] = sysprep

	resouces["guest_customization"] = guestCustom

	powerStateMechanism := make(map[string]interface{})
	powerStateMechanism["mechanism"] = resp.Status.Resources.PowerStateMechanism.Mechanism

	guestTransition := make(map[string]interface{})
	guestTransition["should_fail_on_script_failure"] = resp.Status.Resources.PowerStateMechanism.GuestTransitionConfig.ShouldFailOnScriptFailure
	guestTransition["enable_script_exec"] = resp.Status.Resources.PowerStateMechanism.GuestTransitionConfig.EnableScriptExec

	powerStateMechanism["guest_transition_config"] = guestTransition

	resouces["power_state_mechanism"] = powerStateMechanism

	diskList := make([]map[string]interface{}, len(resp.Status.Resources.DiskList))
	for k, v := range resp.Status.Resources.DiskList {
		disk := make(map[string]interface{})
		disk["uuid"] = v.UUID
		disk["disk_size_bytes"] = v.DiskSizeBytes
		disk["disk_size_mib"] = v.DiskSizeMib

		dsourceRef := make(map[string]interface{})
		dsourceRef["kind"] = v.DataSourceReference.Kind
		dsourceRef["name"] = v.DataSourceReference.Name
		dsourceRef["uuid"] = v.DataSourceReference.UUID

		disk["data_source_reference"] = dsourceRef

		volumeRef := make(map[string]interface{})
		volumeRef["kind"] = v.VolumeGroupReference.Kind
		volumeRef["name"] = v.VolumeGroupReference.Name
		volumeRef["uuid"] = v.VolumeGroupReference.UUID

		disk["volume_group_reference"] = volumeRef

		deviceProps := make(map[string]interface{})
		deviceProps["device_type"] = v.DeviceProperties.DeviceType

		diskAddress := make(map[string]interface{})
		diskAddress["device_index"] = v.DeviceProperties.DiskAddress.DeviceIndex
		diskAddress["adapter_type"] = v.DeviceProperties.DiskAddress.AdapterType

		deviceProps["disk_address"] = diskAddress

		disk["device_properties"] = deviceProps

		diskList[k] = disk
	}

	resouces["disk_list"] = diskList

	status["resources"] = resouces

	metadata := make(map[string]interface{})
	metadata["last_update_time"] = resp.Metadata.LastUpdateTime
	metadata["kind"] = resp.Metadata.Kind
	metadata["uuid"] = resp.Metadata.UUID
	metadata["creation_time"] = resp.Metadata.CreationTime
	metadata["spec_version"] = resp.Metadata.SpecVersion
	metadata["spec_hash"] = resp.Metadata.SpecHash
	metadata["categories"] = resp.Metadata.Categories
	metadata["name"] = resp.Metadata.Name

	pr := make(map[string]interface{})
	pr["kind"] = resp.Metadata.ProjectReference.Kind
	pr["name"] = resp.Metadata.ProjectReference.Name
	pr["uuid"] = resp.Metadata.ProjectReference.UUID

	or := make(map[string]interface{})
	or["kind"] = resp.Metadata.OwnerReference.Kind
	or["name"] = resp.Metadata.OwnerReference.Name
	or["uuid"] = resp.Metadata.OwnerReference.UUID

	metadata["project_reference"] = pr
	metadata["owner_reference"] = or

	d.Set("api_version", resp.APIVersion)
	d.Set("status", status)
	d.Set("metadata", metadata)
	d.SetId(resource.UniqueId())

	return nil
}

// func resourceNutanixVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
// 	// Enable partial state mode
// 	d.Partial(true)
// 	client := meta.(*V3Client)
// 	machine := vmconfig.SetMachineConfig(d)
// 	machine.Metadata.SpecVersion = version

// 	APIInstance := setAPIInstance(client)
// 	uuid := d.Id()
// 	log.Printf("[DEBUG] Updating Virtual Machine: %s, %s", machine.Spec.Name, d.Id())

// 	if d.HasChange("name") || d.HasChange("spec") || d.HasChange("metadata") {
// 		_, APIResponse, err := APIInstance.VmsUuidPut(uuid, machine)
// 		if err != nil {
// 			return err
// 		}
// 		err = checkAPIResponse(*APIResponse)
// 		if err != nil {
// 			return err
// 		}
// 		d.SetPartial("spec")
// 		d.SetPartial("metadata")
// 	}
// 	//Disabling partial state mode. This will cause terraform to save all fields again
// 	d.Partial(false)
// 	status, err := client.WaitForProcess(uuid)
// 	if status != true {
// 		return err
// 	}
// 	d.Set("ip_address", "")
// 	if len(machine.Spec.Resources.NicList) > 0 && machine.Spec.Resources.PowerState == powerON {
// 		log.Printf("[DEBUG] Polling for IP\n")
// 		err := client.WaitForIP(uuid, d)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func setVMMetadata(m map[string]interface{}) (v3.VMMetadata, error) {

	metadata := v3.VMMetadata{}

	if v, ok := m["kind"]; ok {
		metadata.Kind = v.(string)
	}
	if v, ok := m["name"]; ok {
		metadata.Name = v.(string)
	}
	if v, ok := m["categories"]; ok {
		metadata.Categories = v.(map[string]string)
	}

	return metadata, nil
}

func setVMSpec(m map[string]interface{}) (v3.VM, error) {

	vm := v3.VM{
		Name: m["name"].(string),
	}

	if v, ok := m["description"]; ok {
		vm.Description = v.(string)
	}

	if v, ok := m["availability_zone_reference"]; ok {
		azr := v.(map[string]string)
		vm.AvailabilityZoneReference.Kind = azr["kind"]
		vm.AvailabilityZoneReference.UUID = azr["uuid"]
		if j, ok1 := azr["name"]; ok1 {
			vm.AvailabilityZoneReference.Name = j
		}
	}

	if v, ok := m["cluster_reference"]; ok {
		clr := v.(map[string]string)
		vm.ClusterReference.Kind = clr["kind"]
		vm.ClusterReference.UUID = clr["uuid"]
		if j, ok1 := clr["name"]; ok1 {
			vm.ClusterReference.Name = j
		}
	}

	resources := m["resources"].(map[string]interface{})

	if v, ok := resources["vnuma_config"]; ok {
		vm.Resources.VMVnumaConfig.NumVnumaNodes = int64(v.(int))
	}

	if v, ok := resources["nic_list"]; ok {
		var nics []v3.VMNic

		for _, val := range v.([]map[string]interface{}) {
			nic := v3.VMNic{}

			if value, ok := val["nic_type"]; ok {
				nic.NicType = value.(string)
			}
			if value, ok := val["uuid"]; ok {
				nic.UUID = value.(string)
			}
			if value, ok := val["network_function_nic_type"]; ok {
				nic.NetworkFunctionNicType = value.(string)
			}
			if value, ok := val["mac_address"]; ok {
				nic.MacAddress = value.(string)
			}
			if value, ok := val["model"]; ok {
				nic.Model = value.(string)
			}
			if value, ok := val["ip_endpoint_list"]; ok {
				var ip []v3.IPAddress
				for _, v := range value.([]map[string]interface{}) {
					ip = append(ip, v3.IPAddress{IP: v["ip"].(string), Type: v["type"].(string)})
				}
				nic.IPEndpointList = ip
			}
			if value, ok := val["network_function_chain_reference"]; ok {
				v := value.(map[string]string)
				nic.NetworkFunctionChainReference.Kind = v["kind"]
				nic.NetworkFunctionChainReference.UUID = v["uuid"]
				if j, ok1 := v["name"]; ok1 {
					nic.NetworkFunctionChainReference.Name = j
				}
			}
			if value, ok := val["subnet_reference"]; ok {
				v := value.(map[string]string)
				nic.SubnetReference.Kind = v["kind"]
				nic.SubnetReference.UUID = v["uuid"]
				if j, ok1 := v["name"]; ok1 {
					nic.SubnetReference.Name = j
				}
			}

			nics = append(nics, nic)
		}

		vm.Resources.NicList = nics
	}
	if v, ok := resources["guest_tools"]; ok {
		ngt := v.(map[string]interface{})
		if k, ok1 := ngt["nutanix_guest_tools"]; ok1 {
			ngts := k.(map[string]interface{})
			if val, ok2 := ngts["iso_mount_state"]; ok2 {
				vm.Resources.GuestTools.NutanixGuestTools.IsoMountState = val.(string)
			}
			if val, ok2 := ngts["state"]; ok2 {
				vm.Resources.GuestTools.NutanixGuestTools.State = val.(string)
			}
			if val, ok2 := ngts["enabled_capability_list"]; ok2 {
				var l []string
				for _, list := range val.([]interface{}) {
					l = append(l, list.(string))
				}
				vm.Resources.GuestTools.NutanixGuestTools.EnabledCapabilityList = l
			}
		}
	}
	if v, ok := resources["gpu_list"]; ok {
		var gpl []v3.VMGpu
		for _, val := range v.([]map[string]interface{}) {
			gpu := v3.VMGpu{}
			if value, ok1 := val["vendor"]; ok1 {
				gpu.Vendor = value.(string)
			}
			if value, ok1 := val["device_id"]; ok1 {
				gpu.DeviceID = int64(value.(int))
			}
			if value, ok1 := val["mode"]; ok1 {
				gpu.Mode = value.(string)
			}
			gpl = append(gpl, gpu)
		}
		vm.Resources.GpuList = gpl
	}
	if v, ok := resources["parent_reference"]; ok {
		val := v.(map[string]string)
		vm.Resources.ParentReference.Kind = val["kind"]
		vm.Resources.ParentReference.UUID = val["uuid"]
		if j, ok1 := val["name"]; ok1 {
			vm.Resources.ParentReference.Name = j
		}
	}
	if v, ok := resources["boot_config"]; ok {
		val := v.(map[string]interface{})
		if value1, ok1 := val["boot_device_order_list"]; ok1 {
			var b []string
			for _, boot := range value1.([]interface{}) {
				b = append(b, boot.(string))
			}
			vm.Resources.BootConfig.BootDeviceOrderList = b
		}
		if value1, ok1 := val["boot_device"]; ok1 {
			bdi := value1.(map[string]interface{})
			bd := v3.VMBootDevice{}
			if value2, ok2 := bdi["disk_address"]; ok2 {
				dai := value2.(map[string]interface{})
				da := v3.DiskAddress{}
				if value3, ok3 := dai["device_index"]; ok3 {
					da.DeviceIndex = int64(value3.(int))
				}
				if value3, ok3 := dai["adapter_type"]; ok3 {
					da.AdapterType = value3.(string)
				}
				bd.DiskAddress = da
			}
			if value2, ok2 := bdi["mac_address"]; ok2 {
				bd.MacAddress = value2.(string)
			}
			vm.Resources.BootConfig.BootDevice = bd
		}
	}

	if v, ok := resources["guest_customization"]; ok {
		gci := v.(map[string]interface{})
		gc := v3.GuestCustomization{}

		if v1, ok1 := gci["cloud_init"]; ok1 {
			cii := v1.(map[string]interface{})
			if v2, ok2 := cii["meta_data"]; ok2 {
				gc.CloudInit.MetaData = v2.(string)
			}
			if v2, ok2 := cii["user_data"]; ok2 {
				gc.CloudInit.UserData = v2.(string)
			}
			if v2, ok2 := cii["custom_key_values"]; ok2 {
				gc.CloudInit.CustomKeyValues = v2.(map[string]string)
			}
		}
		if v1, ok1 := gci["sysprep"]; ok1 {
			spi := v1.(map[string]interface{})
			if v2, ok2 := spi["install_type"]; ok2 {
				gc.Sysprep["install_type"] = v2.(string)
			}
			if v2, ok2 := spi["unattend_xml"]; ok2 {
				gc.Sysprep["unattend_xml"] = v2.(string)
			}
			if v2, ok2 := spi["custom_key_values"]; ok2 {
				gc.Sysprep["custom_key_values"] = v2.(map[string]string)
			}
		}
		if v1, ok1 := gci["is_overridable"]; ok1 {
			gc.IsOverridable = v1.(bool)
		}

		vm.Resources.GuestCustomization = gc
	}

	return vm, nil
}

func waitForVMProcess(conn *v3.Client, uuid string) (bool, error) {
	for {
		resp, err := conn.V3.GetVM(uuid)
		if err != nil {
			return false, err
		}

		if resp.Status.State == "COMPLETE" {
			return true, nil
		} else if resp.Status.State == "ERROR" {
			return false, fmt.Errorf("Error while waiting for resource to be up")
		}
		time.Sleep(3000 * time.Millisecond)
	}
	return false, nil
}

func waitForIP(conn *v3.Client, uuid string, d *schema.ResourceData) error {
	for {
		resp, err := conn.V3.GetVM(uuid)
		if err != nil {
			return err
		}

		if len(resp.Status.Resources.NicList) != 0 {
			for i := range resp.Status.Resources.NicList {
				if len(resp.Status.Resources.NicList[i].IPEndpointList) != 0 {
					if ip := resp.Status.Resources.NicList[i].IPEndpointList[0].IP; ip != "" {
						// TODO set ip address
						d.Set("ip_address", ip)
						return nil
					}
				}
			}
		}
		time.Sleep(3000 * time.Millisecond)
	}
	return nil
}

func resourceNutanixVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceNutanixVirtualMachineCreate,
		Read:   resourceNutanixVirtualMachineRead,
		// Update: resourceNutanixVirtualMachineUpdate,
		// Delete: resourceNutanixVirtualMachineDelete,
		// Exists: resourceNutanixVirtualMachineExists,

		Schema: map[string]*schema.Schema{
			"status": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"state": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"availability_zone_reference": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"kind": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"uuid": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"message_list": &schema.Schema{
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"message": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"reason": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"details": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
									},
								},
							},
						},
						"cluster_reference": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"kind": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"uuid": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"resources": &schema.Schema{
							Type:     schema.TypeMap,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"vnuma_config": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"num_vnuma_nodes": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									"nic_list": &schema.Schema{
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nic_type": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"ip_endpoint_list": &schema.Schema{
													Type:     schema.TypeList,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"ip": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"type": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
												"network_function_chain_reference": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"kind": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"name": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
												"floating_ip": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"network_function_nic_type": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"mac_address": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"subnet_reference": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"kind": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"name": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
												"model": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"host_reference": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"kind": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"guest_os_id": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"power_state": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"guest_tools": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"nutanix_guest_tools": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"available_version": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"iso_mount_state": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"state": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"version": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"guest_os_version": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"enabled_capability_list": &schema.Schema{
																Type:     schema.TypeList,
																Computed: true,
																Elem:     &schema.Schema{Type: schema.TypeString},
															},
															"vss_snapshot_capable": &schema.Schema{
																Type:     schema.TypeBool,
																Computed: true,
															},
															"is_reachable": &schema.Schema{
																Type:     schema.TypeBool,
																Computed: true,
															},
															"vm_mobility_drivers_installed": &schema.Schema{
																Type:     schema.TypeBool,
																Computed: true,
															},
														},
													},
												},
											},
										},
									},
									"hypervisor_type": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"num_vcpus_per_socket": &schema.Schema{
										Type:     schema.TypeInt,
										Computed: true,
									},
									"num_sockets": &schema.Schema{
										Type:     schema.TypeInt,
										Computed: true,
									},
									"gpu_list": &schema.Schema{
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"frame_buffer_size_mib": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
												"vendor": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"pci_address": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"fraction": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
												"mode": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"num_virtual_display_heads": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
												"guest_driver_version": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"device_id": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									"parent_reference": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"kind": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"memory_size_mib": &schema.Schema{
										Type:     schema.TypeInt,
										Computed: true,
									},
									"boot_config": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"boot_device_order_list": &schema.Schema{
													Type:     schema.TypeList,
													Computed: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
												},
												"boot_device": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"disk_address": &schema.Schema{
																Type:     schema.TypeMap,
																Computed: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"device_index": &schema.Schema{
																			Type:     schema.TypeInt,
																			Computed: true,
																		},
																		"adapter_type": &schema.Schema{
																			Type:     schema.TypeString,
																			Computed: true,
																		},
																	},
																},
															},
															"mac_address": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
											},
										},
									},
									"hardware_clock_timezone": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
									},
									"guest_customization": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"cloud_init": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"meta_data": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"user_data": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"custom_key_values": &schema.Schema{
																Type:     schema.TypeMap,
																Computed: true,
															},
														},
													},
												},
												"is_overridable": &schema.Schema{
													Type:     schema.TypeBool,
													Computed: true,
												},
												"sysprep": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"install_type": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"unattend_xml": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"custom_key_values": &schema.Schema{
																Type:     schema.TypeMap,
																Computed: true,
															},
														},
													},
												},
											},
										},
									},
									"power_state_mechanism": &schema.Schema{
										Type:     schema.TypeMap,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"guest_transition_config": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"should_fail_on_script_failure": &schema.Schema{
																Type:     schema.TypeBool,
																Computed: true,
															},
															"enable_script_exec": &schema.Schema{
																Type:     schema.TypeBool,
																Computed: true,
															},
														},
													},
												},
												"mechanism": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
									},
									"vga_console_enabled": &schema.Schema{
										Type:     schema.TypeBool,
										Computed: true,
									},
									"disk_list": &schema.Schema{
										Type:     schema.TypeList,
										Computed: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Computed: true,
												},
												"disk_size_bytes": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
												"device_properties": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"device_type": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"disk_address": &schema.Schema{
																Type:     schema.TypeMap,
																Computed: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"device_index": &schema.Schema{
																			Type:     schema.TypeInt,
																			Computed: true,
																		},
																		"adapter_type": &schema.Schema{
																			Type:     schema.TypeString,
																			Computed: true,
																		},
																	},
																},
															},
														},
													},
												},
												"data_source_reference": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"kind": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"name": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
												"disk_size_mib": &schema.Schema{
													Type:     schema.TypeInt,
													Computed: true,
												},
												"volume_group_reference": &schema.Schema{
													Type:     schema.TypeMap,
													Computed: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"kind": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"name": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Computed: true,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"spec": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vm": &schema.Schema{
							Type:     schema.TypeMap,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"availability_zone_reference": &schema.Schema{
										Type:     schema.TypeMap,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"kind": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
									"description": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"resources": &schema.Schema{
										Type:     schema.TypeMap,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"vnuma_config": &schema.Schema{
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"num_vnuma_nodes": &schema.Schema{
																Type:     schema.TypeInt,
																Optional: true,
															},
														},
													},
												},
												"nic_list": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"nic_type": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"ip_endpoint_list": &schema.Schema{
																Type:     schema.TypeList,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"ip": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"type": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																	},
																},
															},
															"network_function_chain_reference": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"kind": &schema.Schema{
																			Type:     schema.TypeString,
																			Required: true,
																		},
																		"name": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"uuid": &schema.Schema{
																			Type:     schema.TypeString,
																			Required: true,
																		},
																	},
																},
															},
															"network_function_nic_type": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"mac_address": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"subnet_reference": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"kind": &schema.Schema{
																			Type:     schema.TypeString,
																			Required: true,
																		},
																		"name": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"uuid": &schema.Schema{
																			Type:     schema.TypeString,
																			Required: true,
																		},
																	},
																},
															},
															"model": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
												"guest_os_id": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"power_state": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"guest_tools": &schema.Schema{
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"nutanix_guest_tools": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"iso_mount_state": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"state": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"enabled_capability_list": &schema.Schema{
																			Type:     schema.TypeList,
																			Optional: true,
																			Elem:     &schema.Schema{Type: schema.TypeString},
																		},
																	},
																},
															},
														},
													},
												},
												"num_vcpus_per_socket": &schema.Schema{
													Type:     schema.TypeInt,
													Optional: true,
												},
												"num_sockets": &schema.Schema{
													Type:     schema.TypeInt,
													Optional: true,
												},
												"gpu_list": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"vendor": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"mode": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"device_id": &schema.Schema{
																Type:     schema.TypeInt,
																Optional: true,
															},
														},
													},
												},
												"parent_reference": &schema.Schema{
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"kind": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"name": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
												"memory_size_mib": &schema.Schema{
													Type:     schema.TypeInt,
													Optional: true,
												},
												"boot_config": &schema.Schema{
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"boot_device_order_list": &schema.Schema{
																Type:     schema.TypeList,
																Optional: true,
																Elem:     &schema.Schema{Type: schema.TypeString},
															},
															"boot_device": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"disk_address": &schema.Schema{
																			Type:     schema.TypeMap,
																			Optional: true,
																			Elem: &schema.Resource{
																				Schema: map[string]*schema.Schema{
																					"device_index": &schema.Schema{
																						Type:     schema.TypeInt,
																						Optional: true,
																					},
																					"adapter_type": &schema.Schema{
																						Type:     schema.TypeString,
																						Optional: true,
																					},
																				},
																			},
																		},
																		"mac_address": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																	},
																},
															},
														},
													},
												},
												"hardware_clock_timezone": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"guest_customization": &schema.Schema{
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"cloud_init": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"meta_data": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"user_data": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"custom_key_values": &schema.Schema{
																			Type:     schema.TypeMap,
																			Optional: true,
																		},
																	},
																},
															},
															"is_overridable": &schema.Schema{
																Type:     schema.TypeBool,
																Optional: true,
															},
															"sysprep": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"install_type": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"unattend_xml": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"custom_key_values": &schema.Schema{
																			Type:     schema.TypeMap,
																			Optional: true,
																		},
																	},
																},
															},
														},
													},
												},
												"power_state_mechanism": &schema.Schema{
													Type:     schema.TypeMap,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"guest_transition_config": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"should_fail_on_script_failure": &schema.Schema{
																			Type:     schema.TypeBool,
																			Optional: true,
																		},
																		"enable_script_exec": &schema.Schema{
																			Type:     schema.TypeBool,
																			Optional: true,
																		},
																	},
																},
															},
															"mechanism": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
														},
													},
												},
												"vga_console_enabled": &schema.Schema{
													Type:     schema.TypeBool,
													Optional: true,
												},
												"disk_list": &schema.Schema{
													Type:     schema.TypeList,
													Optional: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"uuid": &schema.Schema{
																Type:     schema.TypeString,
																Optional: true,
															},
															"disk_size_bytes": &schema.Schema{
																Type:     schema.TypeInt,
																Optional: true,
															},
															"device_properties": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"device_type": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"disk_address": &schema.Schema{
																			Type:     schema.TypeMap,
																			Optional: true,
																			Elem: &schema.Resource{
																				Schema: map[string]*schema.Schema{
																					"device_index": &schema.Schema{
																						Type:     schema.TypeInt,
																						Optional: true,
																					},
																					"adapter_type": &schema.Schema{
																						Type:     schema.TypeString,
																						Optional: true,
																					},
																				},
																			},
																		},
																	},
																},
															},
															"data_source_reference": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"kind": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"uuid": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"name": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																	},
																},
															},
															"disk_size_mib": &schema.Schema{
																Type:     schema.TypeInt,
																Optional: true,
															},
															"volume_group_reference": &schema.Schema{
																Type:     schema.TypeMap,
																Optional: true,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"kind": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"uuid": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																		"name": &schema.Schema{
																			Type:     schema.TypeString,
																			Optional: true,
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
									"cluster_reference": &schema.Schema{
										Type:     schema.TypeMap,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"kind": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"uuid": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"api_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"metadata": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"last_update_time": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"kind": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"uuid": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"project_reference": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"kind": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"uuid": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"creation_time": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"spec_version": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"spec_hash": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"owner_reference": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"kind": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"uuid": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"categories": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}
