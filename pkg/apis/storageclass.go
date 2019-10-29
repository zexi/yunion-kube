package apis

// StorageClass is a representation of a kubernetes StorageClass object.
type StorageClass struct {
	ObjectMeta
	TypeMeta

	// provisioner is the driver expected to handle this StorageClass.
	// This is an optionally-prefixed name, like a label key.
	// For example: "kubernetes.io/gce-pd" or "kubernetes.io/aws-ebs".
	// This value may not be empty.
	Provisioner string `json:"provisioner"`

	// parameters holds parameters for the provisioner.
	// These values are opaque to the  system and are passed directly
	// to the provisioner.  The only validation done on keys is that they are
	// not empty.  The maximum number of parameters is
	// 512, with a cumulative max size of 256K
	// +optional
	Parameters map[string]string `json:"parameters"`
}

// StorageClassDetail provides the presentation layer view of Kubernetes StorageClass resource,
// It is StorageClassDetail plus PersistentVolumes associated with StorageClass.
type StorageClassDetail struct {
	StorageClass
	PersistentVolumeList []PersistentVolume `json:"persistentVolumes"`
}
