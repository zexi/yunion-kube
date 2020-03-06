package drivers

import (
	"yunion.io/x/yunion-kube/pkg/apis"
)

func GetControlplaneMachineDatas(clusterId string, data []*apis.CreateMachineData) ([]*apis.CreateMachineData, []*apis.CreateMachineData) {
	controls := make([]*apis.CreateMachineData, 0)
	nodes := make([]*apis.CreateMachineData, 0)
	for _, d := range data {
		if len(clusterId) != 0 {
			d.ClusterId = clusterId
		}
		if d.Role == apis.RoleTypeControlplane {
			controls = append(controls, d)
		} else {
			nodes = append(nodes, d)
		}
	}
	return controls, nodes
}
