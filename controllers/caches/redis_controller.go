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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cachesv1 "operator/caches/apis/caches/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=caches.wise2c.com,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=caches.wise2c.com,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;list;watch;create;update;patch;delete

func (r *RedisReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("redis", req.NamespacedName)
	redis := &cachesv1.Redis{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, redis)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("redis resource not found, ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		r.Log.Error(err, "failed to get redis")
		return reconcile.Result{}, err
	}
	found := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: redis.Name, Namespace: redis.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.DeploymentForRedis(redis)
		r.Log.Info("creating a new deployment", "name", dep.Name, "in", dep.Namespace)
		err = r.Client.Create(context.TODO(), dep)
		if err != nil {
			r.Log.Error(err, "failed to create new deployment", "name", dep.Name)
			return reconcile.Result{}, err
		}
	}
	// your logic here

	return ctrl.Result{}, nil
}

func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachesv1.Redis{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *RedisReconciler) DeploymentForRedis(app *cachesv1.Redis) *appsv1.Deployment {
	ls := labels(app.Name)
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

func labels(name string) map[string]string {
	return map[string]string{"app": "redis", "cr": name}
}
