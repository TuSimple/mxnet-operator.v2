package mxcontroller

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"

	mxv1alpha2 "github.com/kubeflow/mxnet-operator.v2/pkg/apis/mxnet/v1alpha2"
	mxlogger "github.com/kubeflow/mxnet-operator.v2/pkg/logger"
)

const (
	failedMarshalMXJobReason = "FailedMarshalMXJob"
	terminatedMXJobReason    = "MXJobTerminated"
)

// When a pod is added, set the defaults and enqueue the current mxjob.
func (tc *MXController) addMXJob(obj interface{}) {
	// Convert from unstructured object.
	mxJob, err := mxJobFromUnstructured(obj)
	if err != nil {
		un, ok := obj.(*metav1unstructured.Unstructured)
		logger := &log.Entry{}
		if ok {
			logger = mxlogger.LoggerForUnstructured(un, mxv1alpha2.Kind)
		}
		logger.Errorf("Failed to convert the MXJob: %v", err)
		// Log the failure to conditions.
		if err == errFailedMarshal {
			errMsg := fmt.Sprintf("Failed to unmarshal the object to MXJob object: %v", err)
			logger.Warn(errMsg)
			tc.Recorder.Event(mxJob, v1.EventTypeWarning, failedMarshalMXJobReason, errMsg)
		}
		return
	}

	// Set default for the new mxjob.
	scheme.Scheme.Default(mxJob)

	msg := fmt.Sprintf("MXJob %s is created.", mxJob.Name)
	logger := mxlogger.LoggerForJob(mxJob)
	logger.Info(msg)

	// Add a created condition.
	err = updateMXJobConditions(mxJob, mxv1alpha2.MXJobCreated, mxJobCreatedReason, msg)
	if err != nil {
		logger.Errorf("Append mxJob condition error: %v", err)
		return
	}

	// Convert from mxjob object
	err = unstructuredFromMXJob(obj, mxJob)
	if err != nil {
		logger.Error("Failed to convert the obj: %v", err)
		return
	}
	tc.enqueueMXJob(obj)
}

// When a pod is updated, enqueue the current mxjob.
func (tc *MXController) updateMXJob(old, cur interface{}) {
	oldMXJob, err := mxJobFromUnstructured(old)
	if err != nil {
		return
	}
	log.Infof("Updating mxjob: %s", oldMXJob.Name)
	tc.enqueueMXJob(cur)
}

func (tc *MXController) deletePodsAndServices(mxJob *mxv1alpha2.MXJob, pods []*v1.Pod) error {
	if len(pods) == 0 {
		return nil
	}
	tc.Recorder.Event(mxJob, v1.EventTypeNormal, terminatedMXJobReason,
		"MXJob is terminated, deleting pods and services")

	// Delete nothing when the cleanPodPolicy is None.
	if *mxJob.Spec.CleanPodPolicy == mxv1alpha2.CleanPodPolicyNone {
		return nil
	}

	for _, pod := range pods {
		if *mxJob.Spec.CleanPodPolicy == mxv1alpha2.CleanPodPolicyRunning && pod.Status.Phase != v1.PodRunning && pod.Status.Phase != v1.PodSucceeded{
			continue
		}

		if err := tc.PodControl.DeletePod(pod.Namespace, pod.Name, mxJob); err != nil {
			return err
		}
		// Pod and service have the same name, thus the service could be deleted using pod's name.
		if err := tc.ServiceControl.DeleteService(pod.Namespace, pod.Name, mxJob); err != nil {
			return err
		}
	}
	return nil
}

func (tc *MXController) cleanupMXJob(mxJob *mxv1alpha2.MXJob) error {
	currentTime := time.Now()
	ttl := mxJob.Spec.TTLSecondsAfterFinished
	if ttl == nil {
		// do nothing if the cleanup delay is not set
		return nil
	}
	duration := time.Second * time.Duration(*ttl)
	if currentTime.After(mxJob.Status.CompletionTime.Add(duration)) {
		err := tc.deleteMXJobHandler(mxJob)
		if err != nil {
			mxlogger.LoggerForJob(mxJob).Warnf("Cleanup MXJob error: %v.", err)
			return err
		}
		return nil
	}
	key, err := KeyFunc(mxJob)
	if err != nil {
		mxlogger.LoggerForJob(mxJob).Warnf("Couldn't get key for mxjob object: %v", err)
		return err
	}
	tc.WorkQueue.AddRateLimited(key)
	return nil
}

// deleteMXJob delets the given MXJob.
func (tc *MXController) deleteMXJob(mxJob *mxv1alpha2.MXJob) error {
	return tc.mxJobClientSet.KubeflowV1alpha2().MXJobs(mxJob.Namespace).Delete(mxJob.Name, &metav1.DeleteOptions{})
}
