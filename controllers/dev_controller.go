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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	msg := fmt.Sprintf("dev.Name: %v dev.Status:%v", dev.Name, dev.Status)
	log.Info(msg)

	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	for i, job := range childJobs.Items {
		msg := fmt.Sprintf("i:%v job:%v", i, job)
		log.Info(msg)

	}

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
