package common

import (
	"encoding/json"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
)

func JsonDecode(data jsonutils.JSONObject, obj interface{}) error {
	dataStr, err := data.GetString()
	if err != nil {
		return err
	}
	err = json.NewDecoder(strings.NewReader(dataStr)).Decode(obj)
	return err
}

func GetK8sObjectCreateMeta(data jsonutils.JSONObject) (*metav1.ObjectMeta, error) {
	name, err := data.GetString("name")
	if err != nil {
		return nil, httperrors.NewInputParameterError("name not provided")
	}

	labels := make(map[string]string)
	annotations := make(map[string]string)

	data.Unmarshal(&labels, "labels")
	data.Unmarshal(&annotations, "annotations")
	return &metav1.ObjectMeta{
		Name:        name,
		Labels:      labels,
		Annotations: annotations,
	}, nil
}

func GenerateName(base string) string {
	maxNameLength := 63
	randomLength := 5
	maxGeneratedNameLength := maxNameLength - randomLength
	if len(base) > maxGeneratedNameLength {
		base = base[:maxGeneratedNameLength]
	}
	return fmt.Sprintf("%s%s", base, rand.String(randomLength))
}
