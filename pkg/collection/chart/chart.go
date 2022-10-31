package chart

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	gname "github.com/google/go-containerregistry/pkg/name"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/nikkelma/hauler/pkg/artifact"
	"github.com/nikkelma/hauler/pkg/artifact/local"
	"github.com/nikkelma/hauler/pkg/content/chart"
	"github.com/nikkelma/hauler/pkg/content/image"
)

var _ artifact.Collection = (*tchart)(nil)

// tchart is a thick chart that includes all the dependent images as well as the chart itself
type tchart struct {
	name         string
	repo         string
	version      string
	chart        *chart.Chart
	valuesConfig tchartValuesConfig

	computed bool
	contents map[gname.Reference]artifact.OCI
}

type tchartValuesConfig struct {
	disableDefaults     bool
	valuesOverrides     []map[string]interface{}
	valuesFileOverrides []string
}

func NewChart(
	name,
	repo,
	version string,
	disableDefaults bool,
	valuesOverrides []map[string]interface{},
	valuesFileOverrides []string,
) (artifact.Collection, error) {
	o, err := chart.NewChart(name, repo, version)
	if err != nil {
		return nil, err
	}

	return &tchart{
		name:    name,
		repo:    repo,
		version: version,
		chart:   o,
		valuesConfig: tchartValuesConfig{
			disableDefaults:     disableDefaults,
			valuesOverrides:     valuesOverrides,
			valuesFileOverrides: valuesFileOverrides,
		},
		contents: make(map[gname.Reference]artifact.OCI),
	}, nil
}

func (c *tchart) Contents() (map[gname.Reference]artifact.OCI, error) {
	if err := c.compute(); err != nil {
		return nil, err
	}
	return c.contents, nil
}

func (c *tchart) compute() error {
	if c.computed {
		return nil
	}

	if err := c.dependentImages(); err != nil {
		return err
	}

	if err := c.chartContents(); err != nil {
		return err
	}

	c.computed = true
	return nil
}

func (c *tchart) chartContents() error {
	oci, err := chart.NewChart(c.name, c.repo, c.version)
	if err != nil {
		return err
	}

	tag := c.version
	if tag == "" {
		tag = gname.DefaultTag
	}

	ref, err := gname.ParseReference(c.name, gname.WithDefaultRegistry(""), gname.WithDefaultTag(tag))
	if err != nil {
		return err
	}

	c.contents[ref] = oci
	return nil
}

func (c *tchart) dependentImages() error {
	ch, err := c.chart.Load()
	if err != nil {
		return err
	}

	var allOverrides []map[string]interface{}

	// if default values should be used, first render using empty values
	if !c.valuesConfig.disableDefaults {
		allOverrides = append(allOverrides, nil)
	}
	allOverrides = append(allOverrides, c.valuesConfig.valuesOverrides...)

	for _, valuesFile := range c.valuesConfig.valuesFileOverrides {
		if err := func() error {
			var getter local.Opener
			if strings.HasPrefix(valuesFile, "http") || strings.HasPrefix(valuesFile, "https") {
				getter = remoteOpener(valuesFile)
			} else {
				getter = localOpener(valuesFile)
			}

			reader, err := getter()
			if err != nil {
				return fmt.Errorf("get values file %s: %v", valuesFile, err)
			}
			defer reader.Close()

			values := make(map[string]interface{})

			decoder := yaml.NewYAMLToJSONDecoder(reader)
			if err := decoder.Decode(&values); err != nil {
				return fmt.Errorf("could not parse values from file %s: %v", valuesFile, err)
			}

			allOverrides = append(allOverrides, values)

			return nil
		}(); err != nil {
			return err
		}
	}

	var allImageRefs []string
	seenImageRefs := map[string]bool{}

	for _, values := range allOverrides {
		valuesOpt := WithValues(values)
		imgs, err := ImagesInChart(ch, valuesOpt)
		if err != nil {
			return err
		}

		for _, img := range imgs.Spec.Images {
			if !seenImageRefs[img.Ref] {
				allImageRefs = append(allImageRefs, img.Ref)
				seenImageRefs[img.Ref] = true
			}
		}
	}

	for _, imgRef := range allImageRefs {
		ref, err := gname.ParseReference(imgRef)
		if err != nil {
			return err
		}

		i, err := image.NewImage(imgRef)
		if err != nil {
			return err
		}
		c.contents[ref] = i
	}

	return nil
}

func localOpener(path string) local.Opener {
	return func() (io.ReadCloser, error) {
		return os.Open(path)
	}
}

func remoteOpener(url string) local.Opener {
	return func() (io.ReadCloser, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
}
