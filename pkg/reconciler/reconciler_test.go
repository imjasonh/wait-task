package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	r := &v1alpha1.Run{
		ObjectMeta: metav1.ObjectMeta{
			Name: "run",
		},
		Spec: v1alpha1.RunSpec{
			Ref: &v1alpha1.TaskRef{
				APIVersion: "example.dev/v0",
				Kind:       "Wait",
			},
			Params: []v1beta1.Param{{
				Name:  "duration",
				Value: *v1beta1.NewArrayOrString("1s"),
			}},
		},
	}
	rec := &Reconciler{}
	// Mock the EnqueueAfter method to sleep, then call ReconcileKind
	// again. This will get called after ReconcileKind is called the first
	// time, below. After reconciling again, check that the status is as
	// expected.
	rec.EnqueueAfter = func(_ interface{}, d time.Duration) {
		time.Sleep(d)
		rec.ReconcileKind(ctx, r)

		if !r.IsSuccessful() {
			t.Errorf("Run was not successful after second reconcile: %v", r.Status.GetCondition(apis.ConditionSucceeded).Status)
		}

		// Reconciling into the final completed state should take <2s.
		dur := r.Status.CompletionTime.Time.Sub(r.Status.StartTime.Time)
		if dur > 2*time.Second {
			t.Fatalf("completion_time - start_time > 2s: %s", dur)
		}
	}

	// Start reconciling the Run.
	// This will not return until the second Reconcile is done.
	rec.ReconcileKind(ctx, r)
}

func TestReconcile_Failure(t *testing.T) {
	for _, c := range []struct {
		desc string
		r    *v1alpha1.Run
	}{{
		desc: "no params",
		r:    &v1alpha1.Run{},
	}, {
		desc: "no duration param",
		r: &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "not-duration",
					Value: *v1beta1.NewArrayOrString("blah"),
				}},
			},
		},
	}, {
		desc: "extra params",
		r: &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "not-duration",
					Value: *v1beta1.NewArrayOrString("blah"),
				}, {
					Name:  "duration",
					Value: *v1beta1.NewArrayOrString("1h"),
				}},
			},
		},
	}, {
		desc: "duration param not a string",
		r: &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "duration",
					Value: *v1beta1.NewArrayOrString("blah", "blah", "blah"),
				}},
			},
		},
	}, {
		desc: "invalid duration value",
		r: &v1alpha1.Run{
			Spec: v1alpha1.RunSpec{
				Params: []v1beta1.Param{{
					Name:  "duration",
					Value: *v1beta1.NewArrayOrString("blah"),
				}},
			},
		},
	}} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := context.Background()
			rec := &Reconciler{
				EnqueueAfter: func(interface{}, time.Duration) {
					t.Fatal("EnqueueAfter called")
				},
			}

			// Start reconciling the Run.
			// This will not return until the second Reconcile is done.
			rec.ReconcileKind(ctx, c.r)

			if !c.r.IsDone() {
				t.Fatal("Run was not done")
			} else if c.r.IsSuccessful() {
				t.Fatal("Run was successful")
			}
		})
	}
}
