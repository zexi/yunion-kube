// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compute

import "yunion.io/x/onecloud/pkg/apis"

type IsolateDeviceDetails struct {
	apis.StandaloneResourceDetails
	HostResourceInfo

	SIsolatedDevice

	// 云主机名称
	Guest string `json:"guest"`
	// 云主机状态
	GuestStatus string `json:"guest_status"`
}

type IsolatedDeviceListInput struct {
	apis.StandaloneResourceListInput
	apis.DomainizedResourceListInput

	HostFilterListInput

	// 只列出GPU直通设备
	Gpu *bool `json:"gpu"`
	// 只列出USB直通设备
	Usb *bool `json:"usb"`
	// 只列出未使用的直通设备
	Unused *bool `json:"unused"`

	// # PCI / GPU-HPC / GPU-VGA / USB / NIC
	// 设备类型
	DevType []string `json:"dev_type"`

	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model []string `json:"model"`

	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr []string `json:"addr"`

	// 设备VENDOE编号
	VendorDeviceId []string `json:"vendor_device_id"`
}

type IsolatedDeviceCreateInput struct {
	apis.StandaloneResourceCreateInput

	HostResourceInput

	// 设备类型USB/GPU
	// example: GPU
	DevType string `json:"dev_type"`

	// 设备型号
	// # Specific device name read from lspci command, e.g. `Tesla K40m` ...
	Model string `json:"model"`

	// PCI地址
	// # pci address of `Bus:Device.Function` format, or usb bus address of `bus.addr`
	Addr string `json:"addr"`

	// 设备VendorId
	VendorDeviceId string `json:"vendor_device_id"`
}
