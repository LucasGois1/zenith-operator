# Changelog

## [0.2.15](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.14...zenith-operator-0.2.15) (2025-11-27)


### Bug Fixes

* improve LoadBalancer detection and DNS configuration in config-setup-job ([f00a99c](https://github.com/LucasGois1/zenith-operator/commit/f00a99c6eaf3e0d2d6cd758ff14564496c2f665f))

## [0.2.14](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.13...zenith-operator-0.2.14) (2025-11-27)


### Bug Fixes

* improve Envoy Gateway service detection and DNS configuration in config-setup-job ([8bd804d](https://github.com/LucasGois1/zenith-operator/commit/8bd804de610fea03d586d2da738d121b3c7cf929))

## [0.2.13](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.12...zenith-operator-0.2.13) (2025-11-27)


### Bug Fixes

* add missing Tekton Pipelines CRDs to Helm chart ([#82](https://github.com/LucasGois1/zenith-operator/issues/82)) ([6487950](https://github.com/LucasGois1/zenith-operator/commit/64879500c77576759d6add0268df67e5e9ce560c))

## [0.2.12](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.11...zenith-operator-0.2.12) (2025-11-27)


### Bug Fixes

* enable MetalLB and Dapr in values.yaml ([73033ba](https://github.com/LucasGois1/zenith-operator/commit/73033baa9b63c71d56657894ea1d046589446dd7))

## [0.2.11](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.10...zenith-operator-0.2.11) (2025-11-27)


### Bug Fixes

* remove test tag from operator image in values-dev.yaml ([2b2a519](https://github.com/LucasGois1/zenith-operator/commit/2b2a5191bd882da2bafae2a80b6377cec3709188))

## [0.2.10](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.9...zenith-operator-0.2.10) (2025-11-27)


### Bug Fixes

* add missing CRDs for Gateway API, Knative Serving, and Knative Eventing ([#72](https://github.com/LucasGois1/zenith-operator/issues/72)) ([2f1ab2c](https://github.com/LucasGois1/zenith-operator/commit/2f1ab2c7b98f6604346c5b1036cd4e5e294a5177))

## [0.2.9](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.8...zenith-operator-0.2.9) (2025-11-27)


### Bug Fixes

* helm chartinstallation ([#69](https://github.com/LucasGois1/zenith-operator/issues/69)) ([58917d5](https://github.com/LucasGois1/zenith-operator/commit/58917d5255536904a4225bfec68dd4ad82d29f2b))

## [0.2.8](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.7...zenith-operator-0.2.8) (2025-11-27)


### Bug Fixes

* helm chart crds ([#63](https://github.com/LucasGois1/zenith-operator/issues/63)) ([8ea9180](https://github.com/LucasGois1/zenith-operator/commit/8ea9180bda7317063c3290de34ca378bbb064622))

## [0.2.7](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.6...zenith-operator-0.2.7) (2025-11-27)


### Bug Fixes

* **helm:** add missing CRDs, KUBERNETES_MIN_VERSION, MetalLB auto-detection, and registry config ([#59](https://github.com/LucasGois1/zenith-operator/issues/59)) ([f51b682](https://github.com/LucasGois1/zenith-operator/commit/f51b68290a580b7406b1cf06a00ee28bee6aca63))

## [0.2.6](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.5...zenith-operator-0.2.6) (2025-11-27)


### Features

* add NodePort support for internal registry and fix image pull in Knative ([055259b](https://github.com/LucasGois1/zenith-operator/commit/055259bf1d4dfe6de13033b2721a35eed9cda0d5))

## [0.2.5](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.4...zenith-operator-0.2.5) (2025-11-27)


### Bug Fixes

* **tekton:** enable step-actions feature flag for buildpacks-phases Task ([#52](https://github.com/LucasGois1/zenith-operator/issues/52)) ([48eb54a](https://github.com/LucasGois1/zenith-operator/commit/48eb54a4b10ee67aa5a10743889a64fc7dd39dda))

## [0.2.4](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.3...zenith-operator-0.2.4) (2025-11-27)


### Bug Fixes

* **helm:** add missing RBAC permissions for Tekton tasks and taskruns ([#48](https://github.com/LucasGois1/zenith-operator/issues/48)) ([66b9f9b](https://github.com/LucasGois1/zenith-operator/commit/66b9f9bb9053edefb4cb89ac1a9e1ec84a7607b8))

## [0.2.3](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.2...zenith-operator-0.2.3) (2025-11-27)


### Bug Fixes

* reduce kubectl wait timeouts from 300s to 60s in post-install hooks ([#44](https://github.com/LucasGois1/zenith-operator/issues/44)) ([975cc40](https://github.com/LucasGois1/zenith-operator/commit/975cc40b83e981e321e090a8ea8d4d2c6165bacb))

## [0.2.2](https://github.com/LucasGois1/zenith-operator/compare/zenith-operator-0.2.1...zenith-operator-0.2.2) (2025-11-27)


### Bug Fixes

* tekton task dynamic creation ([#35](https://github.com/LucasGois1/zenith-operator/issues/35)) ([dedd38f](https://github.com/LucasGois1/zenith-operator/commit/dedd38f7c5c039b55d4e551b17b61f1849b9afd9))
