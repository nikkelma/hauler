package chart

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/rancher/wrangler/pkg/yaml"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/jsonpath"

	"github.com/nikkelma/hauler/pkg/apis/hauler.cattle.io/v1alpha1"
)

var defaultKnownImagePaths = []string{
	// Deployments & DaemonSets
	"{.spec.template.spec.initContainers[*].image}",
	"{.spec.template.spec.containers[*].image}",

	// Pods
	"{.spec.initContainers[*].image}",
	"{.spec.containers[*].image}",
}

type ImagesInChartOption interface {
	Apply(options *imagesInChartOptions)
}

type imagesInChartOptions struct {
	Values map[string]interface{}
}

func WithValues(values map[string]interface{}) ImagesInChartOption {
	return withValues(values)
}

type withValues map[string]interface{}

func (o withValues) Apply(options *imagesInChartOptions) {
	options.Values = o
}

// ImagesInChart will render a chart and identify all dependent images from it
func ImagesInChart(c *helmchart.Chart, opts ...ImagesInChartOption) (v1alpha1.Images, error) {
	opt := &imagesInChartOptions{}
	for _, o := range opts {
		o.Apply(opt)
	}

	objs, err := template(c, opt.Values)

	if err != nil {
		return v1alpha1.Images{}, err
	}

	var imageRefs []string
	for _, o := range objs {
		d, err := o.(*unstructured.Unstructured).MarshalJSON()
		if err != nil {
			// TODO: Should we actually capture these errors?
			continue
		}

		var obj interface{}
		if err := json.Unmarshal(d, &obj); err != nil {
			continue
		}

		j := jsonpath.New("")
		j.AllowMissingKeys(true)

		for _, p := range defaultKnownImagePaths {
			r, err := parseJSONPath(obj, j, p)
			if err != nil {
				continue
			}

			imageRefs = append(imageRefs, r...)
		}
	}

	ims := v1alpha1.Images{
		Spec: v1alpha1.ImageSpec{
			Images: []v1alpha1.Image{},
		},
	}

	seenRefs := map[string]bool{}

	for _, ref := range imageRefs {
		if !seenRefs[ref] {
			ims.Spec.Images = append(ims.Spec.Images, v1alpha1.Image{Ref: ref})
			seenRefs[ref] = true
		}
	}
	return ims, nil
}

func template(c *helmchart.Chart, values map[string]interface{}) ([]runtime.Object, error) {
	s := storage.Init(driver.NewMemory())

	templateCfg := &action.Configuration{
		RESTClientGetter: nil,
		Releases:         s,
		KubeClient:       &fake.PrintingKubeClient{Out: io.Discard},
		Capabilities:     chartutil.DefaultCapabilities,
		Log:              func(format string, v ...interface{}) {},
	}

	client := action.NewInstall(templateCfg)
	client.ReleaseName = "dry"
	client.DryRun = true
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	release, err := client.Run(c, values)
	if err != nil {
		return nil, err
	}

	return yaml.ToObjects(bytes.NewBufferString(release.Manifest))
}

func parseJSONPath(data interface{}, parser *jsonpath.JSONPath, template string) ([]string, error) {
	buf := new(bytes.Buffer)
	if err := parser.Parse(template); err != nil {
		return nil, err
	}

	if err := parser.Execute(buf, data); err != nil {
		return nil, err
	}

	f := func(s rune) bool { return s == ' ' }
	r := strings.FieldsFunc(buf.String(), f)
	return r, nil
}
