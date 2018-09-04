// -----------------------------------
// FILE TAKEN FROM: k8s.io/kubernetes/pkg/fieldpath/fieldpath.go
// Commit hash: 6f52cd3d0ba0cafc7d427a29dfaf69b97c9c1968 (K8S 1.7.2)
// -----------------------------------

package pod

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
)

// FormatMap formats map[string]string to a string.
func FormatMap(m map[string]string) (fmtStr string) {
	for key, value := range m {
		fmtStr += fmt.Sprintf("%v=%q\n", key, value)
	}
	fmtStr = strings.TrimSuffix(fmtStr, "\n")

	return
}

// ExtractFieldPathAsString extracts the field from the given object
// and returns it as a string.  The object must be a pointer to an
// API type.
func ExtractFieldPathAsString(obj interface{}, fieldPath string) (string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", nil
	}

	switch fieldPath {
	case "metadata.annotations":
		return FormatMap(accessor.GetAnnotations()), nil
	case "metadata.labels":
		return FormatMap(accessor.GetLabels()), nil
	case "metadata.name":
		return accessor.GetName(), nil
	case "metadata.namespace":
		return accessor.GetNamespace(), nil
	}

	return "", fmt.Errorf("unsupported fieldPath: %v", fieldPath)
}
