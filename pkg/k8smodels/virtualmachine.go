package k8smodels

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	VirtualMachineManager *SVirtualMachineManager
)

func init() {
	VirtualMachineManager = &SVirtualMachineManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(new(SVirtualMachine), "virtualmachine", "virtualmachines"),
	}
	VirtualMachineManager.SetVirtualObject(VirtualMachineManager)
}

type SVirtualMachineManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SVirtualMachine struct {
	model.SK8SNamespaceResourceBase
	UnstructuredResourceBase
}

func (m *SVirtualMachineManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameVirtualMachine,
		KindName:     apis.KindNameVirtualMachine,
		Object:       &unstructured.Unstructured{},
	}
}

func (obj *SVirtualMachine) GetAPIObject() (*apis.VirtualMachine, error) {
	vm := new(apis.VirtualMachine)
	if err := obj.ConvertToAPIObject(obj, vm); err != nil {
		return nil, err
	}
	return vm, nil
}

func (obj *SVirtualMachine) FillAPIObjectBySpec(specObj jsonutils.JSONObject, output IUnstructuredOutput) error {
	vm := output.(*apis.VirtualMachine)
	vm.Hypervisor, _ = specObj.GetString("vmConfig", "hypervisor")
	if cpuCount, _ := specObj.Int("vmConfig", "vcpuCount"); cpuCount != 0 {
		vm.VcpuCount = &cpuCount
	}
	if mem, _ := specObj.Int("vmConfig", "vmemSizeGB"); mem != 0 {
		vm.VmemSizeGB = &mem
	}
	instanceType, _ := specObj.GetString("vmConfig", "instanceType")
	vm.InstanceType = instanceType
	return nil
}

func (obj *SVirtualMachine) FillAPIObjectByStatus(statusObj jsonutils.JSONObject, output IUnstructuredOutput) error {
	vm := output.(*apis.VirtualMachine)
	phase, _ := statusObj.GetString("phase")
	vm.VirtualMachineStatus.Status = phase
	createTimes, _ := statusObj.Int("createTimes")
	vm.VirtualMachineStatus.CreateTimes = int32(createTimes)
	extInfo := &vm.VirtualMachineStatus.ExternalInfo
	if extraInfoObj, err := statusObj.Get("externalInfo"); err == nil {
		extraInfoObj.Unmarshal(extInfo)
	}
	return nil
}
