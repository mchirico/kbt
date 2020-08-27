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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
)

func (r *DevReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("dev", req.NamespacedName)

	// your logic here
	var devList webappv1.DevList
	if err := r.List(ctx, &devList, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DevReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Dev{}).
		Complete(r)
}
