package v1alpha1_test

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/AdeshDeshmukh/vexbridge/api/v1alpha1"
)

func TestVEXSource_SchemeRegistration(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add v1alpha1 to scheme: %v", err)
	}

	gv := v1alpha1.GroupVersion
	if gv.Group != "security.vexbridge.io" || gv.Version != "v1alpha1" {
		t.Errorf("unexpected GroupVersion: %s", gv.String())
	}

	knownTypes := scheme.KnownTypes(gv)
	if _, ok := knownTypes["VEXSource"]; !ok {
		t.Error("VEXSource type not registered in scheme")
	}
	if _, ok := knownTypes["VEXSourceList"]; !ok {
		t.Error("VEXSourceList type not registered in scheme")
	}
}

func TestVEXSource_DeepCopy(t *testing.T) {
	now := metav1.Now()
	dur := metav1.Duration{Duration: 6 * time.Hour}

	orig := &v1alpha1.VEXSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vexsource",
			Namespace: "vexbridge-system",
		},
		Spec: v1alpha1.VEXSourceSpec{
			URL:             "https://example.com/vex.json",
			Format:          v1alpha1.FormatOpenVEX,
			ImageSelector:   "nginx:*",
			RefreshInterval: &dur,
		},
		Status: v1alpha1.VEXSourceStatus{
			LastSyncTime:   &now,
			FeedHash:       "a1b2c3d4",
			StatementCount: 42,
			Conditions: []metav1.Condition{
				{
					Type:    "Synced",
					Status:  metav1.ConditionTrue,
					Reason:  "FetchSucceeded",
					Message: "Ingested 42 statements",
				},
			},
		},
	}

	cp := orig.DeepCopy()
	if cp == orig {
		t.Fatal("DeepCopy returned pointer to original struct")
	}

	if cp.Name != orig.Name || cp.Spec.URL != orig.Spec.URL {
		t.Errorf("mismatch in deepcopied fields: got %v, want %v", cp, orig)
	}

	// Verify deep copy mutation independence
	cp.Spec.URL = "https://mutated.com/vex.json"
	if orig.Spec.URL == cp.Spec.URL {
		t.Error("mutating copy affected original spec URL")
	}

	// Verify runtime.Object interface
	var obj runtime.Object = orig
	objCp := obj.DeepCopyObject()
	if objCp == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
}
