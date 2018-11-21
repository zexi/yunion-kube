package sync

import (
	"errors"
	"fmt"
	"path/filepath"

	api "k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/models"
	o "yunion.io/x/yunion-kube/pkg/options"
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
	RegionId  string
	Interface string
	Url       string
	Id        string
}

func shouldAddServiceToCloud(svc *api.Service) bool {
	anno := svc.ObjectMeta.Annotations
	if len(anno) == 0 {
		return false
	}
	for _, annoKey := range []string{
		YUNION_SERVICE_TYPE,
		YUNION_SERVICE_NAME,
	} {
		if _, ok := anno[annoKey]; !ok {
			return false
		}
	}
	return true
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
		createParams.Add(jsonutils.NewString(info.Name), "name")
		createParams.Add(jsonutils.JSONTrue, "enabled")
		obj, err = modules.ServicesV3.Create(session, createParams)
		if err != nil {
			return nil, err
		}
	}
	cloudSvc := serviceInfo{}
	err = obj.Unmarshal(&cloudSvc)
	return &cloudSvc, err
}

func (ep *endpointInfo) toCreateParams() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(ep.ServiceId), "service_id")
	params.Add(jsonutils.NewString(ep.RegionId), "region_id")
	params.Add(jsonutils.NewString(ep.Interface), "interface")
	params.Add(jsonutils.NewString(ep.Url), "url")
	params.Add(jsonutils.JSONTrue, "enabled")
	if ep.Name != "" {
		params.Add(jsonutils.NewString(ep.Name), "name")
	}
	return params
}

func (ep *endpointInfo) toUpdateParams() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(ep.Url), "url")
	if ep.Name != "" {
		params.Add(jsonutils.NewString(ep.Name), "name")
	}
	params.Add(jsonutils.JSONTrue, "enabled")
	return params
}

func (ep *endpointInfo) toSearchParams() *jsonutils.JSONDict {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(ep.ServiceId), "service_id")
	params.Add(jsonutils.NewString(ep.RegionId), "region_id")
	params.Add(jsonutils.NewString(ep.Interface), "interface")
	return params
}

func (ep *endpointInfo) createOrUpdateFromCloud() (jsonutils.JSONObject, error) {
	session, err := models.GetAdminSession()
	if err != nil {
		return nil, err
	}
	if len(ep.Id) == 0 {
		return modules.EndpointsV3.Create(session, ep.toCreateParams())
	}
	return modules.EndpointsV3.Patch(session, ep.Id, ep.toUpdateParams())
}

func getCloudEndpointInfo(svc *api.Service, cloudSvc *serviceInfo) (*endpointInfo, error) {
	anno := svc.ObjectMeta.Annotations
	region := anno[YUNION_ENDPOINT_REGION]
	if region == "" {
		region = o.Options.Region
	}
	zone := anno[YUNION_ENDPOINT_ZONE]
	regionId := mcclient.RegionID(region, zone)
	inf := anno[YUNION_ENDPOINT_INTERFACE]
	if inf == "" {
		inf = "internal"
	}
	if regionId == "" {
		return nil, errors.New("RegionId is empty")
	}
	if cloudSvc.Id == "" {
		return nil, errors.New("ServiceId is empty")
	}
	if inf == "" || !utils.IsInStringArray(inf, []string{"internal", "public", "admin"}) {
		return nil, fmt.Errorf("Invalid endpoint interface: %q", inf)
	}
	var port int32
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
	ep := &endpointInfo{
		Name:      anno[YUNION_ENDPOINT_NAME],
		ServiceId: cloudSvc.Id,
		RegionId:  regionId,
		Url:       endPointUrl,
		Interface: inf,
	}
	session, err := models.GetAdminSession()
	if err != nil {
		return nil, err
	}
	eps, err := modules.EndpointsV3.List(session, ep.toSearchParams())
	if err != nil {
		return nil, err
	}
	if len(eps.Data) == 0 {
		return ep, nil
	}
	epId, err := eps.Data[0].GetString("id")
	if err != nil {
		return nil, err
	}
	ep.Id = epId
	return ep, nil
}

func createOrUpdateCloudEndpointByService(svc *api.Service) error {
	cloudSvc, err := getCloudServiceInfo(svc).getOrCreateFromCloud()
	if err != nil {
		return err
	}
	epInfo, err := getCloudEndpointInfo(svc, cloudSvc)
	if err != nil {
		return err
	}
	ret, err := epInfo.createOrUpdateFromCloud()
	if err != nil {
		return err
	}
	log.Infof("Create endpoint %s", ret)
	return nil
}

func deleteCloudEndpointByService(svc *api.Service) error {
	cloudSvc, err := getCloudServiceInfo(svc).getOrCreateFromCloud()
	if err != nil {
		return err
	}
	epInfo, err := getCloudEndpointInfo(svc, cloudSvc)
	if err != nil {
		return err
	}
	if epInfo.Id != "" {
		session, err := models.GetAdminSession()
		if err != nil {
			return err
		}
		_, err = modules.EndpointsV3.Delete(session, epInfo.Id, nil)
	}
	return err
}
