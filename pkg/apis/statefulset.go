package apis

// StatefulSet is a presentation layer view of Kubernetes Stateful Set resource. This means it is
// Stateful Set plus additional augmented data we can get from other sources (like services that
// target the same pods).
type StatefulSet struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this Pet Set.
	Pods PodInfo `json:"podsInfo"`

	// Container images of the Stateful Set.
	ContainerImages []string `json:"containerImages"`

	// Init container images of the Stateful Set.
	InitContainerImages []string          `json:"initContainerImages"`
	Status              string            `json:"status"`
	Selector            map[string]string `json:"selector"`
}

// StatefulSetDetail is a presentation layer view of Kubernetes Stateful Set resource. This means it is Stateful
// Set plus additional augmented data we can get from other sources (like services that target the same pods).
type StatefulSetDetail struct {
	StatefulSet
	PodList     []Pod     `json:"pods"`
	EventList   []Event   `json:"events"`
	ServiceList []Service `json:"services"`
}
