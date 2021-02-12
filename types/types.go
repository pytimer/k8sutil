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
