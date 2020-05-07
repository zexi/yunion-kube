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

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	proxyapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/proxy"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

type CloudenvResourceInfo struct {
	// 云平台名称
	// example: Google
	Provider string `json:"provider,omitempty"`

	// 云平台品牌
	// example: Google
	Brand string `json:"brand,omitempty"`

	// 云环境
	// example: public
	CloudEnv string `json:"cloud_env,omitempty"`

	// Environment
	Environment string `json:"environment,omitempty"`
}

type CloudenvResourceListInput struct {
	// 列出指定云平台的资源，支持的云平台如下
	//
	// | Provider  | 开始支持版本 | 平台                                |
	// |-----------|------------|-------------------------------------|
	// | OneCloud  | 0.0        | OneCloud内置私有云，包括KVM和裸金属管理 |
	// | VMware    | 1.2        | VMware vCenter                      |
	// | OpenStack | 2.6        | OpenStack M版本以上私有云             |
	// | ZStack    | 2.10       | ZStack私有云                         |
	// | Aliyun    | 2.0        | 阿里云                               |
	// | Aws       | 2.3        | Amazon AWS                          |
	// | Azure     | 2.2        | Microsoft Azure                     |
	// | Google    | 2.13       | Google Cloud Platform               |
	// | Qcloud    | 2.3        | 腾讯云                               |
	// | Huawei    | 2.5        | 华为公有云                           |
	// | Ucloud    | 2.7        | UCLOUD                               |
	// | Ctyun     | 2.13       | 天翼云                               |
	// | S3        | 2.11       | 通用s3对象存储                        |
	// | Ceph      | 2.11       | Ceph对象存储                         |
	// | Xsky      | 2.11       | XSKY启明星辰Ceph对象存储              |
	//
	// enum: OneCloud,VMware,Aliyun,Qcloud,Azure,Aws,Huawei,OpenStack,Ucloud,ZStack,Google,Ctyun,S3,Ceph,Xsky"
	Providers []string `json:"providers"`
	// swagger:ignore
	// Deprecated
	Provider []string `json:"provider" "yunion:deprecated-by":"providers"`

	// 列出指定云平台品牌的资源，一般来说brand和provider相同，除了以上支持的provider之外，还支持以下band
	//
	// |   Brand  | Provider | 说明        |
	// |----------|----------|------------|
	// | DStack   | ZStack   | 滴滴云私有云 |
	//
	Brands []string `json:"brands"`
	// swagger:ignore
	// Deprecated
	Brand []string `json:"brand" "yunion:deprecated-by":"brands"`

	// 列出指定云环境的资源，支持云环境如下：
	//
	// | CloudEnv  | 说明   |
	// |-----------|--------|
	// | public    | 公有云  |
	// | private   | 私有云  |
	// | onpremise | 本地IDC |
	//
	// enum: public,private,onpremise
	CloudEnv string `json:"cloud_env"`

	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	PublicCloud bool `json:"public_cloud"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsPublic bool `json:"is_public"`

	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	PrivateCloud bool `json:"private_cloud"`
	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsPrivate bool `json:"is_private"`

	// swagger:ignore
	// Deprecated
	// description: this param will be deprecate at 3.0
	IsOnPremise bool `json:"is_on_premise"`

	// 以平台名称排序
	// pattern:asc|desc
	OrderByProvider string `json:"order_by_provider"`

	// 以平台品牌排序
	// pattern:asc|desc
	OrderByBrand string `json:"order_by_brand"`
}

type CloudaccountResourceInfo struct {
	CloudenvResourceInfo

	// 云账号名称
	// example: google-account
	Account string `json:"account,omitempty"`
}

type CloudaccountCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// 指定云平台
	// Qcloud: 腾讯云
	// Ctyun: 天翼云
	// enum: VMware, Aliyun, Qcloud, Azure, Aws, Huawei, OpenStack, Ucloud, ZStack, Google, Ctyun
	Provider string `json:"provider"`
	// swagger:ignore
	AccountId string

	// 指定云平台品牌, 此参数默认和provider相同
	// requried: false
	//
	//
	//
	// | provider | 支持的参数 |
	// | -------- | ---------- |
	// | VMware | VMware |
	// | Aliyun | Aliyun |
	// | Qcloud | Qcloud |
	// | Azure | Azure |
	// | Aws | Aws |
	// | Huawei | Huawei |
	// | OpenStack | OpenStack |
	// | Ucloud | Ucloud |
	// | ZStack | ZStack, DStack |
	// | Google | Google |
	// | Ctyun | Ctyun |
	Brand string `json:"brand"`

	// swagger:ignore
	IsPublicCloud bool
	// swagger:ignore
	IsOnPremise bool

	// 指定云账号所属的项目
	// Tenant string `json:"tenant"`
	// swagger:ignore
	// TenantId string

	apis.ProjectizedResourceInput

	// 启用自动同步
	// default: false
	EnableAutoSync bool `json:"enable_auto_sync"`

	// 自动同步间隔时间
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// 自动根据云上项目或订阅创建本地项目
	// default: false
	AutoCreateProject bool `json:"auto_create_project"`

	// 额外信息,例如账单的access key
	Options *jsonutils.JSONDict `json:"options"`

	proxyapi.ProxySettingResourceInput

	cloudprovider.SCloudaccount
	cloudprovider.SCloudaccountCredential
}

type CloudaccountShareModeInput struct {
	apis.Meta

	ShareMode string
}

func (i CloudaccountShareModeInput) Validate() error {
	if len(i.ShareMode) == 0 {
		return httperrors.NewMissingParameterError("share_mode")
	}
	if !utils.IsInStringArray(i.ShareMode, CLOUD_ACCOUNT_SHARE_MODES) {
		return httperrors.NewInputParameterError("invalid share_mode %s", i.ShareMode)
	}
	return nil
}

type CloudaccountListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput

	ManagedResourceListInput

	CapabilityListInput

	SyncableBaseResourceListInput

	// 账号健康状态
	HealthStatus []string `json:"health_status"`

	// 共享模式
	ShareMode []string `json:"share_mode"`
}

type ProviderProject struct {
	// 子订阅项目名称
	// example: system
	Tenant string `json:"tenant"`

	// 子订阅项目Id
	// 9a48383a-467a-4542-8b50-4e15b0a8715f
	TenantId string `json:"tenant_id"`
}

type CloudaccountDetail struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	SCloudaccount

	// 子订阅项目信息
	Projects []ProviderProject `json:"projects"`

	// 同步时间间隔
	// example: 3600
	SyncIntervalSeconds int `json:"sync_interval_seconds"`

	// 同步状态
	SyncStatus2 string `json:"sync_stauts2"`

	// 云账号环境类型
	// public: 公有云
	// private: 私有云
	// onpremise: 本地IDC
	// example: public
	CloudEnv string `json:"cloud_env"`

	// 云账号项目名称
	// example: system
	Tenant string `json:"tenant"`

	// 弹性公网Ip数量
	// example: 2
	EipCount int `json:"eip_count,allowempty"`

	// 虚拟私有网络数量
	// example: 4
	VpcCount int `json:"vpc_count,allowempty"`

	// 云盘数量
	// example: 12
	DiskCount int `json:"disk_count,allowempty"`

	// 宿主机数量(不计算虚拟机宿主机数量)
	// example: 0
	HostCount int `json:"host_count,allowempty"`

	// 云主机数量
	// example: 4
	GuestCount int `json:"guest_count,allowempty"`

	// 块存储数量
	// example: 12
	StorageCount int `json:"storage_count,allowempty"`

	// 子订阅数量
	// example: 1
	ProviderCount int `json:"provider_count,allowempty"`

	// 路由表数量
	// example: 0
	RoutetableCount int `json:"routetable_count,allowempty"`

	// 存储缓存数量
	// example: 10
	StoragecacheCount int `json:"storagecache_count,allowempty"`

	ProxySetting proxyapi.SProxySetting `json:"proxy_setting"`
}

type CloudaccountUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput

	// 同步周期，单位为秒
	SyncIntervalSeconds *int64 `json:"sync_interval_seconds"`

	// 待更新的options key/value
	Options *jsonutils.JSONDict `json:"options"`
	// 带删除的options key
	RemoveOptions []string `json:"remove_options"`

	proxyapi.ProxySettingResourceInput
}

type CloudaccountPerformPublicInput struct {
	apis.PerformPublicDomainInput

	// 共享模式，可能值为provider_domain, system
	// example: provider_domain
	ShareMode string `json:"share_mode"`
}
