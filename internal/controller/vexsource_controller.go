package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	securityv1alpha1 "github.com/AdeshDeshmukh/vexbridge/api/v1alpha1"
	"github.com/AdeshDeshmukh/vexbridge/internal/fetcher"
	"github.com/AdeshDeshmukh/vexbridge/internal/store"
)

const (
	conditionSynced  = "Synced"
	defaultRefresh   = 6 * time.Hour
	requeueAfterSync = 30 * time.Second
)

// VEXSourceReconciler reconciles VEXSource objects.
type VEXSourceReconciler struct {
	client.Client
	Store    *store.VEXStore
	Fetchers map[securityv1alpha1.VEXFormat]fetcher.Fetcher
}

// +kubebuilder:rbac:groups=security.vexbridge.io,resources=vexsources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.vexbridge.io,resources=vexsources/status,verbs=get;update;patch

func (r *VEXSourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var source securityv1alpha1.VEXSource
	if err := r.Get(ctx, req.NamespacedName, &source); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	f, ok := r.Fetchers[source.Spec.Format]
	if !ok {
		return r.setFailed(ctx, &source, fmt.Errorf("unsupported format: %s", source.Spec.Format))
	}

	stmts, digest, err := f.Fetch(ctx, source.Spec.URL)
	if err != nil {
		return r.setFailed(ctx, &source, err)
	}

	if digest != "" && digest != source.Status.FeedHash {
		r.Store.Upsert(stmts)
		logger.Info("feed updated", "source", req.NamespacedName, "statements", len(stmts))
	}

	refresh := defaultRefresh
	if source.Spec.RefreshInterval != nil {
		refresh = source.Spec.RefreshInterval.Duration
	}

	now := metav1.Now()
	source.Status.LastSyncTime = &now
	source.Status.FeedHash = digest
	source.Status.StatementCount = len(stmts)
	meta.SetStatusCondition(&source.Status.Conditions, metav1.Condition{
		Type:               conditionSynced,
		Status:             metav1.ConditionTrue,
		Reason:             "FetchSucceeded",
		Message:            fmt.Sprintf("ingested %d statements", len(stmts)),
		LastTransitionTime: now,
	})

	if err := r.Status().Update(ctx, &source); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: refresh}, nil
}

func (r *VEXSourceReconciler) setFailed(
	ctx context.Context,
	source *securityv1alpha1.VEXSource,
	fetchErr error,
) (ctrl.Result, error) {
	now := metav1.Now()
	meta.SetStatusCondition(&source.Status.Conditions, metav1.Condition{
		Type:               conditionSynced,
		Status:             metav1.ConditionFalse,
		Reason:             "FetchFailed",
		Message:            fetchErr.Error(),
		LastTransitionTime: now,
	})
	_ = r.Status().Update(ctx, source)
	return ctrl.Result{RequeueAfter: requeueAfterSync}, nil
}

func (r *VEXSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&securityv1alpha1.VEXSource{}).
		Complete(r)
}
