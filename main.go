package main

import (
	"context"
	"fmt"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const controllerName = "wait-task-controller"

func main() {
	sharedmain.Main(controllerName, newController)
}

func newController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	c := &Reconciler{}
	impl := runreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName: controllerName,
		}
	})
	c.enqueueAfter = impl.EnqueueAfter

	runinformer.Get(ctx).Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: FilterRunRef("example.dev/v0", "Wait"),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	return impl
}

// FilterRunRef returns a filter that can be passed to a Run Informer, which
// filters out Runs for apiVersion and kinds that this controller doesn't care
// about.
// TODO: Provide this as a helper function.
func FilterRunRef(apiVersion, kind string) func(interface{}) bool {
	return func(obj interface{}) bool {
		r, ok := obj.(*v1alpha1.Run)
		if !ok {
			// Somehow got informed of a non-Run object.
			// Ignore.
			return false
		}
		if r == nil || r.Spec.Ref == nil {
			// These are invalid, but just in case they get
			// created somehow, don't panic.
			return false
		}

		return r.Spec.Ref.APIVersion == apiVersion && r.Spec.Ref.Kind == v1alpha1.TaskKind(kind)
	}
}

type Reconciler struct {
	enqueueAfter func(interface{}, time.Duration)
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, r *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling %s/%s", r.Namespace, r.Name)

	// Ignore completed waits.
	if r.IsDone() {
		logger.Info("Run is finished, done reconciling")
		return nil
	}

	expr := r.Spec.GetParam("duration")
	if expr == nil || expr.Value.StringVal == "" {
		r.Status.Status.SetConditions([]apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "MissingDuration",
			Message: "The duration param was not passed",
		}})
		return nil
	}
	dur, err := time.ParseDuration(expr.Value.StringVal)
	if err != nil {
		r.Status.Status.SetConditions([]apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "InvalidDuration",
			Message: fmt.Sprintf("The duration param was invalid: %v", err),
		}})
		return nil
	}

	if r.Status.StartTime == nil {
		now := metav1.Now()
		r.Status.StartTime = &now
		r.Status.Status.SetConditions([]apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  "Waiting",
			Message: "Waiting for duration to elapse",
		}})
	}

	done := r.Status.StartTime.Time.Add(dur)

	if time.Now().After(done) {
		now := metav1.Now()
		r.Status.CompletionTime = &now
		r.Status.Status.SetConditions([]apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionTrue,
			Reason:  "DurationElapsed",
			Message: "The wait duration has elapsed",
		}})
	} else {
		// Enqueue another check when the timeout should be elapsed.
		c.enqueueAfter(r, time.Until(r.Status.StartTime.Time.Add(dur)))
	}

	return reconciler.NewEvent(corev1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", r.Namespace, r.Name)
}
