package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VEXFormat names the wire format of an external feed.
type VEXFormat string

const (
	FormatOpenVEX VEXFormat = "OpenVEX"
	FormatCSAF    VEXFormat = "CSAF"
)

// VEXSourceSpec declares a single external VEX feed.
type VEXSourceSpec struct {
	// URL is the HTTP(S) endpoint serving the feed document or index.
	URL string `json:"url"`

	// Format selects the parser: OpenVEX or CSAF.
	Format VEXFormat `json:"format"`

	// ImageSelector scopes ingestion to images whose reference
	// matches this glob pattern. Empty means accept all images.
	// +optional
	ImageSelector string `json:"imageSelector,omitempty"`

	// RefreshInterval is how often the controller re-fetches the feed.
	// Defaults to 6h if unset.
	// +optional
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`
}

// VEXSourceStatus records the last sync result.
type VEXSourceStatus struct {
	// LastSyncTime is when the feed was last successfully fetched.
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// FeedHash is the SHA-256 of the last fetched body, used for
	// change detection without storing the full document.
	// +optional
	FeedHash string `json:"feedHash,omitempty"`

	// StatementCount is how many VEX statements were ingested from
	// the most recent fetch.
	// +optional
	StatementCount int `json:"statementCount,omitempty"`

	// Conditions follows the standard Kubernetes condition convention.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Format",type=string,JSONPath=".spec.format"
// +kubebuilder:printcolumn:name="LastSync",type=date,JSONPath=".status.lastSyncTime"
// +kubebuilder:printcolumn:name="Statements",type=integer,JSONPath=".status.statementCount"

// VEXSource declares an external VEX feed the controller should ingest.
type VEXSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VEXSourceSpec   `json:"spec"`
	Status            VEXSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VEXSourceList contains a list of VEXSource
type VEXSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VEXSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VEXSource{}, &VEXSourceList{})
}
