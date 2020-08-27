/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ref "k8s.io/client-go/tools/reference"
	"time"

	"github.com/go-logr/logr"
	kbatch "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"
	webappv1 "github.com/mchirico/kbt/api/v1"
)

// DevReconciler reconciles a Dev object
type DevReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Clock
}

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

// +kubebuilder:rbac:groups=webapp.dev.cwxstat.io,resources=devs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.dev.cwxstat.io,resources=devs/status,verbs=get;update;patch

// +kubebuilder:rbac:groups=webapp.dev.cwxstat.io,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.dev.cwxstat.io,resources=jobs/status,verbs=get

var (
	scheduledTimeAnnotation = "webapp.dev.cwxstat.io/scheduled-at"
	jobOwnerKey             = ".metadata.controller"
	apiGVStr                = webappv1.GroupVersion.String()
)

func (r *DevReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("dev", req.NamespacedName)

	var dev webappv1.Dev
	if err := r.Get(ctx, req.NamespacedName, &dev); err != nil {
		log.Error(err, "unable to fetch dev ..")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg := fmt.Sprintf("\n\nDEBUG:__________(start a)"+
		"\n\ndev.Name: %v\ndev.Status.Active:%v"+
		"\ndev: %v"+
		"\n\nDEBUG:__________(end a)\n\n\n", dev.Name, dev.Status.Active, dev)
	log.Info(msg)

	var devList webappv1.DevList
	if err := r.List(ctx, &devList); err != nil {
		log.Error(err, "unable to fetch devList ..")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	for i, devItem := range devList.Items {

		msg = fmt.Sprintf(
			"\n\nDEBUG:__________(start %v)\n\n"+
				"devItem.Name: %v\n"+
				"devItem.Status.Active: %v\n"+
				"devItem: %v\n"+
				"\n\nDEBUG:__________(end %v)\n\n\n", i, devItem.Name, devItem.Status.Active, devItem, i)
		log.Info(msg)

	}

	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	// find the active list of jobs
	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time // find the last run so we can update the status

	isJobFinished := func(job *kbatch.Job) (bool, kbatch.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}

	getScheduledTimeForJob := func(job *kbatch.Job) (*time.Time, error) {
		timeRaw := job.Annotations[scheduledTimeAnnotation]
		if len(timeRaw) == 0 {
			return nil, nil
		}

		timeParsed, err := time.Parse(time.RFC3339, timeRaw)
		if err != nil {
			return nil, err
		}
		return &timeParsed, nil
	}

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

		// We'll store the launch time in an annotation, so we'll reconstitute that from
		// the active jobs themselves.
		scheduledTimeForJob, err := getScheduledTimeForJob(&job)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &job)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}

	if mostRecentTime != nil {
		dev.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		dev.Status.LastScheduleTime = nil
	}
	dev.Status.Active = nil

	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		dev.Status.Active = append(dev.Status.Active, *jobRef)
	}

	log.V(1).Info("info on dev:")
	msg = fmt.Sprintf("\n\ndev.Status.Active: %v\n"+
		"dev.Status.LastScheduleTime: %v\n\n",
		dev.Status.Active, dev.Status.LastScheduleTime)
	log.V(1).Info(msg)

	log.V(1).Info("job count", "active jobs", len(activeJobs),
		"successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	/*
		Using the date we've gathered, we'll update the status of our CRD.
		Just like before, we use our client.  To specifically update the status
		subresource, we'll use the `Status` part of the client, with the `Update`
		method.
		The status subresource ignores changes to spec, so it's less likely to conflict
		with any other updates, and can have separate permissions.
	*/

	//if err := r.Status().Update(ctx, &dev); err != nil {
	//	log.Error(err, "unable to update Dev status")
	//	return ctrl.Result{}, err
	//}

	///*
	//	Once we've updated our status, we can move on to ensuring that the status of
	//	the world matches what we want in our spec.
	//	### 3: Clean up old jobs according to the history limit
	//	First, we'll try to clean up old jobs, so that we don't leave too many lying
	//	around.
	//*/
	//
	//// NB: deleting these is "best effort" -- if we fail on a particular one,
	//// we won't requeue just to finish the deleting.
	//if dev.Spec.FailedJobsHistoryLimit != nil {
	//	sort.Slice(failedJobs, func(i, j int) bool {
	//		if failedJobs[i].Status.StartTime == nil {
	//			return failedJobs[j].Status.StartTime != nil
	//		}
	//		return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
	//	})
	//	for i, job := range failedJobs {
	//		if int32(i) >= int32(len(failedJobs))-*dev.Spec.FailedJobsHistoryLimit {
	//			break
	//		}
	//		if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
	//			log.Error(err, "unable to delete old failed job", "job", job)
	//		} else {
	//			log.V(0).Info("deleted old failed job", "job", job)
	//		}
	//	}
	//}
	//
	//if dev.Spec.SuccessfulJobsHistoryLimit != nil {
	//	sort.Slice(successfulJobs, func(i, j int) bool {
	//		if successfulJobs[i].Status.StartTime == nil {
	//			return successfulJobs[j].Status.StartTime != nil
	//		}
	//		return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
	//	})
	//	for i, job := range successfulJobs {
	//		if int32(i) >= int32(len(successfulJobs))-*dev.Spec.SuccessfulJobsHistoryLimit {
	//			break
	//		}
	//		if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); (err) != nil {
	//			log.Error(err, "unable to delete old successful job", "job", job)
	//		} else {
	//			log.V(0).Info("deleted old successful job", "job", job)
	//		}
	//	}
	//}
	//
	///* ### 4: Check if we're suspended
	//If this object is suspended, we don't want to run any jobs, so we'll stop now.
	//This is useful if something's broken with the job we're running and we want to
	//pause runs to investigate or putz with the cluster, without deleting the object.
	//*/
	//
	//if dev.Spec.Suspend != nil && *dev.Spec.Suspend {
	//	log.V(1).Info("cronjob suspended, skipping")
	//	return ctrl.Result{}, nil
	//}
	//
	///*
	//	Ref:
	//
	//	https://github.com/kubernetes-sigs/kubebuilder/blob/6ffec14d92c311dcfda7aca43934215ed4afafb8/docs/book/src/cronjob-tutorial/testdata/project/controllers/cronjob_controller.go
	//
	//
	//*/

	return ctrl.Result{}, nil
}

func (r *DevReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// set up a real clock, since we're not in a test
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(&kbatch.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		job := rawObj.(*kbatch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...make sure it's a CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "Dev" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Dev{}).
		Complete(r)
}
