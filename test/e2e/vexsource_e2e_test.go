//go:build e2e

package e2e_test

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	securityv1alpha1 "github.com/AdeshDeshmukh/vexbridge/api/v1alpha1"
)

func TestVEXSource_RedHatFeedE2E(t *testing.T) {
	cfg, err := config.GetConfig()
	if err != nil {
		t.Fatalf("getting kubeconfig: %v", err)
	}

	c, err := client.New(cfg, client.Options{})
	if err != nil {
		t.Fatalf("creating client: %v", err)
	}

	source := &securityv1alpha1.VEXSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-redhat-vex",
			Namespace: "default",
		},
		Spec: securityv1alpha1.VEXSourceSpec{
			URL:    "https://access.redhat.com/security/data/csaf/v2/vex/",
			Format: securityv1alpha1.FormatCSAF,
		},
	}

	ctx := context.Background()
	if err := c.Create(ctx, source); err != nil {
		t.Fatalf("creating VEXSource: %v", err)
	}
	t.Cleanup(func() { _ = c.Delete(ctx, source) })

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		var got securityv1alpha1.VEXSource
		if err := c.Get(ctx, types.NamespacedName{
			Name:      source.Name,
			Namespace: source.Namespace,
		}, &got); err != nil {
			t.Logf("polling: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if got.Status.StatementCount > 0 {
			t.Logf("E2E passed: ingested %d statements, feedHash=%s",
				got.Status.StatementCount, got.Status.FeedHash)
			return
		}
		time.Sleep(5 * time.Second)
	}
	t.Fatal("timed out waiting for VEXSource to sync")
}
