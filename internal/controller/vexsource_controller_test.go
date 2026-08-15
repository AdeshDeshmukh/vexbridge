package controller_test

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	securityv1alpha1 "github.com/AdeshDeshmukh/vexbridge/api/v1alpha1"
	"github.com/AdeshDeshmukh/vexbridge/internal/controller"
	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
	"github.com/AdeshDeshmukh/vexbridge/internal/store"
)

type dummyFetcher struct {
	stmts  []fetcher.Statement
	digest string
	err    error
}

func (d *dummyFetcher) Fetch(ctx context.Context, url string) ([]fetcher.Statement, string, error) {
	return d.stmts, d.digest, d.err
}

func TestVEXSourceReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = securityv1alpha1.AddToScheme(scheme)

	source := &securityv1alpha1.VEXSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-source",
			Namespace: "default",
		},
		Spec: securityv1alpha1.VEXSourceSpec{
			URL:    "https://example.com/vex.json",
			Format: securityv1alpha1.FormatOpenVEX,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(source).
		WithStatusSubresource(source).
		Build()

	st := store.New()
	mockFetcher := &dummyFetcher{
		stmts: []fetcher.Statement{
			{
				VulnID:   "CVE-2024-0001",
				Status:   "not_affected",
				Products: []string{"nginx:latest"},
			},
		},
		digest: "hash123",
	}

	r := &controller.VEXSourceReconciler{
		Client: fakeClient,
		Store:  st,
		Fetchers: map[securityv1alpha1.VEXFormat]fetcher.Fetcher{
			securityv1alpha1.FormatOpenVEX: mockFetcher,
		},
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-source",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("unexpected reconcile error: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Error("expected non-zero RequeueAfter")
	}

	var updated securityv1alpha1.VEXSource
	if err := fakeClient.Get(ctx, req.NamespacedName, &updated); err != nil {
		t.Fatalf("failed to fetch updated VEXSource: %v", err)
	}

	if updated.Status.StatementCount != 1 {
		t.Errorf("got StatementCount %d, want 1", updated.Status.StatementCount)
	}
	if updated.Status.FeedHash != "hash123" {
		t.Errorf("got FeedHash %s, want hash123", updated.Status.FeedHash)
	}

	stmt, ok := st.Lookup("CVE-2024-0001", "nginx:latest")
	if !ok {
		t.Error("expected statement to be upserted into store")
	}
	if stmt.Status != "not_affected" {
		t.Errorf("got statement status %s, want not_affected", stmt.Status)
	}
}

func TestVEXSourceReconciler_FetchFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = securityv1alpha1.AddToScheme(scheme)

	source := &securityv1alpha1.VEXSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-source-fail",
			Namespace: "default",
		},
		Spec: securityv1alpha1.VEXSourceSpec{
			URL:    "https://example.com/bad-vex.json",
			Format: securityv1alpha1.FormatCSAF,
		},
	}

	fakeClient := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(source).
		WithStatusSubresource(source).
		Build()

	st := store.New()
	mockFetcher := &dummyFetcher{
		err: errors.New("network timeout"),
	}

	r := &controller.VEXSourceReconciler{
		Client: fakeClient,
		Store:  st,
		Fetchers: map[securityv1alpha1.VEXFormat]fetcher.Fetcher{
			securityv1alpha1.FormatCSAF: mockFetcher,
		},
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-source-fail",
			Namespace: "default",
		},
	}

	_, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("reconcile should handle fetch errors gracefully, got: %v", err)
	}

	var updated securityv1alpha1.VEXSource
	_ = fakeClient.Get(ctx, req.NamespacedName, &updated)
	if len(updated.Status.Conditions) == 0 || updated.Status.Conditions[0].Status != metav1.ConditionFalse {
		t.Error("expected Synced condition to be False on fetch failure")
	}
}
