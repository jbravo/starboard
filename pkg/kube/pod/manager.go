package pod

import (
	"fmt"
	"io"

	"github.com/aquasecurity/starboard/pkg/kube"

	apps "k8s.io/api/apps/v1"
	batch "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Manager struct {
	clientset kubernetes.Interface
}

func NewPodManager(clientset kubernetes.Interface) *Manager {
	return &Manager{
		clientset: clientset,
	}
}

// GetPodSpecByWorkload returns a PodSpec of the specified Workload.
func (pw *Manager) GetPodSpecByWorkload(workload kube.Workload) (spec core.PodSpec, err error) {
	ns := workload.Namespace
	switch workload.Kind {
	case kube.WorkloadKindPod:
		var pod *core.Pod
		pod, err = pw.GetPodByName(ns, workload.Name)
		if err != nil {
			return
		}
		spec = pod.Spec
		return
	case kube.WorkloadKindReplicaSet:
		var rs *apps.ReplicaSet
		rs, err = pw.clientset.AppsV1().ReplicaSets(ns).Get(workload.Name, meta.GetOptions{})
		if err != nil {
			return
		}
		spec = rs.Spec.Template.Spec
		return
	case kube.WorkloadKindReplicationController:
		var rc *core.ReplicationController
		rc, err = pw.clientset.CoreV1().ReplicationControllers(ns).Get(workload.Name, meta.GetOptions{})
		if err != nil {
			return
		}
		spec = rc.Spec.Template.Spec
		return
	case kube.WorkloadKindDeployment:
		var deploy *apps.Deployment
		deploy, err = pw.clientset.AppsV1().Deployments(ns).Get(workload.Name, meta.GetOptions{})
		if err != nil {
			return
		}
		spec = deploy.Spec.Template.Spec
		return
	case kube.WorkloadKindStatefulSet:
		var sts *apps.StatefulSet
		sts, err = pw.clientset.AppsV1().StatefulSets(ns).Get(workload.Name, meta.GetOptions{})
		if err != nil {
			return
		}
		spec = sts.Spec.Template.Spec
		return
	case kube.WorkloadKindDaemonSet:
		var ds *apps.DaemonSet
		ds, err = pw.clientset.AppsV1().DaemonSets(ns).Get(workload.Name, meta.GetOptions{})
		if err != nil {
			return
		}
		spec = ds.Spec.Template.Spec
		return
	case kube.WorkloadKindCronJob:
		var cj *batchv1beta1.CronJob
		cj, err = pw.clientset.BatchV1beta1().CronJobs(ns).Get(workload.Name, meta.GetOptions{})
		if err != nil {
			return
		}
		spec = cj.Spec.JobTemplate.Spec.Template.Spec
		return
	}
	err = fmt.Errorf("unrecognized workload: %s", workload.Kind)
	return
}

func (pw *Manager) GetPodByName(namespace, name string) (*core.Pod, error) {
	return pw.clientset.CoreV1().Pods(namespace).Get(name, meta.GetOptions{})
}

func (pw *Manager) GetPodLogsByJob(job *batch.Job, container string) (io.ReadCloser, error) {
	pod, err := pw.GetPodByJob(job)
	if err != nil {
		return nil, err
	}

	return pw.GetPodLogs(pod, container)
}

// GetPodByJob gets the Pod controller by the specified Job.
func (pw *Manager) GetPodByJob(job *batch.Job) (*core.Pod, error) {
	refreshedJob, err := pw.clientset.BatchV1().Jobs(job.Namespace).Get(job.Name, meta.GetOptions{})
	if err != nil {
		return nil, err
	}
	selector := fmt.Sprintf("controller-uid=%s", refreshedJob.Spec.Selector.MatchLabels["controller-uid"])
	podList, err := pw.clientset.CoreV1().Pods(job.Namespace).List(meta.ListOptions{
		LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	return &podList.Items[0], nil
}

func (pw *Manager) GetPodLogs(pod *core.Pod, container string) (io.ReadCloser, error) {
	req := pw.clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &core.PodLogOptions{
		Follow: true, Container: container})
	return req.Stream()
}
