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

	"yunion.io/x/onecloud/pkg/apis"
)

/*

Architecture For DnsZone


                                                   +-----------+                    +----------------+               +-----------+
                                                   | RecordSet |                    | TrafficPolicy  |               | RecordSet |                                         +-------------+
                                                   | (A)       |                    | (Aliyun)       |               | (TXT)     |                                         |  Vpc1       |
                                                   |           |                    |                |               |           |                                         |  (Aws)      |
                                                   |           |                    |                |               |           |                                         |             |
                  +-----------------------+        |           |                    |                |               |           |               +-----------------+       |             |
    API           |  DnsZone  example.com |        | RecordSet |                    | TrafficPolicy  |               | RecordSet |               | DnsZone abc.app |       |  Vpc2       |
                  |  (Public)             | ------>| (AAAA)    | -----------------> | (Tencent)      | <-------------| (CAA)     | <-------------| (Private)       |-----> |  (Tencent)  |
                  +-----------------------+        |           |                    |                |               |           |               +-----------------+       |             |
                          ^                        |           |                    |                |               |           |                       ^                 |             |
                          |                        |           |                    |                |               |           |                       |                 |  Vpc3       |
                          |                        | RecordSet |                    | TrafficPolicy  |               | RecordSet |                       |                 |  (Aws)      |
                          |                        | (NS)      |                    | (Aws)          |               | (PTR)     |                       |                 +-------------+
                          |                        +-----------+                    +----------------+               +-----------+                       |
                          |                                                                                                                              |
                          |                                                                                                                              |
                  ------------------------------------------------------------------------------------------------------------------------------------------------------------------------
                          |                                                                                                                              |
                          v                                                                                                                              |
                  +-----------------+                                                                                                                    |
                  |                 |                                                                                                                    |
                  |                 |            +----------+                                                                                            v
                  |  example.com <-------------> | Account1 |                                                             +----------+           +---------------+
                  |                 |            | (Aliyun) |                                                             | Account3 | <-------> |     abc.app   |
                  |                 |            +----------+                              +------------+                 | (Aws)    |           |               |
                  |                 |                                                      | Account2   |                 +----------+           |               |
                  |  example.com <-------------------------------------------------------> | (Tencent)  |                                        |               |
   Cache          |                 |                                                      +------------+                                        |               |
                  |                 |                                                                                                            |               |
                  |                 |            +----------+                                                                                    |               |
                  |  example.com <-------------> | Account4 | <--------------------------------------------------------------------------------> |     abc.app   |
                  |                 |            | (Aliyun) |                                                                                    |               |
                  |                 |            +----------+                                                                                    +---------------+
                  +-----------------+

               ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------


                                                *************                           ***************                   *************
                                            ****             ****                   ****               ****           ****             ****
 Public Cloud                                **     Aliyun    **                     **     Tencent     **             **	  Aws      **
                                            ****             ****                   ****               ****           ****             ****
                                                *************                           ***************                   *************


*/

const (
	DNS_ZONE_STATUS_AVAILABLE               = "available"               // ??????
	DNS_ZONE_STATUS_CREATING                = "creating"                // ?????????
	DNS_ZONE_STATUS_CREATE_FAILE            = "create_failed"           // ????????????
	DNS_ZONE_STATUS_UNCACHING               = "uncaching"               // ?????????????????????
	DNS_ZONE_STATUS_UNCACHE_FAILED          = "uncache_failed"          // ????????????????????????
	DNS_ZONE_STATUS_CACHING                 = "caching"                 // ?????????????????????
	DNS_ZONE_STATUS_CACHE_FAILED            = "cache_failed"            // ????????????????????????
	DNS_ZONE_STATUS_SYNC_VPCS               = "sync_vpcs"               // ??????VPC???
	DNS_ZONE_STATUS_SYNC_VPCS_FAILED        = "sync_vpcs_failed"        // ??????VPC??????
	DNS_ZONE_STATUS_SYNC_RECORD_SETS        = "sync_record_sets"        // ?????????????????????
	DNS_ZONE_STATUS_SYNC_RECORD_SETS_FAILED = "sync_record_sets_failed" // ????????????????????????
	DNS_ZONE_STATUS_DELETING                = "deleting"                // ?????????
	DNS_ZONE_STATUS_DELETE_FAILED           = "delete_failed"           // ????????????
	DNS_ZONE_STATUS_UNKNOWN                 = "unknown"                 // ??????
)

type DnsZoneFilterListBase struct {
	DnsZoneId string `json:"dns_zone_id"`
}

type DnsZoneCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput

	// ????????????
	//
	//
	// | ??????			| ??????    |
	// |----------		|---------|
	// | PublicZone		| ??????    |
	// | PrivateZone	| ??????    |
	ZoneType string `json:"zone_type"`
	// ????????????

	// VPC id??????, ??????zone_type???PrivateZone?????????, vpc?????????????????????????????????
	VpcIds []string `json:"vpc_ids"`

	// ?????????Id, ??????zone_type???PublicZone?????????, ?????????????????????????????????
	CloudaccountId string `json:"cloudaccount_id"`

	// ????????????
	Options *jsonutils.JSONDict `json:"options"`
}

type DnsZoneDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	SDnsZone

	// Dns????????????
	DnsRecordsetCount int `json:"dns_recordset_count"`
	// ??????vpc??????
	VpcCount int `json:"vpc_count"`
}

type DnsZoneListInput struct {
	apis.EnabledStatusInfrasResourceBaseListInput

	// ????????????
	//
	//
	// | ??????			| ??????    |
	// |----------		|---------|
	// | PublicZone		| ??????    |
	// | PrivateZone	| ??????    |
	ZoneType string `json:"zone_type"`

	// Filter dns zone By vpc
	VpcId string `json:"vpc_id"`
}

type DnsZoneSyncStatusInput struct {
}

type DnsZoneCacheInput struct {
	// ?????????Id
	//
	//
	// | ??????								|
	// |----------							|
	// | 1. dns zone ???????????????available		|
	// | 2. dns zone zone_type ?????????PublicZone |
	// | 3. ?????????????????????????????????????????? dns zone |
	CloudaccountId string
}

type DnsZoneUnacheInput struct {
	// ?????????Id
	CloudaccountId string
}

type DnsZoneAddVpcsInput struct {
	// VPC id??????
	VpcIds []string `json:"vpc_ids"`
}

type DnsZoneRemoveVpcsInput struct {
	// VPC id??????
	VpcIds []string `json:"vpc_ids"`
}

type DnsZoneSyncRecordSetsInput struct {
}

type DnsZonePurgeInput struct {
}
