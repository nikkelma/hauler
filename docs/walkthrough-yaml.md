# Walkthrough - YAML-based

## Installation

The recommended version of hauler at this time is `v0.2.2-alpha1` due to some critical bugs in `v0.3.0`.
All examples below assume using version `v0.2.2-alpha1` - install this version at this repo's release page: <https://github.com/nikkelma/hauler/releases/tag/v0.2.2-alpha1>.
Download the right binary for your platform, ensure it's in your `PATH`, and all below commands will run as expected.

## Collecting charts and images using `Charts` and `Images` API objects

The most common use of `hauler` is to target a chart and associated container images for download and moving into an air gap.
Below are two methods for collecting the `cert-manager` chart and its images, as well as the required CRDs file.

### Step 1: create `Chart` + `Images` YAMLs

The simplest way to collect a chart and images is to define them separately, in two YAMLs.

Chart:

`cert-manager_v1.9.1_chart.yaml`
```yaml
apiVersion: content.hauler.cattle.io/v1alpha2
kind: Charts
metadata:
  name: cert-manager-v1-9-1
spec:
  charts:
    - repoURL: "https://charts.jetstack.io"
      name: "cert-manager"
      version: "v1.9.1"
```

Images:

`cert-manager_v1.9.1_images.yaml`
```yaml
apiVersion: content.hauler.cattle.io/v1alpha2
kind: Images
metadata:
  name: cert-manager-v1-9-1
spec:
  images:
    - name: quay.io/jetstack/cert-manager-controller:v1.9.1
    - name: quay.io/jetstack/cert-manager-webhook:v1.9.1
    - name: quay.io/jetstack/cert-manager-cainjector:v1.9.1
    - name: quay.io/jetstack/cert-manager-ctl:v1.9.1
```

For a complete example to get `cert-manager` fully installed, we also need to fetch a file:

File:

`cert-manager_v1.9.1_files.yaml`
```yaml
apiVersion: content.hauler.cattle.io/v1alpha2
kind: Files
metadata:
  name: cert-manager-v1-9-1-crds
spec:
  files:
    - path: https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.crds.yaml
```

### Step 2: sync stores using YAMLs

Now that we have our hauler YAML definitions, we can use the hauler CLI to sync the dependencies.

> **NOTE:** In this example, the choice has been made to create two packages: one for container images, and one for files / artifacts.
> This allows for easier copying of the container images into an already-existing registry.

```shell
hauler store sync --store cm-1-9-1-images \
  -f cert-manager_v1.9.1_images.yaml

hauler store sync --store cm-1-9-1-artifacts \
  -f cert-manager_v1.9.1_files.yaml \
  -f cert-manager_v1.9.1_chart.yaml
```

### Step 3: create packages for moving into the air gap

We've downloaded the required files and containers locally, now let's prep the packages for moving into the air gap.

```shell
hauler store save --store cm-1-9-1-images \
  -f cm-1-9-1-images.tar.zst

hauler store save --store cm-1-9-1-artifacts \
  -f cm-1-9-1-artifacts.tar.zst
```

### Step 4: move `hauler` binary and packages into the air gap

This process will be different per environment, but the `hauler` binary matching the target host's OS/architecture needs to be moved into the air gap.
Similarly, the `cm-1-9-1-images.tar.zst` and `cm-1-9-1-artifacts.tar.zst` package files also need to be moved into the air gap.

### Step 5: load packages into hauler in air gap

Although we have a compressed package inside the air gap now, hauler needs to decompress and restructure the package into a format it can use for further actions.

> **NOTE** the lack of `-f` in these commands!
> UX improvements are on their way.

```shell
hauler store load --store cm-1-9-1-images \
  cm-1-9-1-images.tar.zst
hauler store load --store cm-1-9-1-artifacts \
  cm-1-9-1-artifacts.tar.zst
```

### Step 6: extract, copy, or serve stores

Hauler can now extract, copy, or serve the contents of these stores.

```shell
hauler store copy --store cm-1-9-1-images my-reg.example.org

mkdir -p ./artifacts
hauler store extract --store cm-1-9-1-artifacts ./artifacts
```

## Collecting charts and images using `ThickCharts` API objects

Instead of manually specifying a chart and its required images, the `ThickChart` API object can automatically inspect a chart and pull any specified images out of it.

> **IMPORTANT NOTE:** due to current limited functionality of `hauler`, a `ThickChart` will collect helm charts and container images into the same store.
> This means copying images into a destination registry will also attempt to copy the chart into the destination registry, which may not be desired.
> Work is underway to allow finer-grained copies as well as separating a `ThickChart` into groups of similar artifact types for easier copying / serving.

Due to the above note, this example will combine all artifacts into the same store.


### Step 1: create `ThickChart` YAMLs

In case some images are not directly used by the chart, but are required at runtime, the `extraImages` field can add in any missing images.

ThickChart:

`cert-manager_v1.9.1_thickchart.yaml`
```yaml
apiVersion: collection.hauler.cattle.io/v1alpha2
kind: ThickCharts
metadata:
  name: cert-manager-v1-9-1
spec:
  charts:
    - repoURL: "https://charts.jetstack.io"
      name: "cert-manager"
      version: "v1.9.1"
      extraImages:
        - ref: quay.io/jetstack/cert-manager-ctl:v1.9.1
```

For a complete example to get `cert-manager` fully installed, we also need to fetch a file:

File:

`cert-manager_v1.9.1_files.yaml`
```yaml
apiVersion: content.hauler.cattle.io/v1alpha2
kind: Files
metadata:
  name: cert-manager-v1-9-1-crds
spec:
  files:
    - path: https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.crds.yaml
```

### Step 2: sync stores using YAMLs

Now that we have our hauler YAML definitions, we can use the hauler CLI to sync the dependencies.

> **NOTE:** In this example, the choice has been made to create a single package, due to the reasoning provided in "_Collecting charts and images using `ThickCharts` API objects_"

```shell
hauler store sync --store cm-1-9-1 \
  -f cert-manager_v1.9.1_thickchart.yaml \
  -f cert-manager_v1.9.1_files.yaml
```

### Step 3: create packages for moving into the air gap

We've downloaded the required files and containers locally, now let's prep the packages for moving into the air gap.

```shell
hauler store save --store cm-1-9-1 \
  -f cm-1-9-1.tar.zst
```

### Step 4: move `hauler` binary and packages into the air gap

This process will be different per environment, but the `hauler` binary matching the target host's OS/architecture needs to be moved into the air gap.
Similarly, the `cm-1-9-1.tar.zst` package file also needs to be moved into the air gap.

### Step 5: load packages into hauler in air gap

Although we have a compressed package inside the air gap now, hauler needs to decompress and restructure the package into a format it can use for further actions.

> **NOTE** the lack of `-f` in these commands!
> UX improvements are on their way.

```shell
hauler store load --store cm-1-9-1 \
  cm-1-9-1.tar.zst
```

### Step 6: extract, copy, or serve store

Hauler can now extract, copy, or serve the contents of this store.

```shell
hauler store copy --store cm-1-9-1 my-reg.example.org

mkdir -p ./artifacts
hauler store extract --store cm-1-9-1 ./artifacts
```
