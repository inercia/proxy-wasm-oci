# Proxy-WASM for OCI

## Description

The Proxy-WASM for OCI is a set of utilities for packaging, publishing and using
Proxy-WASM extensions using OCI images and registries.

The main utiulity, `pwo`, can be used for:

* packaging and publishing a Proxy-WASM to an OCI image and registry.
  ```console
  $ pwo publish main.wasm oci://myregistry.com/myrepo/myimage:mytag
  ```
* downloading a Proxy-WASM from an OCI image and registry.
  ```console
  $ pwo download oci://myregistry.com/myrepo/myimage:mytag
  ```
* serving a Proxy-WASM from an OCI image and registry to Envoy in a HTTP port
  ```console
  $ pwo serve --port 15111
  ```
  so Envoy can download the Proxy-WASM from `http://localhost:15111`.

## Acknoledgements

Many parts of the OCI downloader were adapted from the Helm code. This is the
original Copyright notice:

```
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```