# Changelog

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
