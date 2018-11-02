package sync

import (
	"fmt"
	"path/filepath"

	api "k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/models"
)

const (
	YUNION_SERVICE_TYPE = "service.yunion.io/type"
	YUNION_SERVICE_NAME = "service.yunion.io/name"

	YUNION_ENDPOINT_REGION    = "endpoint.yunion.io/region"
	YUNION_ENDPOINT_NAME      = "endpoint.yunion.io/name"
	YUNION_ENDPOINT_ZONE      = "endpoint.yunion.io/zone"
	YUNION_ENDPOINT_INTERFACE = "endpoint.yunion.io/interface"
	YUNION_ENDPOINT_PROTOCOL  = "endpoint.yunion.io/protocol"
	YUNION_ENDPOINT_SUFFIX    = "endpoint.yunion.io/suffix"
	YUNION_ENDPOINT_DOMAIN    = "endpoint.yunion.io/domain"
)

type serviceInfo struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Id   string `json:"id"`
}

type endpointInfo struct {
	Name      string
	ServiceId string
	Region    string
	Zone      string
	Interface string
	Url       string
}

func getCloudServiceInfo(svc *api.Service) *serviceInfo {
	return &serviceInfo{
		Type: svc.ObjectMeta.Annotations[YUNION_SERVICE_TYPE],
		Name: svc.ObjectMeta.Annotations[YUNION_SERVICE_NAME],
	}
}

func (info *serviceInfo) getOrCreateFromCloud() (*serviceInfo, error) {
	session, err := models.GetAdminSession()
	if err != nil {
		return nil, err
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(info.Name), "name__icontains")
	query.Add(jsonutils.NewString(info.Type), "type__icontains")
	ret, err := modules.ServicesV3.List(session, query)
	if err != nil {
		return nil, err
	}
	var obj jsonutils.JSONObject = nil
	for _, nObj := range ret.Data {
		name, _ := nObj.GetString("name")
		if name == info.Name {
			obj = nObj
		}
	}
	if obj == nil {
		// create it
		createParams := jsonutils.NewDict()
		createParams.Add(jsonutils.NewString(info.Type), "type")
		createParams.Add(jsonutils.NewString(info.name), "name")
		createParams.Add(jsonutils.JSONTrue, "enabled")
		obj, err = modules.ServicesV3.Create(session, createParams)
		if err != nil {
			return nil, err
		}
	}
	cloudSvc := serviceInfo{}
	err = obj.Unmarshal(&cloudSvc)
	return cloudSvc, err
}

func (ep *endpointInfo) toCreateParams() (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(ep.ServiceId), "service_id")
	region := mcclient.RegionID(ep.Region, ep.Zone)
	params.Add(jsonutils.NewString(region), "region_id")
	params.Add(jsonutils.NewString(ep.Interface), "interface")
	params.Add(jsonutils.NewString(ep.Url), "url")
	params.Add(jsonutils.JSONTrue, "enabled")
	if ep.Name != "" {
		params.Add(jsonutils.NewString(ep.Name), "name")
	}
	return params
}

func getCloudEndpointInfo(svc *api.Service, cloudSvc *serviceInfo) (*endpointInfo, error) {
	anno := svc.ObjectMeta.Annotations
	region := anno[YUNION_ENDPOINT_REGION]
	zone := anno[YUNION_ENDPOINT_ZONE]
	inf := anno[YUNION_ENDPOINT_INTERFACE]
	if inf == "" {
		inf = "internal"
	}
	port := 0
	if len(svc.Spec.Ports) > 0 {
		port = svc.Spec.Ports[0].Port
	}
	endPointUrl := fmt.Sprintf("%s.%s", svc.Name, svc.Namespace)
	domain := anno[YUNION_ENDPOINT_DOMAIN]
	if domain != "" {
		endPointUrl = fmt.Sprintf("%s.%s", endPointUrl, domain)
	}
	if port > 0 {
		endPointUrl = fmt.Sprintf("%s:%d", endPointUrl, port)
	}
	suffix := anno[YUNION_ENDPOINT_SUFFIX]
	if suffix != "" {
		endPointUrl = filepath.Join(endPointUrl, suffix)
	}
	protocol := anno[YUNION_ENDPOINT_PROTOCOL]
	if protocol == "" {
		protocol = "http"
	}
	endPointUrl = fmt.Sprintf("%s://%s", protocol, endPointUrl)
	return &endpointInfo{
		Name:      anno[YUNION_ENDPOINT_NAME],
		ServiceId: cloudSvc.Id,
		Region:    region,
		Zone:      zone,
		Url:       endPointUrl,
		Interface: inf,
	}, nil
}

func createCloudEndpointByService(svc *api.Service) error {
	cloudSvc, err := getCloudServiceInfo(svc).getOrCreateFromCloud()
	if err != nil {
		return err
	}
}
