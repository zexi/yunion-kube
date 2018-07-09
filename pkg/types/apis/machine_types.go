package apis

type Node struct {
	Name              string        `json:"name"`
	Etcd              bool          `json:"etcd"`
	ControlPlane      bool          `json:"controlPlane"`
	Worker            bool          `json:"worker"`
	RequestedHostname string        `json:"requestedHostname"`
	CustomConfig      *CustomConfig `json:"customConfig"`
	DockerInfo        *DockerInfo   `json:"dockerInfo"`
	//Status            NodeStatus    `json:"status"`
}

//type NodeStatus struct {
//DockerInfo *DockerInfo `json:"dockerInfo"`
//}

type CustomConfig struct {
	Address         string   `json:"address"`
	InternalAddress string   `json:"internalAddress"`
	DockerSocket    string   `json:"dockerSocket"`
	Roles           []string `json:"roles"`
}

type NodeSpec struct {
}
