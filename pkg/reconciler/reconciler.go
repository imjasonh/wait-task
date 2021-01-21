package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	kreconciler "knative.dev/pkg/reconciler"
)

type Reconciler struct {
	EnqueueAfter func(interface{}, time.Duration)
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, r *v1alpha1.Run) kreconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling %s/%s", r.Namespace, r.Name)

	// Ignore completed waits.
	if r.IsDone() {
		logger.Info("Run is finished, done reconciling")
		return nil
	}

	if r.Spec.Ref == nil ||
		r.Spec.Ref.APIVersion != "example.dev/v0" || r.Spec.Ref.Kind != "Wait" {
		// This is not a Run we should have been notified about; do nothing.
		return nil
	}
	if r.Spec.Ref.Name != "" {
		r.Status.Status.SetConditions([]apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "UnexpectedName",
			Message: fmt.Sprintf("Found unexpected ref name: %s", r.Spec.Ref.Name),
		}})
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
	if len(r.Spec.Params) != 1 {
		var found []string
		for _, p := range r.Spec.Params {
			if p.Name == "duration" {
				continue
			}
			found = append(found, p.Name)
		}
		r.Status.Status.SetConditions([]apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "UnexpectedParams",
			Message: fmt.Sprintf("Found unexpected params: %v", found),
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
		c.EnqueueAfter(r, time.Until(r.Status.StartTime.Time.Add(dur)))
	}

	return kreconciler.NewEvent(corev1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", r.Namespace, r.Name)
}
