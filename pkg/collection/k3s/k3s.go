package k3s

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/rancherfederal/ocil/pkg/artifacts"
	"github.com/rancherfederal/ocil/pkg/artifacts/image"

	"github.com/rancherfederal/ocil/pkg/artifacts/file"

	"github.com/rancherfederal/ocil/pkg/artifacts/file/getter"

	"github.com/rancherfederal/hauler/pkg/reference"
)

var _ artifacts.OCICollection = (*k3s)(nil)

const (
	releaseUrl   = "https://github.com/k3s-io/k3s/releases/download"
	channelUrl   = "https://update.k3s.io/v1-release/channels"
	bootstrapUrl = "https://get.k3s.io"
)

var (
	ErrImagesNotFound     = errors.New("k3s dependent images not found")
	ErrFetchingImages     = errors.New("failed to fetch k3s dependent images")
	ErrExecutableNotfound = errors.New("k3s executable not found")
	ErrChannelNotFound    = errors.New("desired k3s channel not found")
)

type k3s struct {
	version string
	arch    string

	computed bool
	contents map[string]artifacts.OCI
	channels map[string]string
	client   *getter.Client
}

func NewK3s(version string) (artifacts.OCICollection, error) {
	return &k3s{
		version:  version,
		contents: make(map[string]artifacts.OCI),
	}, nil
}

func (k *k3s) Contents() (map[string]artifacts.OCI, error) {
	if err := k.compute(); err != nil {
		return nil, err
	}
	return k.contents, nil
}

func (k *k3s) compute() error {
	if k.computed {
		return nil
	}

	if err := k.fetchChannels(); err == nil {
		if version, ok := k.channels[k.version]; ok {
			k.version = version
		}
	}

	if err := k.images(); err != nil {
		return err
	}

	if err := k.executable(); err != nil {
		return err
	}

	if err := k.bootstrap(); err != nil {
		return err
	}

	k.computed = true
	return nil
}

func (k *k3s) executable() error {
	n := "k3s"
	if k.arch != "" && k.arch != "amd64" {
		n = fmt.Sprintf("name-%s", k.arch)
	}
	fref := k.releaseUrl(n)

	resp, err := http.Head(fref)
	if resp.StatusCode != http.StatusOK || err != nil {
		return ErrExecutableNotfound
	}

	f := file.NewFile(fref)

	ref := fmt.Sprintf("%s/k3s:%s", reference.DefaultNamespace, k.dnsCompliantVersion())
	k.contents[ref] = f
	return nil
}

func (k *k3s) bootstrap() error {
	c := getter.NewClient(getter.ClientOptions{NameOverride: "k3s-init.sh"})
	f := file.NewFile(bootstrapUrl, file.WithClient(c))

	ref := fmt.Sprintf("%s/k3s-init.sh:%s", reference.DefaultNamespace, reference.DefaultTag)
	k.contents[ref] = f
	return nil
}

func (k *k3s) images() error {
	resp, err := http.Get(k.releaseUrl("k3s-images.txt"))
	if resp.StatusCode != http.StatusOK {
		return ErrFetchingImages
	} else if err != nil {
		return ErrImagesNotFound
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		reference := scanner.Text()
		o, err := image.NewImage(reference)
		if err != nil {
			return err
		}

		k.contents[reference] = o
	}
	return nil
}

func (k *k3s) releaseUrl(artifact string) string {
	u, _ := url.Parse(releaseUrl)
	complete := []string{u.Path}
	u.Path = path.Join(append(complete, []string{k.version, artifact}...)...)
	return u.String()
}

func (k *k3s) dnsCompliantVersion() string {
	return strings.ReplaceAll(k.version, "+", "-")
}

func (k *k3s) fetchChannels() error {
	resp, err := http.Get(channelUrl)
	if err != nil {
		return err
	}

	var c channel
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return err
	}

	channels := make(map[string]string)
	for _, ch := range c.Data {
		channels[ch.Name] = ch.Latest
	}

	k.channels = channels
	return nil
}

type channel struct {
	Data []channelData `json:"data"`
}

type channelData struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Latest string `json:"latest"`
}
