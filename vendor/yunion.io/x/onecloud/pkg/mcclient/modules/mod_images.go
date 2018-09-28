package modules

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

type ImageManager struct {
	ResourceManager
}

const (
	IMAGE_META          = "X-Image-Meta-"
	IMAGE_META_PROPERTY = "X-Image-Meta-Property-"
)

func decodeMeta(str string) string {
	s, e := url.QueryUnescape(str)
	if e == nil && s != str {
		return decodeMeta(s)
	} else {
		return str
	}
}

func fetchImageMeta(h http.Header) jsonutils.JSONObject {
	meta := jsonutils.NewDict()
	meta.Add(jsonutils.NewDict(), "properties")
	for k, v := range h {
		if len(k) > len(IMAGE_META_PROPERTY) && k[:len(IMAGE_META_PROPERTY)] == IMAGE_META_PROPERTY {
			meta.Add(jsonutils.NewString(decodeMeta(v[0])), "properties", strings.ToLower(k[len(IMAGE_META_PROPERTY):]))
		} else if len(k) > len(IMAGE_META) && k[:len(IMAGE_META)] == IMAGE_META {
			meta.Add(jsonutils.NewString(decodeMeta(v[0])), strings.ToLower(k[len(IMAGE_META):]))
		}
	}
	return meta
}

func (this *ImageManager) GetById(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	path := fmt.Sprintf("%s/%s", this.URLPath(), id)
	if params != nil {
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	h, _, e := this.jsonRequest(session, "HEAD", path, nil, nil)
	if e != nil {
		return nil, e
	}
	return fetchImageMeta(h), nil
}

func (this *ImageManager) GetByName(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var dict *jsonutils.JSONDict
	if params == nil {
		dict = jsonutils.NewDict()
	} else {
		dict, _ = params.(*jsonutils.JSONDict)
	}
	dict.Add(jsonutils.NewString(id), "name")
	dict.Add(jsonutils.JSONTrue, "details")
	listresults, e := this.List(session, dict)
	if e != nil {
		return nil, e
	}
	if len(listresults.Data) == 0 {
		return nil, httperrors.NewImageNotFoundError(id)
	} else if len(listresults.Data) == 1 {
		return listresults.Data[0], nil
	} else {
		return nil, httperrors.NewDuplicateNameError("image name", id)
	}
}

func (this *ImageManager) Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	r, e := this.GetById(session, id, params)
	if e == nil {
		return r, e
	}
	je, ok := e.(*httputils.JSONClientError)
	if ok && je.Code == 404 {
		return this.GetByName(session, id, params)
	} else {
		return nil, e
	}
}

func (this *ImageManager) GetId(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (string, error) {
	img, e := this.Get(session, id, nil)
	if e != nil {
		return "", e
	}
	return img.GetString("id")
}

func (this *ImageManager) BatchGet(session *mcclient.ClientSession, idlist []string, params jsonutils.JSONObject) []SubmitResult {
	return BatchDo(idlist, func(id string) (jsonutils.JSONObject, error) {
		return this.Get(session, id, params)
	})
}

func (this *ImageManager) List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*ListResult, error) {
	path := fmt.Sprintf("/%s", this.URLPath())
	if params != nil {
		details, _ := params.Bool("details")
		if details {
			path = fmt.Sprintf("%s/detail", path)
		}
		dictparams, _ := params.(*jsonutils.JSONDict)
		dictparams.RemoveIgnoreCase("details")
		qs := params.QueryString()
		if len(qs) > 0 {
			path = fmt.Sprintf("%s?%s", path, qs)
		}
	}
	return this._list(session, path, this.KeywordPlural)
}

func (this *ImageManager) GetPrivateImageCount(s *mcclient.ClientSession, ownerId string, isAdmin bool) (int, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("none"), "is_public")
	params.Add(jsonutils.NewString(ownerId), "owner")
	if isAdmin {
		params.Add(jsonutils.JSONTrue, "admin")
	}

	result, err := this.List(s, params)
	if err != nil {
		return 0, err
	}
	return len(result.Data), nil
}

type ImageUsageCount struct {
	Count int64
	Size  int64
}

func (this *ImageManager) countUsage(session *mcclient.ClientSession, deleted bool) (map[string]*ImageUsageCount, error) {
	var limit int64 = 1000
	var offset int64 = 0
	ret := make(map[string]*ImageUsageCount)
	count := func(ret map[string]*ImageUsageCount, results *ListResult) {
		for _, r := range results.Data {
			format, _ := r.GetString("disk_format")
			status, _ := r.GetString("status")
			img_size, _ := r.Int("size")
			if len(format) > 0 {
				if _, ok := ret[format]; !ok {
					ret[format] = &ImageUsageCount{}
				}
				ret[format].Size += img_size
				ret[format].Count += 1
			}
			if len(status) > 0 {
				if _, ok := ret[status]; !ok {
					ret[status] = &ImageUsageCount{}
				}
				ret[status].Size += img_size
				ret[status].Count += 1
			}
		}
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewInt(limit), "limit")
	query.Add(jsonutils.NewInt(offset), "offset")
	query.Add(jsonutils.JSONTrue, "admin")
	if deleted {
		query.Add(jsonutils.NewString("true"), "pending_delete")
	}
	if result, e := this.List(session, query); e != nil {
		return nil, e
	} else {
		count(ret, result)
		offset += limit

		for result.Total > int(offset) {
			query.Add(jsonutils.NewInt(offset), "offset")
			if result, e := this.List(session, query); e != nil {
				return nil, e
			} else {
				count(ret, result)
				offset += limit
			}
		}
	}
	return ret, nil
}

func (this *ImageManager) GetUsage(session *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	body := jsonutils.NewDict()
	pendingDelete := jsonutils.NewDict()
	for deleted, data := range map[bool]*jsonutils.JSONDict{false: body, true: pendingDelete} {
		if ret, err := this.countUsage(session, deleted); err != nil {
			return nil, err
		} else {
			for k, v := range ret {
				stat := jsonutils.NewDict()
				stat.Add(jsonutils.NewInt(v.Size), "size")
				stat.Add(jsonutils.NewInt(v.Count), "count")
				data.Add(stat, k)
			}
		}
	}
	body.Add(pendingDelete, "pending_delete")
	return body, nil
}

func setImageMeta(params jsonutils.JSONObject) (http.Header, error) {
	header := http.Header{}
	p, e := params.(*jsonutils.JSONDict).GetMap()
	if e != nil {
		return header, e
	}
	for k, v := range p {
		if ok, _ := utils.InStringArray(k, []string{"copy_from"}); ok {
			continue
		}
		if k == "properties" {
			pp, e := v.(*jsonutils.JSONDict).GetMap()
			if e != nil {
				return header, e
			}
			for kk, vv := range pp {
				vvs, _ := vv.GetString()
				header.Add(fmt.Sprintf("%s%s", IMAGE_META_PROPERTY, utils.Capitalize(kk)), vvs)
			}
		} else {
			vs, _ := v.GetString()
			header.Add(fmt.Sprintf("%s%s", IMAGE_META, utils.Capitalize(k)), vs)
		}
	}
	return header, nil
}

func (this *ImageManager) ListMemberProjects(s *mcclient.ClientSession, imageId string) (*ListResult, error) {
	result, e := this.ListMemberProjectIds(s, imageId)
	if e != nil {
		return nil, e
	}
	for i, member := range result.Data {
		projectIdstr, e := member.GetString("member_id")
		if e != nil {
			return nil, e
		}
		project, e := Projects.GetById(s, projectIdstr, nil)
		if e != nil {
			return nil, e
		}
		result.Data[i] = project
	}
	return result, nil
}

func (this *ImageManager) ListMemberProjectIds(s *mcclient.ClientSession, imageId string) (*ListResult, error) {
	path := fmt.Sprintf("/%s/%s/members", this.URLPath(), url.PathEscape(imageId))
	return this._list(s, path, "members")
}

func (this *ImageManager) AddMembership(s *mcclient.ClientSession, img string, proj string, canShare bool) error {
	image, e := Images.Get(s, img, nil)
	if e != nil {
		return e
	}
	projectId, e := Projects.GetId(s, proj, nil)
	if e != nil {
		return e
	}
	imageOwner, e := image.GetString("owner")
	if e != nil {
		return e
	}
	if imageOwner == projectId {
		return fmt.Errorf("Project %s owns image %s", proj, img)
	}
	imageName, e := image.GetString("name")
	if e != nil {
		return e
	}
	imageId, e := image.GetString("id")
	if e != nil {
		return e
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString(projectId), "owner")
	_, e = Images.GetByName(s, imageName, query)
	if e != nil {
		je, ok := e.(*httputils.JSONClientError)
		if ok && je.Code == 404 { // no same name image
			sharedImgIds, e := this.ListSharedImageIds(s, projectId)
			if e != nil {
				return e
			}
			for _, sharedImgId := range sharedImgIds.Data {
				sharedImgIdstr, e := sharedImgId.GetString()
				if e != nil {
					return e
				}
				if sharedImgIdstr == imageId { // already shared, do update
					break
				} else {
					sharedImg, e := this.GetById(s, sharedImgIdstr, nil)
					if e != nil {
						return e
					}
					sharedImgName, e := sharedImg.GetString("name")
					if e != nil {
						return e
					}
					if sharedImgName == imageName {
						return fmt.Errorf("Name %s conflict with other shared images", imageName)
					}
				}
			}
			return this._addMembership(s, imageId, projectId, canShare)
		}
	}
	return fmt.Errorf("Image name conflict")
}

func (this *ImageManager) _addMembership(s *mcclient.ClientSession, image_id string, project_id string, canShare bool) error {
	params := jsonutils.NewDict()
	// params.Add(jsonutils.NewString(project_id), "member_id")
	if canShare {
		params.Add(jsonutils.JSONTrue, "member", "can_share")
	} else {
		params.Add(jsonutils.JSONFalse, "member", "can_share")
	}
	path := fmt.Sprintf("/%s/%s/members/%s", this.URLPath(), url.PathEscape(image_id), url.PathEscape(project_id))
	_, e := this._put(s, path, params, "")
	return e
}

func (this *ImageManager) _addMemberships(s *mcclient.ClientSession, image_id string, projectIds []string, canShare bool) error {
	memberships := jsonutils.NewArray()
	for _, projectId := range projectIds {
		member := jsonutils.NewDict()
		member.Add(jsonutils.NewString(projectId), "member_id")
		if canShare {
			member.Add(jsonutils.JSONTrue, "can_share")
		} else {
			member.Add(jsonutils.JSONFalse, "can_share")
		}
		memberships.Add(member)
	}
	params := jsonutils.NewDict()
	params.Add(memberships, "memberships")
	path := fmt.Sprintf("/%s/%s/members", this.URLPath(), url.PathEscape(image_id))
	_, e := this._put(s, path, params, "")
	return e
}

func (this *ImageManager) RemoveMembership(s *mcclient.ClientSession, image string, project string) error {
	imgid, e := this.GetId(s, image, nil)
	if e != nil {
		return e
	}
	projid, e := Projects.GetId(s, project, nil)
	if e != nil {
		return e
	}
	return this._removeMembership(s, imgid, projid)
}

func (this *ImageManager) _removeMembership(s *mcclient.ClientSession, image_id string, project_id string) error {
	path := fmt.Sprintf("/%s/%s/members/%s", this.URLPath(), url.PathEscape(image_id), url.PathEscape(project_id))
	_, e := this._delete(s, path, nil, "")
	return e
}

func (this *ImageManager) ListSharedImageIds(s *mcclient.ClientSession, projectId string) (*ListResult, error) {
	path := fmt.Sprintf("/shared-images/%s", projectId)
	// {"shared_images": [{"image_id": "4d82c731-937e-4420-959b-de9c213efd2b", "can_share": false}]}
	return this._list(s, path, "shared_images")
}

func (this *ImageManager) ListSharedImages(s *mcclient.ClientSession, projectId string) (*ListResult, error) {
	result, e := this.ListSharedImageIds(s, projectId)
	if e != nil {
		return nil, e
	}
	for i, imgId := range result.Data {
		imgIdstr, e := imgId.GetString("image_id")
		if e != nil {
			return nil, e
		}
		img, e := this.GetById(s, imgIdstr, nil)
		if e != nil {
			return nil, e
		}
		result.Data[i] = img
	}
	return result, nil
}

func (this *ImageManager) Create(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return this._create(s, params, nil, 0)
}

func (this *ImageManager) Upload(s *mcclient.ClientSession, params jsonutils.JSONObject, body io.Reader, size int64) (jsonutils.JSONObject, error) {
	return this._create(s, params, body, size)
}

func (this *ImageManager) IsNameDuplicate(s *mcclient.ClientSession, name string) (bool, error) {
	dupName := true
	_, e := this.GetByName(s, name, nil)
	if e != nil {
		switch e.(type) {
		case *httputils.JSONClientError:
			je := e.(*httputils.JSONClientError)
			if je.Code == 404 {
				dupName = false
			}
		default:
			log.Errorf("GetByName fail %s", e)
			return false, e
		}
	}
	return dupName, nil
}

func (this *ImageManager) _create(s *mcclient.ClientSession, params jsonutils.JSONObject, body io.Reader, size int64) (jsonutils.JSONObject, error) {
	format, _ := params.GetString("disk-format")
	if len(format) == 0 {
		format, _ = params.GetString("disk_format")
		if len(format) == 0 {
			return nil, httperrors.NewMissingParameterError("disk_format")
		}
	}
	exists, _ := utils.InStringArray(format, []string{"qcow2", "raw", "vhd", "vmdk", "iso", "docker"})
	if !exists {
		return nil, fmt.Errorf("Unsupported image format %s", format)
	}
	imageId, _ := params.GetString("image_id")
	path := fmt.Sprintf("/%s", this.URLPath())
	method := "POST"
	if len(imageId) == 0 {
		osType, err := params.GetString("properties", "os_type")
		if err != nil {
			return nil, httperrors.NewMissingParameterError("os_type")
		}
		exists, _ = utils.InStringArray(osType, []string{"Windows", "Linux", "Freebsd", "Android", "macOS", "VMWare"})
		if !exists {
			return nil, fmt.Errorf("OS type must be specified")
		}
		name, _ := params.GetString("name")
		if len(name) == 0 {
			return nil, httperrors.NewMissingParameterError("name")
		}
		dupName, e := this.IsNameDuplicate(s, name)
		if dupName {
			return nil, httperrors.NewDuplicateNameError("name", name)
		}
		if e != nil {
			return nil, fmt.Errorf("Check name duplicate error %s", e)
		}
	} else {
		path = fmt.Sprintf("/%s/%s", this.URLPath(), imageId)
		method = "PUT"
	}
	headers, e := setImageMeta(params)
	if e != nil {
		return nil, e
	}
	copyFromUrl, _ := params.GetString("copy_from")
	if len(copyFromUrl) != 0 {
		if size != 0 {
			return nil, fmt.Errorf("Can't use copy_from and upload file at the same time")
		}
		body = nil
		size = 0
		headers.Set("x-glance-api-copy-from", copyFromUrl)
	}
	headers.Set(fmt.Sprintf("%s%s", IMAGE_META, utils.Capitalize("container-format")), "bare")
	if body != nil {
		headers.Add("Content-Type", "application/octet-stream")
		if size > 0 {
			headers.Add("Content-Length", fmt.Sprintf("%d", size))
		}
	}
	resp, err := this.rawRequest(s, method, path, headers, body)
	_, json, err := s.ParseJSONResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return json.Get("image")
}

func (this *ImageManager) Update(s *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	img, err := this.Get(s, id, nil)
	if err != nil {
		return nil, err
	}
	idstr, err := img.GetString("id")
	if err != nil {
		return nil, err
	}
	properties, _ := img.Get("properties")
	if properties != nil {
		propDict := properties.(*jsonutils.JSONDict)
		propMap, _ := propDict.GetMap()
		if propMap != nil {
			paramsDict := params.(*jsonutils.JSONDict)
			for k, val := range propMap {
				if !paramsDict.Contains("properties", k) {
					paramsDict.Add(val, "properties", k)
				}
			}
		}
	}

	return this._update(s, idstr, params, nil)
}

func (this *ImageManager) _update(s *mcclient.ClientSession, id string, params jsonutils.JSONObject, body io.Reader) (jsonutils.JSONObject, error) {
	headers, err := setImageMeta(params)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/%s/%s", this.URLPath(), url.PathEscape(id))
	resp, err := this.rawRequest(s, "PUT", path, headers, body)
	_, json, err := s.ParseJSONResponse(resp, err)
	if err != nil {
		return nil, err
	}
	return json.Get("image")
}

func (this *ImageManager) Download(s *mcclient.ClientSession, id string) (jsonutils.JSONObject, io.Reader, error) {
	path := fmt.Sprintf("/%s/%s", this.URLPath(), url.PathEscape(id))
	resp, err := this.rawRequest(s, "GET", path, nil, nil)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fetchImageMeta(resp.Header), resp.Body, nil
	} else {
		_, _, err = s.ParseJSONResponse(resp, err)
		return nil, nil, err
	}
}

var (
	Images ImageManager
)

func init() {
	Images = ImageManager{NewImageManager("image", "images",
		[]string{"ID", "Name", "Tags", "Disk_format",
			"Size", "Is_public", "OS_Type",
			"OS_Distribution", "OS_version",
			"Min_disk", "Min_ram", "Status",
			"Notes", "OS_arch", "Preference",
			"OS_Codename", "Parent_id"},
		[]string{"Owner", "Owner_name"})}
	register(&Images)
}
