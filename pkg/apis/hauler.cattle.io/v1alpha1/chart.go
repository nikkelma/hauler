package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ChartsContentKind    = "Charts"
	ChartsCollectionKind = "ThickCharts"
)

type Charts struct {
	*metav1.TypeMeta  `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ChartSpec `json:"spec,omitempty"`
}

type ChartSpec struct {
	Charts []Chart `json:"charts,omitempty"`
}

type Chart struct {
	Name    string `json:"name"`
	RepoURL string `json:"repoURL"`
	Version string `json:"version"`
}

type ThickCharts struct {
	*metav1.TypeMeta  `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ThickChartSpec `json:"spec,omitempty"`
}

type ThickChartSpec struct {
	Charts []ThickChart `json:"charts,omitempty"`
}

type ThickChart struct {
	Name           string                 `json:"name"`
	RepoURL        string                 `json:"repoURL"`
	Version        string                 `json:"version"`
	ValuesConfig   ThickChartValuesConfig `json:"valuesConfig,omitempty"`
	ExtraJSONPaths []string               `json:"extraJSONPaths,omitempty"`
}

type ThickChartValuesConfig struct {
	DisableDefault bool                      `json:"disableDefault,omitempty"`
	Overrides      []ThickChartValueOverride `json:"overrides,omitempty"`
}

type ThickChartValueOverride struct {
	Values     map[string]interface{} `json:"values,omitempty"`
	ValuesFile string                 `json:"valuesFile,omitempty"`
}
