/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FunctionSpec defines the desired state of Function.
type FunctionSpec struct {
	// O URL do repositório Git contendo o código-fonte da função.
	// +kubebuilder:validation:Required
	GitRepo string `json:"gitRepo"`

	// Opcional. A revisão Git (branch, tag, ou hash) a ser usada.
	// Padrão: 'main' se não especificado.
	// +kubebuilder:validation:Optional
	GitRevision string `json:"gitRevision,omitempty"`

	// Opcional. O nome do Secret usado para autenticar com o repositório Git privado.
	// +kubebuilder:validation:Optional
	GitAuthSecretName string `json:"gitAuthSecretName,omitempty"`

	// Configurações de Build (Tekton)
	// +kubebuilder:validation:Required
	Build BuildSpec `json:"build"`

	// Configurações de Deploy (Knative + Dapr)
	// +kubebuilder:validation:Required
	Deploy DeploySpec `json:"deploy"`

	// Opcional. Configurações de Eventing (Knative Eventing)
	// +kubebuilder:validation:Optional
	Eventing EventingSpec `json:"eventing,omitempty"`

	// Opcional. Configurações de Observabilidade (OpenTelemetry)
	// +kubebuilder:validation:Optional
	Observability ObservabilitySpec `json:"observability,omitempty"`
}

// BuildSpec define os parâmetros para o pipeline de build
type BuildSpec struct {
	// O nome do Secret do tipo 'kubernetes.io/dockerconfigjson'
	// no mesmo namespace, usado para autenticar com o registry.
	// Opcional. Se não especificado, assume-se que o registry é público.
	// +kubebuilder:validation:Optional
	RegistrySecretName string `json:"registrySecretName,omitempty"`

	// A imagem de destino completa (ex: "docker.io/my-org/my-func")
	// O pipeline irá adicionar o digest @sha256:
	// +kubebuilder:validation:Required
	Image string `json:"image"`
}

// DeploySpec define os parâmetros para o runtime
type DeploySpec struct {
	// Opcional. Configura a injeção do sidecar Dapr.
	// +kubebuilder:validation:Optional
	Dapr DaprConfig `json:"dapr,omitempty"`

	// Opcional. Variáveis de ambiente para injetar no container da função.
	// Suporta valores estáticos, referências a Secrets/ConfigMaps, e referências a campos do Pod.
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Opcional. Lista de fontes para popular variáveis de ambiente no container.
	// As chaves definidas em um ConfigMap ou Secret serão expostas como variáveis de ambiente.
	// +kubebuilder:validation:Optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
}

// DaprConfig define os parâmetros de injeção do Dapr
type DaprConfig struct {
	// Se verdadeiro, injeta o sidecar Dapr.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`

	// O App ID exclusivo para o Dapr.
	// +kubebuilder:validation:Required
	AppID string `json:"appID"`

	// A porta em que a aplicação (função) escuta.
	// +kubebuilder:validation:Required
	AppPort int `json:"appPort"`
}

// EventingSpec define a subscrição de eventos
type EventingSpec struct {
	// O nome do Broker para se inscrever.
	// Padrão: 'default' se não especificado.
	// +kubebuilder:validation:Optional
	Broker string `json:"broker,omitempty"`

	// Opcional. Um mapa de atributos para filtrar eventos.
	// +kubebuilder:validation:Optional
	Filters map[string]string `json:"filters,omitempty"`
}

// ObservabilitySpec define as configurações de observabilidade
type ObservabilitySpec struct {
	// Opcional. Configurações de tracing distribuído via OpenTelemetry.
	// +kubebuilder:validation:Optional
	Tracing TracingConfig `json:"tracing,omitempty"`
}

// TracingConfig define as configurações de tracing distribuído
type TracingConfig struct {
	// Se verdadeiro, habilita tracing distribuído via OpenTelemetry.
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled"`

	// Taxa de sampling (0.0 a 1.0). Padrão: usar taxa padrão do OTEL Collector.
	// Deve ser uma string representando um número decimal entre 0.0 e 1.0.
	// Exemplos: "0.1", "0.5", "1.0"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^(0(\.\d+)?|1(\.0+)?)$`
	SamplingRate *string `json:"samplingRate,omitempty"`
}

// FunctionStatus defines the observed state of Function.
type FunctionStatus struct {
	// Condições da função, seguindo as convenções de API do Kubernetes.
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// O digest da imagem imutável do último build bem-sucedido.
	// Ex: "docker.io/my-org/my-func@sha256:..."
	// +kubebuilder:validation:Optional
	ImageDigest string `json:"imageDigest,omitempty"`

	// A URL publicamente acessível da função (do Knative Service).
	// +kubebuilder:validation:Optional
	URL string `json:"url,omitempty"`

	// O 'generation' observado do spec.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Function is the Schema for the functions API.
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FunctionList contains a list of Function.
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Function `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Function{}, &FunctionList{})
}
