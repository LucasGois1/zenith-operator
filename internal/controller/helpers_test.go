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

package controller

import (
	"testing"

	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kneventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"

	functionsv1alpha1 "github.com/lucasgois1/zenith-operator/api/v1alpha1"
)

func TestBuildPipelineRun(t *testing.T) {
	tests := []struct {
		name     string
		function *functionsv1alpha1.Function
		validate func(*testing.T, *tektonv1.PipelineRun, *GomegaWithT)
	}{
		{
			name: "basic function with all required fields",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo:     "https://github.com/user/repo",
					GitRevision: "main",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "registry-secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			},
			validate: func(t *testing.T, pr *tektonv1.PipelineRun, g *GomegaWithT) {
				g.Expect(pr.Name).To(Equal("test-func-build"))
				g.Expect(pr.Namespace).To(Equal("default"))
				g.Expect(pr.Spec.PipelineSpec.Tasks).To(HaveLen(2))

				// Verify fetch-source task
				fetchTask := pr.Spec.PipelineSpec.Tasks[0]
				g.Expect(fetchTask.Name).To(Equal("fetch-source"))
				g.Expect(fetchTask.TaskRef.Name).To(Equal("git-clone"))

				urlParam := findParam(fetchTask.Params, "url")
				g.Expect(urlParam).NotTo(BeNil())
				g.Expect(urlParam.Value.StringVal).To(Equal("https://github.com/user/repo"))

				revisionParam := findParam(fetchTask.Params, "revision")
				g.Expect(revisionParam).NotTo(BeNil())
				g.Expect(revisionParam.Value.StringVal).To(Equal("main"))

				// Verify build-and-push task
				buildTask := pr.Spec.PipelineSpec.Tasks[1]
				g.Expect(buildTask.Name).To(Equal("build-and-push"))
				g.Expect(buildTask.TaskRef.Name).To(Equal("buildpacks-phases"))
				g.Expect(buildTask.RunAfter).To(ContainElement("fetch-source"))

				imageParam := findParam(buildTask.Params, "APP_IMAGE")
				g.Expect(imageParam).NotTo(BeNil())
				g.Expect(imageParam.Value.StringVal).To(Equal("registry.io/test:latest"))

				// Verify ServiceAccount (now uses function-name-sa pattern)
				g.Expect(pr.Spec.TaskRunTemplate.ServiceAccountName).To(Equal("test-func-sa"))

				// Verify workspaces
				g.Expect(pr.Spec.PipelineSpec.Workspaces).To(HaveLen(1))
				g.Expect(pr.Spec.PipelineSpec.Workspaces[0].Name).To(Equal("source-workspace"))

				g.Expect(pr.Spec.Workspaces).To(HaveLen(1))
				g.Expect(pr.Spec.Workspaces[0].Name).To(Equal("source-workspace"))
				// Workspace now uses VolumeClaimTemplate instead of EmptyDir
				g.Expect(pr.Spec.Workspaces[0].VolumeClaimTemplate).NotTo(BeNil())
			},
		},
		{
			name: "function without GitRevision defaults to main",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-no-revision",
					Namespace: "test-ns",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/another-repo",
					// GitRevision omitted
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/another:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			},
			validate: func(t *testing.T, pr *tektonv1.PipelineRun, g *GomegaWithT) {
				fetchTask := pr.Spec.PipelineSpec.Tasks[0]
				revisionParam := findParam(fetchTask.Params, "revision")
				g.Expect(revisionParam).NotTo(BeNil())
				g.Expect(revisionParam.Value.StringVal).To(Equal("main"), "GitRevision should default to 'main'")
			},
		},
		{
			name: "function with custom branch",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-custom-branch",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo:     "https://github.com/user/repo",
					GitRevision: "feature/new-feature",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			},
			validate: func(t *testing.T, pr *tektonv1.PipelineRun, g *GomegaWithT) {
				fetchTask := pr.Spec.PipelineSpec.Tasks[0]
				revisionParam := findParam(fetchTask.Params, "revision")
				g.Expect(revisionParam).NotTo(BeNil())
				g.Expect(revisionParam.Value.StringVal).To(Equal("feature/new-feature"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &FunctionReconciler{}
			pr := r.buildPipelineRun(tt.function)
			g.Expect(pr).NotTo(BeNil())
			tt.validate(t, pr, g)
		})
	}
}

func TestBuildKnativeService(t *testing.T) {
	tests := []struct {
		name     string
		function *functionsv1alpha1.Function
		validate func(*testing.T, *knservingv1.Service, *GomegaWithT)
	}{
		{
			name: "service with Dapr enabled",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-dapr",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: true,
							AppID:   "test-app",
							AppPort: 8080,
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:abc123def456",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				g.Expect(ksvc.Name).To(Equal("test-func-dapr"))
				g.Expect(ksvc.Namespace).To(Equal("default"))

				// Verify Dapr annotations
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(HaveKeyWithValue("dapr.io/enabled", "true"))
				g.Expect(annotations).To(HaveKeyWithValue("dapr.io/app-id", "test-app"))
				g.Expect(annotations).To(HaveKeyWithValue("dapr.io/app-port", "8080"))

				// Verify container spec
				g.Expect(ksvc.Spec.Template.Spec.Containers).To(HaveLen(1))
				container := ksvc.Spec.Template.Spec.Containers[0]
				g.Expect(container.Image).To(Equal("registry.io/test@sha256:abc123def456"))
				g.Expect(container.Ports).To(HaveLen(1))
				g.Expect(container.Ports[0].ContainerPort).To(Equal(int32(8080)))
			},
		},
		{
			name: "service with Dapr disabled",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-no-dapr",
					Namespace: "test-ns",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 3000,
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:def456abc789",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				// Verify no Dapr annotations when disabled
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(BeEmpty())

				// Verify container still uses correct image and port (default is now 8080)
				container := ksvc.Spec.Template.Spec.Containers[0]
				g.Expect(container.Image).To(Equal("registry.io/test@sha256:def456abc789"))
				g.Expect(container.Ports[0].ContainerPort).To(Equal(int32(8080)))
			},
		},
		{
			name: "service with different app port",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-custom-port",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: true,
							AppID:   "custom-app",
							AppPort: 9090,
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:custom123",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(HaveKeyWithValue("dapr.io/app-port", "9090"))

				container := ksvc.Spec.Template.Spec.Containers[0]
				g.Expect(container.Ports[0].ContainerPort).To(Equal(int32(9090)))
			},
		},
		{
			name: "service with only minScale configured",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-minscale",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Scale: &functionsv1alpha1.ScaleSpec{
							MinScale: int32Ptr(1),
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:minscale123",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(HaveKeyWithValue("autoscaling.knative.dev/min-scale", "1"))
				g.Expect(annotations).NotTo(HaveKey("autoscaling.knative.dev/max-scale"))
			},
		},
		{
			name: "service with only maxScale configured",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-maxscale",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Scale: &functionsv1alpha1.ScaleSpec{
							MaxScale: int32Ptr(5),
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:maxscale123",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(HaveKeyWithValue("autoscaling.knative.dev/max-scale", "5"))
				g.Expect(annotations).NotTo(HaveKey("autoscaling.knative.dev/min-scale"))
			},
		},
		{
			name: "service with both minScale and maxScale configured",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-both-scale",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Scale: &functionsv1alpha1.ScaleSpec{
							MinScale: int32Ptr(1),
							MaxScale: int32Ptr(10),
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:bothscale123",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(HaveKeyWithValue("autoscaling.knative.dev/min-scale", "1"))
				g.Expect(annotations).To(HaveKeyWithValue("autoscaling.knative.dev/max-scale", "10"))
			},
		},
		{
			name: "service with scale nil (no scale annotations)",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-no-scale",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Scale: nil,
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:noscale123",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).NotTo(HaveKey("autoscaling.knative.dev/min-scale"))
				g.Expect(annotations).NotTo(HaveKey("autoscaling.knative.dev/max-scale"))
			},
		},
		{
			name: "service with zero values for scale (annotations set to 0)",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-zero-scale",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Scale: &functionsv1alpha1.ScaleSpec{
							MinScale: int32Ptr(0),
							MaxScale: int32Ptr(0),
						},
					},
				},
				Status: functionsv1alpha1.FunctionStatus{
					ImageDigest: "registry.io/test@sha256:zeroscale123",
				},
			},
			validate: func(t *testing.T, ksvc *knservingv1.Service, g *GomegaWithT) {
				annotations := ksvc.Spec.Template.Annotations
				g.Expect(annotations).To(HaveKeyWithValue("autoscaling.knative.dev/min-scale", "0"))
				g.Expect(annotations).To(HaveKeyWithValue("autoscaling.knative.dev/max-scale", "0"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &FunctionReconciler{}
			ksvc := r.buildKnativeService(tt.function)
			g.Expect(ksvc).NotTo(BeNil())
			tt.validate(t, ksvc, g)
		})
	}
}

func TestBuildKnativeTrigger(t *testing.T) {
	tests := []struct {
		name     string
		function *functionsv1alpha1.Function
		validate func(*testing.T, *kneventingv1.Trigger, *GomegaWithT)
	}{
		{
			name: "trigger with custom broker and filters",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-eventing",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker: "custom-broker",
						Filters: map[string]string{
							"type":   "order.created",
							"source": "payment-service",
						},
					},
				},
			},
			validate: func(t *testing.T, trigger *kneventingv1.Trigger, g *GomegaWithT) {
				g.Expect(trigger.Name).To(Equal("test-func-eventing-trigger"))
				g.Expect(trigger.Namespace).To(Equal("default"))
				g.Expect(trigger.Spec.Broker).To(Equal("custom-broker"))

				// Verify filters
				g.Expect(trigger.Spec.Filter).NotTo(BeNil())
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("type", "order.created"))
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("source", "payment-service"))

				// Verify subscriber reference
				g.Expect(trigger.Spec.Subscriber.Ref).NotTo(BeNil())
				g.Expect(trigger.Spec.Subscriber.Ref.Kind).To(Equal("Service"))
				g.Expect(trigger.Spec.Subscriber.Ref.Name).To(Equal("test-func-eventing"))
				g.Expect(trigger.Spec.Subscriber.Ref.Namespace).To(Equal("default"))
				g.Expect(trigger.Spec.Subscriber.Ref.APIVersion).To(Equal("serving.knative.dev/v1"))
			},
		},
		{
			name: "trigger with default broker",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-default-broker",
					Namespace: "test-ns",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						// Broker empty - should use "default"
						Filters: map[string]string{
							"type": "test.event",
						},
					},
				},
			},
			validate: func(t *testing.T, trigger *kneventingv1.Trigger, g *GomegaWithT) {
				g.Expect(trigger.Spec.Broker).To(Equal("default"))
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("type", "test.event"))
			},
		},
		{
			name: "trigger with empty filters",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-no-filters",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker:  "my-broker",
						Filters: map[string]string{},
					},
				},
			},
			validate: func(t *testing.T, trigger *kneventingv1.Trigger, g *GomegaWithT) {
				g.Expect(trigger.Spec.Broker).To(Equal("my-broker"))
				g.Expect(trigger.Spec.Filter.Attributes).To(BeEmpty())
			},
		},
		{
			name: "trigger with multiple filters",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func-multi-filters",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "secret",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker: "events",
						Filters: map[string]string{
							"type":       "transaction.completed",
							"source":     "payment-gateway",
							"dataschema": "v1",
						},
					},
				},
			},
			validate: func(t *testing.T, trigger *kneventingv1.Trigger, g *GomegaWithT) {
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveLen(3))
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("type", "transaction.completed"))
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("source", "payment-gateway"))
				g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("dataschema", "v1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			r := &FunctionReconciler{}
			trigger := r.buildKnativeTrigger(tt.function)
			g.Expect(trigger).NotTo(BeNil())
			tt.validate(t, trigger, g)
		})
	}
}

// Helper function to find a parameter by name
func findParam(params []tektonv1.Param, name string) *tektonv1.Param {
	for i := range params {
		if params[i].Name == name {
			return &params[i]
		}
	}
	return nil
}

func TestResolveEnvVars(t *testing.T) {
	g := NewWithT(t)
	reconciler := &FunctionReconciler{}

	tests := []struct {
		name     string
		function *functionsv1alpha1.Function
		validate func([]v1.EnvVar)
	}{
		{
			name: "should inject OpenTelemetry variables when tracing is enabled",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Observability: functionsv1alpha1.ObservabilitySpec{
						Tracing: functionsv1alpha1.TracingConfig{
							Enabled: true,
						},
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(4))
				g.Expect(envVars[0].Name).To(Equal("OTEL_EXPORTER_OTLP_ENDPOINT"))
				g.Expect(envVars[0].Value).To(Equal("http://otel-collector-collector.opentelemetry-operator-system.svc.cluster.local:4318"))
				g.Expect(envVars[1].Name).To(Equal("OTEL_SERVICE_NAME"))
				g.Expect(envVars[1].Value).To(Equal("test-func"))
				g.Expect(envVars[2].Name).To(Equal("OTEL_RESOURCE_ATTRIBUTES"))
				g.Expect(envVars[2].Value).To(Equal("service.namespace=default,service.version=latest"))
				g.Expect(envVars[3].Name).To(Equal("OTEL_TRACES_EXPORTER"))
				g.Expect(envVars[3].Value).To(Equal("otlp"))
			},
		},
		{
			name: "should inject sampling rate when specified",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-func",
					Namespace: "default",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Observability: functionsv1alpha1.ObservabilitySpec{
						Tracing: functionsv1alpha1.TracingConfig{
							Enabled:      true,
							SamplingRate: stringPtr("0.5"),
						},
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(6))
				g.Expect(envVars[4].Name).To(Equal("OTEL_TRACES_SAMPLER"))
				g.Expect(envVars[4].Value).To(Equal("traceidratio"))
				g.Expect(envVars[5].Name).To(Equal("OTEL_TRACES_SAMPLER_ARG"))
				g.Expect(envVars[5].Value).To(Equal("0.5"))
			},
		},
		{
			name: "should resolve fieldRef metadata.name",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
					UID:       "test-uid-123",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name: "FUNCTION_NAME",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.name",
									},
								},
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(1))
				g.Expect(envVars[0].Name).To(Equal("FUNCTION_NAME"))
				g.Expect(envVars[0].Value).To(Equal("my-function"))
				g.Expect(envVars[0].ValueFrom).To(BeNil())
			},
		},
		{
			name: "should resolve fieldRef metadata.namespace",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name: "NAMESPACE",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(1))
				g.Expect(envVars[0].Name).To(Equal("NAMESPACE"))
				g.Expect(envVars[0].Value).To(Equal("prod"))
			},
		},
		{
			name: "should resolve fieldRef metadata.uid",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
					UID:       "test-uid-456",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name: "FUNCTION_UID",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.uid",
									},
								},
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(1))
				g.Expect(envVars[0].Name).To(Equal("FUNCTION_UID"))
				g.Expect(envVars[0].Value).To(Equal("test-uid-456"))
			},
		},
		{
			name: "should set empty value for unsupported fieldRef paths",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name: "LABELS",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.labels",
									},
								},
							},
							{
								Name: "ANNOTATIONS",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "metadata.annotations",
									},
								},
							},
							{
								Name: "UNKNOWN",
								ValueFrom: &v1.EnvVarSource{
									FieldRef: &v1.ObjectFieldSelector{
										FieldPath: "status.phase",
									},
								},
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(3))
				g.Expect(envVars[0].Value).To(Equal(""))
				g.Expect(envVars[1].Value).To(Equal(""))
				g.Expect(envVars[2].Value).To(Equal(""))
			},
		},
		{
			name: "should set empty value for resourceFieldRef",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name: "CPU_LIMIT",
								ValueFrom: &v1.EnvVarSource{
									ResourceFieldRef: &v1.ResourceFieldSelector{
										Resource: "limits.cpu",
									},
								},
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(1))
				g.Expect(envVars[0].Name).To(Equal("CPU_LIMIT"))
				g.Expect(envVars[0].Value).To(Equal(""))
				g.Expect(envVars[0].ValueFrom).To(BeNil())
			},
		},
		{
			name: "should keep Secret and ConfigMap refs as-is",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name: "DB_PASSWORD",
								ValueFrom: &v1.EnvVarSource{
									SecretKeyRef: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "db-secret"},
										Key:                  "password",
									},
								},
							},
							{
								Name: "CONFIG_VALUE",
								ValueFrom: &v1.EnvVarSource{
									ConfigMapKeyRef: &v1.ConfigMapKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "app-config"},
										Key:                  "value",
									},
								},
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(2))
				g.Expect(envVars[0].Name).To(Equal("DB_PASSWORD"))
				g.Expect(envVars[0].ValueFrom).NotTo(BeNil())
				g.Expect(envVars[0].ValueFrom.SecretKeyRef).NotTo(BeNil())
				g.Expect(envVars[1].Name).To(Equal("CONFIG_VALUE"))
				g.Expect(envVars[1].ValueFrom).NotTo(BeNil())
				g.Expect(envVars[1].ValueFrom.ConfigMapKeyRef).NotTo(BeNil())
			},
		},
		{
			name: "should keep plain value env vars as-is",
			function: &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-function",
					Namespace: "prod",
				},
				Spec: functionsv1alpha1.FunctionSpec{
					Deploy: functionsv1alpha1.DeploySpec{
						Env: []v1.EnvVar{
							{
								Name:  "PLAIN_VAR",
								Value: "plain-value",
							},
						},
					},
				},
			},
			validate: func(envVars []v1.EnvVar) {
				g.Expect(envVars).To(HaveLen(1))
				g.Expect(envVars[0].Name).To(Equal("PLAIN_VAR"))
				g.Expect(envVars[0].Value).To(Equal("plain-value"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.resolveEnvVars(tt.function)
			tt.validate(result)
		})
	}
}

func TestDetectInsecureRegistries(t *testing.T) {
	g := NewWithT(t)
	reconciler := &FunctionReconciler{}

	tests := []struct {
		name     string
		imageURL string
		expected string
		setup    func()
		cleanup  func()
	}{
		{
			name:     "should detect cluster-internal registry",
			imageURL: "registry.registry.svc.cluster.local:5000/myapp:latest",
			expected: "registry.registry.svc.cluster.local:5000",
		},
		{
			name:     "should detect localhost registry",
			imageURL: "localhost:5000/myapp:latest",
			expected: "localhost:5000",
		},
		{
			name:     "should detect 127.0.0.1 registry",
			imageURL: "127.0.0.1:5000/myapp:latest",
			expected: "127.0.0.1:5000",
		},
		{
			name:     "should detect custom registry with port",
			imageURL: "my-registry.local:8080/myapp:latest",
			expected: "my-registry.local:8080",
		},
		{
			name:     "should not detect docker.io as insecure",
			imageURL: "docker.io:443/myapp:latest",
			expected: "",
		},
		{
			name:     "should not detect gcr.io as insecure",
			imageURL: "gcr.io:443/myapp:latest",
			expected: "",
		},
		{
			name:     "should not detect ghcr.io as insecure",
			imageURL: "ghcr.io:443/myapp:latest",
			expected: "",
		},
		{
			name:     "should not detect quay.io as insecure",
			imageURL: "quay.io:443/myapp:latest",
			expected: "",
		},
		{
			name:     "should not detect registry.k8s.io as insecure",
			imageURL: "registry.k8s.io:443/myapp:latest",
			expected: "",
		},
		{
			name:     "should return empty for image without registry",
			imageURL: "myapp:latest",
			expected: "",
		},
		{
			name:     "should return empty for single-part image name",
			imageURL: "myapp",
			expected: "",
		},
		{
			name:     "should use INSECURE_REGISTRIES env var when set",
			imageURL: "any-registry.com/myapp:latest",
			expected: "custom-registry.local:5000,another-registry:8080",
			setup: func() {
				t.Setenv("INSECURE_REGISTRIES", "custom-registry.local:5000,another-registry:8080")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			result := reconciler.detectInsecureRegistries(tt.imageURL)
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}
