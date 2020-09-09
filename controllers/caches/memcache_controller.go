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

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachesv1 "operator/caches/apis/caches/v1"
)

// MemcacheReconciler reconciles a Memcache object
type MemcacheReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=caches.wise2c.com,resources=memcaches,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=caches.wise2c.com,resources=memcaches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;list;watch;create;update;patch;delete

func (r *MemcacheReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("memcache", req.NamespacedName)
	memcache := &cachesv1.Memcache{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, memcache)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("memcache resource not found, ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "failed to get redis")
		return reconcile.Result{}, err
	}
	found := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: memcache.Name, Namespace: memcache.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.DeploymentForMemcache(memcache)
		r.Log.Info("creating a new deployment", "Deployment.Name", dep.Name, "Deployment.Namespace", dep.Namespace)
		err = r.Client.Create(context.TODO(), dep)
		if err != nil {
			r.Log.Error(err, "failed to create new deployment", "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
	}

	// your logic here

	return ctrl.Result{}, nil
}

func (r *MemcacheReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachesv1.Memcache{}).
		Complete(r)
}
func (r *MemcacheReconciler) DeploymentForMemcache(app *cachesv1.Memcache) *appsv1.Deployment {
	ls := labelsForMemcache(app.Name)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: app.Spec.Image,
						Name:  app.Name,
					}},
				},
			},
		},
	}
	controllerutil.SetControllerReference(app, dep, r.Scheme)
	return dep
}

func labelsForMemcache(name string) map[string]string {
	return map[string]string{"app": "memcache", "cr": name}
}
