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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kneventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	functionsv1alpha1 "github.com/lucasgois1/zenith-operator/api/v1alpha1"
)

var _ = Describe("Function Controller Reconciliation", func() {
	const (
		timeout       = time.Second * 10
		interval      = time.Millisecond * 250
		testNamespace = "default" // Default Kubernetes namespace for tests
	)

	Context("ServiceAccount Management", func() {
		It("should create a dedicated ServiceAccount for the Function", func() {
			ctx := context.Background()
			functionName := "test-sa-creation"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: "",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconciliation should create the ServiceAccount
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      functionName,
					Namespace: namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify ServiceAccount was created
			sa := &v1.ServiceAccount{}
			saKey := types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, saKey, sa)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify ServiceAccount has correct labels
			Expect(sa.Labels).To(HaveKeyWithValue("functions.zenith.com/managed-by", "zenith-operator"))

			// Verify OwnerReference is set
			Expect(sa.OwnerReferences).To(HaveLen(1))
			Expect(sa.OwnerReferences[0].Name).To(Equal(functionName))
			Expect(sa.OwnerReferences[0].Kind).To(Equal("Function"))
		})

		It("should attach git auth secret to ServiceAccount", func() {
			ctx := context.Background()
			functionName := "test-git-secret"
			namespace := testNamespace
			secretName := "git-auth-secret"

			// Create the git auth secret first
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Type: v1.SecretTypeBasicAuth,
				StringData: map[string]string{
					"username": "testuser",
					"password": "testpass",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, secret)
			}()

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo:           "https://github.com/user/repo",
					GitAuthSecretName: secretName,
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconciliation creates ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation attaches the secret
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount has the git secret attached
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa)
				if err != nil {
					return false
				}
				for _, secretRef := range sa.Secrets {
					if secretRef.Name == secretName {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should attach registry secret to ServiceAccount imagePullSecrets", func() {
			ctx := context.Background()
			functionName := "test-registry-secret"
			namespace := testNamespace
			secretName := "registry-secret"

			// Create the registry secret first
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Type: v1.SecretTypeDockerConfigJson,
				StringData: map[string]string{
					".dockerconfigjson": `{"auths":{"registry.io":{"username":"test","password":"test"}}}`,
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, secret)
			}()

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image:              "registry.io/test:latest",
						RegistrySecretName: secretName,
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconciliation creates ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation attaches the secret
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount has the registry secret in imagePullSecrets
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa)
				if err != nil {
					return false
				}
				for _, secretRef := range sa.ImagePullSecrets {
					if secretRef.Name == secretName {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})

		It("should set status to GitAuthMissing when git secret does not exist", func() {
			ctx := context.Background()
			functionName := "test-missing-git-secret"
			namespace := testNamespace
			secretName := "nonexistent-secret"

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo:           "https://github.com/user/repo",
					GitAuthSecretName: secretName,
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconciliation creates ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation should detect missing secret
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status condition
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "GitAuthMissing"
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("PipelineRun Lifecycle", func() {
		It("should create PipelineRun when ServiceAccount is ready", func() {
			ctx := context.Background()
			functionName := "test-pipelinerun-create"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconciliation creates ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Second reconciliation creates PipelineRun
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify PipelineRun was created
			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify PipelineRun has correct ServiceAccount
			Expect(pr.Spec.TaskRunTemplate.ServiceAccountName).To(Equal(functionName + "-sa"))

			// Verify OwnerReference is set
			Expect(pr.OwnerReferences).To(HaveLen(1))
			Expect(pr.OwnerReferences[0].Name).To(Equal(functionName))

			// Verify status was updated to Building
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "Building"
			}, timeout, interval).Should(BeTrue())
		})

		It("should update status to BuildFailed when PipelineRun fails", func() {
			ctx := context.Background()
			functionName := "test-pipelinerun-failed"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Create ServiceAccount and PipelineRun
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Simulate PipelineRun failure by updating its status
			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{
				{
					Type:   apis.ConditionSucceeded,
					Status: v1.ConditionFalse,
					Reason: "Failed",
				},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			// Reconcile again to detect the failure
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify status was updated to BuildFailed
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "BuildFailed"
			}, timeout, interval).Should(BeTrue())
		})

		It("should extract image digest when PipelineRun succeeds", func() {
			ctx := context.Background()
			functionName := "test-pipelinerun-success"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Create ServiceAccount and PipelineRun
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Simulate PipelineRun success by updating its status
			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			testDigest := "sha256:abc123def456"
			pr.Status.Conditions = []apis.Condition{
				{
					Type:   apis.ConditionSucceeded,
					Status: v1.ConditionTrue,
					Reason: "Succeeded",
				},
			}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{
					Name: "APP_IMAGE_DIGEST",
					Value: tektonv1.ResultValue{
						Type:      tektonv1.ParamTypeString,
						StringVal: testDigest,
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			// Reconcile again to extract the digest
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify image digest was saved to status
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				expectedImage := "registry.io/test:latest@" + testDigest
				return updatedFunction.Status.ImageDigest == expectedImage
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Knative Service Management", func() {
		It("should create Knative Service after successful build", func() {
			ctx := context.Background()
			functionName := "test-ksvc-create"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Create ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Create PipelineRun
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Simulate successful PipelineRun
			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{
				{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue, Reason: "Succeeded"},
			}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:test123"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			// Reconcile to create Knative Service
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify Knative Service was created
			ksvc := &knservingv1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify OwnerReference
			Expect(ksvc.OwnerReferences).To(HaveLen(1))
			Expect(ksvc.OwnerReferences[0].Name).To(Equal(functionName))

			// Verify image uses digest
			Expect(ksvc.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(ksvc.Spec.Template.Spec.Containers[0].Image).To(ContainSubstring("@sha256:test123"))
		})

		It("should update Knative Service when image changes", func() {
			ctx := context.Background()
			functionName := "test-ksvc-update"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Setup: Create ServiceAccount, PipelineRun, and initial Knative Service
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr) == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue}}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:old123"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			// Wait for Knative Service to be created
			ksvc := &knservingv1.Service{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc) == nil
			}, timeout, interval).Should(BeTrue())

			// Simulate new build with different digest
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:new456"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			// Update Function status with new digest
			updatedFunction := &functionsv1alpha1.Function{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)).To(Succeed())
			updatedFunction.Status.ImageDigest = "registry.io/test:latest@sha256:new456"
			Expect(k8sClient.Status().Update(ctx, updatedFunction)).To(Succeed())

			// Reconcile to update Knative Service
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify Knative Service was updated with new image
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc)
				if err != nil {
					return false
				}
				return ksvc.Spec.Template.Spec.Containers[0].Image == "registry.io/test:latest@sha256:new456"
			}, timeout, interval).Should(BeTrue())
		})

		It("should configure Dapr annotations when enabled", func() {
			ctx := context.Background()
			functionName := "test-ksvc-dapr"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: true,
							AppID:   "test-app",
							AppPort: 8080,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Setup: Create ServiceAccount, PipelineRun with success
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr) == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue}}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:test789"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			// Reconcile to create Knative Service with Dapr
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			// Verify Knative Service has Dapr annotations
			ksvc := &knservingv1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc)
				if err != nil {
					return false
				}
				annotations := ksvc.Spec.Template.Annotations
				return annotations["dapr.io/enabled"] == "true" &&
					annotations["dapr.io/app-id"] == "test-app" &&
					annotations["dapr.io/app-port"] == "8080" &&
					annotations["dapr.io/metrics-port"] == "9095"
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Knative Trigger Management", func() {
		It("should create Trigger when eventing is configured", func() {
			ctx := context.Background()
			functionName := "test-trigger-create"
			namespace := testNamespace
			brokerName := "test-broker"

			// Create broker first
			broker := &kneventingv1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      brokerName,
					Namespace: namespace,
				},
				Spec: kneventingv1.BrokerSpec{},
			}
			Expect(k8sClient.Create(ctx, broker)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, broker)
			}()

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker: brokerName,
						Filters: map[string]string{
							"type": "com.example.test",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Setup: Create ServiceAccount, PipelineRun, Knative Service
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr) == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue}}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:test999"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			// Wait for Knative Service and mark it as ready
			ksvc := &knservingv1.Service{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc) == nil
			}, timeout, interval).Should(BeTrue())

			ksvc.Status.Conditions = duckv1.Conditions{
				{Type: "Ready", Status: v1.ConditionTrue},
			}
			Expect(k8sClient.Status().Update(ctx, ksvc)).To(Succeed())

			// Reconcile to create Trigger
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify Trigger was created
			trigger := &kneventingv1.Trigger{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-trigger", Namespace: namespace}, trigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Verify Trigger configuration
			Expect(trigger.Spec.Broker).To(Equal(brokerName))
			Expect(trigger.OwnerReferences).To(HaveLen(1))
			Expect(trigger.OwnerReferences[0].Name).To(Equal(functionName))
		})

		It("should delete Trigger when eventing is removed", func() {
			ctx := context.Background()
			functionName := "test-trigger-delete"
			namespace := testNamespace
			brokerName := "test-broker-2"

			// Create broker
			broker := &kneventingv1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      brokerName,
					Namespace: namespace,
				},
				Spec: kneventingv1.BrokerSpec{},
			}
			Expect(k8sClient.Create(ctx, broker)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, broker)
			}()

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker: brokerName,
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Setup: Create full stack including Trigger
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr) == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue}}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:test888"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			ksvc := &knservingv1.Service{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc) == nil
			}, timeout, interval).Should(BeTrue())

			ksvc.Status.Conditions = duckv1.Conditions{{Type: "Ready", Status: v1.ConditionTrue}}
			Expect(k8sClient.Status().Update(ctx, ksvc)).To(Succeed())

			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			// Verify Trigger exists
			trigger := &kneventingv1.Trigger{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-trigger", Namespace: namespace}, trigger) == nil
			}, timeout, interval).Should(BeTrue())

			// Remove eventing from Function
			updatedFunction := &functionsv1alpha1.Function{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)).To(Succeed())
			updatedFunction.Spec.Eventing.Broker = ""
			Expect(k8sClient.Update(ctx, updatedFunction)).To(Succeed())

			// Reconcile to delete Trigger
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify Trigger was deleted
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-trigger", Namespace: namespace}, trigger)
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
		})

		It("should set status to BrokerNotFound when broker does not exist", func() {
			ctx := context.Background()
			functionName := "test-broker-missing"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker: "nonexistent-broker",
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Setup: Create ServiceAccount, PipelineRun
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr) == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue}}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:test777"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			// Reconcile should detect missing broker
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status condition
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "BrokerNotFound"
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Environment Variable Validation", func() {
		It("should fail when Secret referenced in Env does not exist", func() {
			ctx := context.Background()
			functionName := "test-env-secret-missing"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Env: []v1.EnvVar{
							{
								Name: "DB_PASSWORD",
								ValueFrom: &v1.EnvVarSource{
									SecretKeyRef: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "nonexistent-secret"},
										Key:                  "password",
									},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile: Create ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for ServiceAccount to be created
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa) == nil
			}, timeout, interval).Should(BeTrue())

			// Second reconcile: Should detect missing secret during PipelineRun creation
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status condition
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "SecretNotFound"
			}, timeout, interval).Should(BeTrue())
		})

		It("should succeed when Secret referenced in Env exists", func() {
			ctx := context.Background()
			functionName := "test-env-secret-exists"
			namespace := testNamespace
			secretName := "db-secret"

			// Create the secret
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				StringData: map[string]string{
					"password": "secret123",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, secret)
			}()

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Env: []v1.EnvVar{
							{
								Name: "DB_PASSWORD",
								ValueFrom: &v1.EnvVarSource{
									SecretKeyRef: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: secretName},
										Key:                  "password",
									},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Reconcile should succeed and create ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount was created (validation passed)
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa) == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should fail when ConfigMap referenced in Env does not exist", func() {
			ctx := context.Background()
			functionName := "test-env-cm-missing"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Env: []v1.EnvVar{
							{
								Name: "APP_CONFIG",
								ValueFrom: &v1.EnvVarSource{
									ConfigMapKeyRef: &v1.ConfigMapKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "nonexistent-cm"},
										Key:                  "config.yaml",
									},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile: Create ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for ServiceAccount to be created
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa) == nil
			}, timeout, interval).Should(BeTrue())

			// Second reconcile: Should detect missing configmap during PipelineRun creation
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status condition
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "ConfigMapNotFound"
			}, timeout, interval).Should(BeTrue())
		})

		It("should fail when Secret referenced in EnvFrom does not exist", func() {
			ctx := context.Background()
			functionName := "test-envfrom-secret-missing"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						EnvFrom: []v1.EnvFromSource{
							{
								SecretRef: &v1.SecretEnvSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "nonexistent-secret"},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile: Create ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for ServiceAccount to be created
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa) == nil
			}, timeout, interval).Should(BeTrue())

			// Second reconcile: Should detect missing secret during PipelineRun creation
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status condition
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "SecretNotFound"
			}, timeout, interval).Should(BeTrue())
		})

		It("should fail when ConfigMap referenced in EnvFrom does not exist", func() {
			ctx := context.Background()
			functionName := "test-envfrom-cm-missing"
			namespace := testNamespace

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						EnvFrom: []v1.EnvFromSource{
							{
								ConfigMapRef: &v1.ConfigMapEnvSource{
									LocalObjectReference: v1.LocalObjectReference{Name: "nonexistent-cm"},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// First reconcile: Create ServiceAccount
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for ServiceAccount to be created
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa) == nil
			}, timeout, interval).Should(BeTrue())

			// Second reconcile: Should detect missing configmap during PipelineRun creation
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			// Verify status condition
			updatedFunction := &functionsv1alpha1.Function{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)
				if err != nil {
					return false
				}
				condition := meta.FindStatusCondition(updatedFunction.Status.Conditions, "Ready")
				return condition != nil && condition.Reason == "ConfigMapNotFound"
			}, timeout, interval).Should(BeTrue())
		})

		It("should skip validation for optional Secret references", func() {
			ctx := context.Background()
			functionName := "test-env-secret-optional"
			namespace := testNamespace
			optional := true

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
						Env: []v1.EnvVar{
							{
								Name: "OPTIONAL_SECRET",
								ValueFrom: &v1.EnvVarSource{
									SecretKeyRef: &v1.SecretKeySelector{
										LocalObjectReference: v1.LocalObjectReference{Name: "nonexistent-secret"},
										Key:                  "key",
										Optional:             &optional,
									},
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Reconcile should succeed (optional secret is skipped)
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify ServiceAccount was created (validation passed)
			sa := &v1.ServiceAccount{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-sa", Namespace: namespace}, sa) == nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Trigger Filter Updates", func() {
		It("should update Trigger when filters change", func() {
			ctx := context.Background()
			functionName := "test-trigger-filter-update"
			namespace := testNamespace
			brokerName := "test-broker-filters"

			// Create broker
			broker := &kneventingv1.Broker{
				ObjectMeta: metav1.ObjectMeta{
					Name:      brokerName,
					Namespace: namespace,
				},
				Spec: kneventingv1.BrokerSpec{},
			}
			Expect(k8sClient.Create(ctx, broker)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, broker)
			}()

			function := &functionsv1alpha1.Function{
				ObjectMeta: metav1.ObjectMeta{
					Name:      functionName,
					Namespace: namespace,
				},
				Spec: functionsv1alpha1.FunctionSpec{
					GitRepo: "https://github.com/user/repo",
					Build: functionsv1alpha1.BuildSpec{
						Image: "registry.io/test:latest",
					},
					Deploy: functionsv1alpha1.DeploySpec{
						Dapr: functionsv1alpha1.DaprConfig{
							Enabled: false,
							AppPort: 8080,
						},
					},
					Eventing: functionsv1alpha1.EventingSpec{
						Broker: brokerName,
						Filters: map[string]string{
							"type": "com.example.v1",
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, function)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, function)
			}()

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			// Setup: Create full stack with initial filters
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})
			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			pr := &tektonv1.PipelineRun{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-build", Namespace: namespace}, pr) == nil
			}, timeout, interval).Should(BeTrue())

			pr.Status.Conditions = []apis.Condition{{Type: apis.ConditionSucceeded, Status: v1.ConditionTrue}}
			pr.Status.Results = []tektonv1.PipelineRunResult{
				{Name: "APP_IMAGE_DIGEST", Value: tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "sha256:filter123"}},
			}
			Expect(k8sClient.Status().Update(ctx, pr)).To(Succeed())

			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			ksvc := &knservingv1.Service{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, ksvc) == nil
			}, timeout, interval).Should(BeTrue())

			ksvc.Status.Conditions = duckv1.Conditions{{Type: "Ready", Status: v1.ConditionTrue}}
			Expect(k8sClient.Status().Update(ctx, ksvc)).To(Succeed())

			_, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace}})

			// Verify Trigger exists with initial filters
			trigger := &kneventingv1.Trigger{}
			Eventually(func() bool {
				return k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-trigger", Namespace: namespace}, trigger) == nil
			}, timeout, interval).Should(BeTrue())

			// Update Function with new filters
			updatedFunction := &functionsv1alpha1.Function{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: functionName, Namespace: namespace}, updatedFunction)).To(Succeed())
			updatedFunction.Spec.Eventing.Filters = map[string]string{
				"type":   "com.example.v2",
				"source": "my-source",
			}
			Expect(k8sClient.Update(ctx, updatedFunction)).To(Succeed())

			// Reconcile to update Trigger
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: functionName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify Trigger was updated with new filters
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: functionName + "-trigger", Namespace: namespace}, trigger)
				if err != nil {
					return false
				}
				return trigger.Spec.Filter != nil &&
					trigger.Spec.Filter.Attributes["type"] == "com.example.v2" &&
					trigger.Spec.Filter.Attributes["source"] == "my-source"
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("extractPipelineRunFailure", func() {
		It("should return default values when PipelineRun has no conditions", func() {
			ctx := context.Background()
			namespace := testNamespace

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-no-conditions",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("BuildFailed"))
			Expect(message).To(Equal("O build falhou"))
		})

		It("should extract reason and message from PipelineRun Succeeded condition", func() {
			ctx := context.Background()
			namespace := testNamespace

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-with-condition",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "PipelineFailed",
								Message: "Pipeline execution failed due to task error",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("PipelineFailed"))
			Expect(message).To(Equal("Pipeline execution failed due to task error"))
		})

		It("should use default reason when condition has empty reason", func() {
			ctx := context.Background()
			namespace := testNamespace

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-empty-reason",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "",
								Message: "Some failure message",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("BuildFailed"))
			Expect(message).To(Equal("Some failure message"))
		})

		It("should extract details from failed TaskRun with PipelineTaskName", func() {
			ctx := context.Background()
			namespace := testNamespace
			taskRunName := "test-pr-taskrun-fetch-source"

			// Create the TaskRun first (without status)
			taskRun := &tektonv1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, taskRun)
			}()

			// Update the status separately (status is a subresource)
			taskRun.Status = tektonv1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  v1.ConditionFalse,
							Reason:  "TaskRunFailed",
							Message: "authentication required for https://github.com/user/repo",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, taskRun)).To(Succeed())

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-with-taskrun",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "Failed",
								Message: "Tasks Completed: 0, Failed: 1",
							},
						},
					},
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						ChildReferences: []tektonv1.ChildStatusReference{
							{
								TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
								Name:             taskRunName,
								PipelineTaskName: "fetch-source",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("TaskRunFailed"))
			Expect(message).To(Equal("Task 'fetch-source' falhou: authentication required for https://github.com/user/repo"))
		})

		It("should use TaskRun name when PipelineTaskName is empty", func() {
			ctx := context.Background()
			namespace := testNamespace
			taskRunName := "test-pr-taskrun-no-pipeline-task"

			// Create the TaskRun first (without status)
			taskRun := &tektonv1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, taskRun)
			}()

			// Update the status separately (status is a subresource)
			taskRun.Status = tektonv1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  v1.ConditionFalse,
							Reason:  "BuildError",
							Message: "build failed: exit code 1",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, taskRun)).To(Succeed())

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-empty-pipeline-task",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "Failed",
								Message: "Tasks Completed: 0, Failed: 1",
							},
						},
					},
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						ChildReferences: []tektonv1.ChildStatusReference{
							{
								TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
								Name:             taskRunName,
								PipelineTaskName: "",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("BuildError"))
			Expect(message).To(Equal("Task '" + taskRunName + "' falhou: build failed: exit code 1"))
		})

		It("should skip non-TaskRun child references", func() {
			ctx := context.Background()
			namespace := testNamespace

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-non-taskrun-child",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "PipelineFailed",
								Message: "Pipeline failed",
							},
						},
					},
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						ChildReferences: []tektonv1.ChildStatusReference{
							{
								TypeMeta: runtime.TypeMeta{Kind: "CustomRun"},
								Name:     "some-custom-run",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("PipelineFailed"))
			Expect(message).To(Equal("Pipeline failed"))
		})

		It("should skip successful TaskRuns", func() {
			ctx := context.Background()
			namespace := testNamespace
			taskRunName := "test-pr-taskrun-success"

			// Create the TaskRun first (without status)
			taskRun := &tektonv1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskRunName,
					Namespace: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, taskRun)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, taskRun)
			}()

			// Update the status separately (status is a subresource)
			taskRun.Status = tektonv1.TaskRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:    apis.ConditionSucceeded,
							Status:  v1.ConditionTrue,
							Reason:  "Succeeded",
							Message: "Task completed successfully",
						},
					},
				},
			}
			Expect(k8sClient.Status().Update(ctx, taskRun)).To(Succeed())

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-success-taskrun",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "PipelineFailed",
								Message: "Pipeline failed for other reasons",
							},
						},
					},
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						ChildReferences: []tektonv1.ChildStatusReference{
							{
								TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
								Name:             taskRunName,
								PipelineTaskName: "fetch-source",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("PipelineFailed"))
			Expect(message).To(Equal("Pipeline failed for other reasons"))
		})

		It("should handle TaskRun not found gracefully", func() {
			ctx := context.Background()
			namespace := testNamespace

			reconciler := &FunctionReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			pr := &tektonv1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pr-taskrun-not-found",
					Namespace: namespace,
				},
				Status: tektonv1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    apis.ConditionSucceeded,
								Status:  v1.ConditionFalse,
								Reason:  "PipelineFailed",
								Message: "Pipeline failed",
							},
						},
					},
					PipelineRunStatusFields: tektonv1.PipelineRunStatusFields{
						ChildReferences: []tektonv1.ChildStatusReference{
							{
								TypeMeta:         runtime.TypeMeta{Kind: "TaskRun"},
								Name:             "nonexistent-taskrun",
								PipelineTaskName: "fetch-source",
							},
						},
					},
				},
			}

			reason, message := reconciler.extractPipelineRunFailure(ctx, pr)
			Expect(reason).To(Equal("PipelineFailed"))
			Expect(message).To(Equal("Pipeline failed"))
		})
	})
})
