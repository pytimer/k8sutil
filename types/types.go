package types

// List of all resource kinds supported by the UI.
const (
	ResourceKindDaemonSet             = "daemonset"
	ResourceKindJob                   = "job"
	ResourceKindPod                   = "pod"
	ResourceKindReplicaSet            = "replicaset"
	ResourceKindReplicationController = "replicationcontroller"
	ResourceKindStatefulSet           = "statefulset"
	ResourceKindService               = "service"
)

// List of all resource pluralized kinds supported by the UI.
const (
	ResourcePluralKindDeployment            = "deployments"
	ResourcePluralKindDaemonSet             = "daemonsets"
	ResourcePluralKindJob                   = "jobs"
	ResourcePluralKindPod                   = "pods"
	ResourcePluralKindReplicaSet            = "replicasets"
	ResourcePluralKindReplicationController = "replicationcontrollers"
	ResourcePluralKindStatefulSet           = "statefulsets"
	ResourcePluralKindService               = "services"
)

const (
	// LabelNodeRolePrefix is a label prefix for node roles
	// It's copied over to here until it's merged in core: https://github.com/kubernetes/kubernetes/pull/39112
	LabelNodeRolePrefix = "node-role.kubernetes.io/"

	// NodeLabelRole specifies the role of a node
	NodeLabelRole = "kubernetes.io/role"

	NodeRoleMaster       = "master"
	NodeRoleControlPlane = "control-plane"
)

const (
	ResourceGPU = "nvidia.com/gpu"
)
