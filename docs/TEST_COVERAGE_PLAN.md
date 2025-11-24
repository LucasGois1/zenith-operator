# Plano de Cobertura de Testes - Zenith Operator

## Sumário Executivo

Este documento apresenta um plano abrangente para adicionar testes unitários ao zenith-operator, com o objetivo de aumentar a cobertura de código reportada pelo codecov e melhorar a qualidade geral do código.

**Situação Atual:**
- ✅ 15+ cenários de testes de integração com Chainsaw (funcionando bem)
- ❌ Cobertura codecov reporta 0% (sem testes unitários Go adequados)
- ❌ Arquivo de teste existente (`function_controller_test.go`) tem apenas 1 teste básico que não valida comportamentos críticos
- ❌ Nenhuma configuração de upload de cobertura para codecov no CI

**Meta de Cobertura:**
- **Pacote `internal/controller`**: 70-80%+ de cobertura
- **Pacote `api/v1alpha1`**: 60-70%+ de cobertura
- **Cobertura geral do repositório**: 60%+ de cobertura

---

## Análise do Código Atual

### Estrutura do Código

**Arquivo Principal: `internal/controller/function_controller.go` (632 linhas)**

1. **`FunctionReconciler.Reconcile()`** (linhas 66-407, ~340 linhas)
   - Método monolítico com múltiplas fases de reconciliação
   - Gerenciamento de ServiceAccount com secrets de registry
   - Criação e monitoramento de PipelineRun
   - Extração de image digest dos resultados do build
   - Criação/atualização de Knative Service
   - Criação de Knative Trigger para eventing
   - Gerenciamento de condições de status

2. **Métodos Helper:**
   - `buildPipelineRun()` (linhas 416-513): Constrói PipelineRun do Tekton
   - `buildKnativeService()` (linhas 521-591): Constrói Knative Service com anotações Dapr
   - `buildKnativeTrigger()` (linhas 593-621): Constrói Knative Trigger

3. **API Types: `api/v1alpha1/function_types.go` (140 linhas)**
   - `FunctionSpec` com `BuildSpec`, `DeploySpec`, `EventingSpec` aninhados
   - `DaprConfig` com validações
   - `FunctionStatus` com condições e imageDigest

### Gaps de Cobertura Identificados

**Crítico (Prioridade Alta):**
- ❌ Lógica de reconciliação do ServiceAccount e imagePullSecrets
- ❌ Transições de estado do PipelineRun (running → succeeded/failed)
- ❌ Extração e validação do APP_IMAGE_DIGEST
- ❌ Lógica de criação vs atualização do Knative Service
- ❌ Detecção de mudanças em imagem e anotações Dapr
- ❌ Lógica condicional de criação de Trigger (quando eventing está configurado)
- ❌ Todas as transições de condições de status

**Importante (Prioridade Média):**
- ❌ Construção correta de specs de PipelineRun (tasks, params, workspaces)
- ❌ Construção correta de specs de Knative Service (imagem, portas, Dapr)
- ❌ Construção correta de specs de Trigger (broker, filtros, subscriber)
- ❌ Comportamento com campos opcionais (GitRevision vazio, Eventing vazio)

**Desejável (Prioridade Baixa):**
- ⚠️ Validação de schema da API (já coberto pelo CRD OpenAPI)
- ⚠️ Comportamento de webhooks (não implementado atualmente)

---

## Estratégia de Testes

### Abordagem em Camadas

**1. Testes de Controller com envtest (Prioridade Máxima)**
- Usar o framework Ginkgo/Gomega já configurado em `suite_test.go`
- Testar `Reconcile()` end-to-end contra um API server real (envtest)
- Não usar mocks pesados do client - usar o `k8sClient` real do envtest
- Criar recursos (Function, PipelineRun, ServiceAccount, etc.) diretamente no cluster de teste
- Verificar comportamento do operador através de asserções nos recursos

**2. Testes Unitários Puros para Helpers (Prioridade Alta)**
- Testar funções "puras" que recebem `*Function` e retornam specs
- `buildPipelineRun()`, `buildKnativeService()`, `buildKnativeTrigger()`
- Não precisam de envtest - podem usar testing padrão do Go
- Rápidos e focados em lógica de construção de objetos

**3. Testes de Integração Chainsaw (Manter como está)**
- Continuar usando para validação end-to-end profunda
- Não contribuem para codecov (não são testes Go)
- Validam comportamento real com Tekton/Knative rodando

### Decisões de Design

**✅ Usar envtest em vez de mocks pesados**
- Já está configurado em `suite_test.go`
- Comporta-se como API server real
- Evita fragilidade de mocks complexos
- Suporta validação de CRD e ownerReferences

**✅ Testar Reconcile() por fases**
- Cada teste foca em uma fase específica da reconciliação
- Arrange: preparar estado inicial do cluster
- Act: chamar `Reconcile()` uma ou mais vezes
- Assert: verificar recursos e status da Function

**⚠️ Não simular controllers do Tekton/Knative**
- Envtest não roda controllers do Tekton/Knative
- Definir `PipelineRun.Status` manualmente nos testes para simular estados
- Focar em testar o que o operador escreve no `Spec`, não o que Knative/Tekton fazem

**✅ Considerar refatoração opcional**
- Se testes ficarem muito complexos, extrair helpers privados:
  - `reconcileServiceAccount()`
  - `reconcileBuild()`
  - `reconcileKnativeService()`
  - `reconcileEventing()`
- Não bloquear testes esperando refatoração - começar com estrutura atual

---

## Plano de Implementação Detalhado

### Fase 1: Configuração e Infraestrutura (1-2 horas)

**1.1. Adicionar CRDs do Knative ao envtest**
```go
// Em suite_test.go, adicionar:
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{
        filepath.Join("..", "..", "config", "crd", "bases"),
        filepath.Join("..", "..", "config", "testcrds", "tekton"),
        filepath.Join("..", "..", "config", "testcrds", "knative"), // NOVO
    },
    ErrorIfCRDPathMissing: true,
}
```

**1.2. Criar diretório para CRDs do Knative**
```bash
mkdir -p config/testcrds/knative
# Baixar CRDs do Knative Serving e Eventing
```

**1.3. Configurar upload de cobertura no CI**
```yaml
# Em .github/workflows/test.yml, adicionar após make test:
- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    files: ./cover.out
    flags: unittests
    name: codecov-umbrella
```

**1.4. Adicionar arquivo de configuração do codecov**
```yaml
# Criar codecov.yml na raiz:
coverage:
  status:
    project:
      default:
        target: 60%
        threshold: 2%
    patch:
      default:
        target: 70%
```

### Fase 2: Testes dos Métodos Helper (2-3 horas)

**2.1. Criar `internal/controller/helpers_test.go`**

```go
package controller

import (
    "testing"
    
    . "github.com/onsi/gomega"
    functionsv1alpha1 "github.com/lucasgois1/zenith-operator/api/v1alpha1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPipelineRun(t *testing.T) {
    g := NewWithT(t)
    
    tests := []struct {
        name     string
        function *functionsv1alpha1.Function
        validate func(*testing.T, *tektonv1.PipelineRun)
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
                        Image: "registry.io/test:latest",
                    },
                },
            },
            validate: func(t *testing.T, pr *tektonv1.PipelineRun) {
                g.Expect(pr.Name).To(Equal("test-func-build"))
                g.Expect(pr.Spec.PipelineSpec.Tasks).To(HaveLen(2))
                g.Expect(pr.Spec.PipelineSpec.Tasks[0].Name).To(Equal("fetch-source"))
                g.Expect(pr.Spec.PipelineSpec.Tasks[1].Name).To(Equal("build-and-push"))
                // Validar params
                fetchTask := pr.Spec.PipelineSpec.Tasks[0]
                g.Expect(fetchTask.Params).To(ContainElement(
                    HaveField("Name", "url"),
                ))
            },
        },
        {
            name: "function without GitRevision defaults to main",
            function: &functionsv1alpha1.Function{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-func",
                    Namespace: "default",
                },
                Spec: functionsv1alpha1.FunctionSpec{
                    GitRepo: "https://github.com/user/repo",
                    // GitRevision omitido
                    Build: functionsv1alpha1.BuildSpec{
                        Image: "registry.io/test:latest",
                    },
                },
            },
            validate: func(t *testing.T, pr *tektonv1.PipelineRun) {
                fetchTask := pr.Spec.PipelineSpec.Tasks[0]
                revisionParam := findParam(fetchTask.Params, "revision")
                g.Expect(revisionParam).NotTo(BeNil())
                g.Expect(revisionParam.Value.StringVal).To(Equal("main"))
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := &FunctionReconciler{}
            pr, err := r.buildPipelineRun(tt.function)
            g.Expect(err).NotTo(HaveOccurred())
            tt.validate(t, pr)
        })
    }
}

func TestBuildKnativeService(t *testing.T) {
    g := NewWithT(t)
    
    tests := []struct {
        name     string
        function *functionsv1alpha1.Function
        validate func(*testing.T, *knservingv1.Service)
    }{
        {
            name: "service with Dapr enabled",
            function: &functionsv1alpha1.Function{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-func",
                    Namespace: "default",
                },
                Spec: functionsv1alpha1.FunctionSpec{
                    Deploy: functionsv1alpha1.DeploySpec{
                        Dapr: functionsv1alpha1.DaprConfig{
                            Enabled: true,
                            AppID:   "test-app",
                            AppPort: 8080,
                        },
                    },
                },
                Status: functionsv1alpha1.FunctionStatus{
                    ImageDigest: "registry.io/test@sha256:abc123",
                },
            },
            validate: func(t *testing.T, ksvc *knservingv1.Service) {
                g.Expect(ksvc.Name).To(Equal("test-func"))
                annotations := ksvc.Spec.Template.Annotations
                g.Expect(annotations["dapr.io/enabled"]).To(Equal("true"))
                g.Expect(annotations["dapr.io/app-id"]).To(Equal("test-app"))
                g.Expect(annotations["dapr.io/app-port"]).To(Equal("8080"))
                
                g.Expect(ksvc.Spec.Template.Spec.Containers).To(HaveLen(1))
                container := ksvc.Spec.Template.Spec.Containers[0]
                g.Expect(container.Image).To(Equal("registry.io/test@sha256:abc123"))
                g.Expect(container.Ports[0].ContainerPort).To(Equal(int32(8080)))
            },
        },
        {
            name: "service with Dapr disabled",
            function: &functionsv1alpha1.Function{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-func",
                    Namespace: "default",
                },
                Spec: functionsv1alpha1.FunctionSpec{
                    Deploy: functionsv1alpha1.DeploySpec{
                        Dapr: functionsv1alpha1.DaprConfig{
                            Enabled: false,
                            AppPort: 8080,
                        },
                    },
                },
                Status: functionsv1alpha1.FunctionStatus{
                    ImageDigest: "registry.io/test@sha256:def456",
                },
            },
            validate: func(t *testing.T, ksvc *knservingv1.Service) {
                annotations := ksvc.Spec.Template.Annotations
                g.Expect(annotations).To(BeEmpty())
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := &FunctionReconciler{}
            ksvc, err := r.buildKnativeService(tt.function)
            g.Expect(err).NotTo(HaveOccurred())
            tt.validate(t, ksvc)
        })
    }
}

func TestBuildKnativeTrigger(t *testing.T) {
    g := NewWithT(t)
    
    tests := []struct {
        name     string
        function *functionsv1alpha1.Function
        validate func(*testing.T, *kneventingv1.Trigger)
    }{
        {
            name: "trigger with custom broker and filters",
            function: &functionsv1alpha1.Function{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-func",
                    Namespace: "default",
                },
                Spec: functionsv1alpha1.FunctionSpec{
                    Eventing: functionsv1alpha1.EventingSpec{
                        Broker: "custom-broker",
                        Filters: map[string]string{
                            "type":   "order.created",
                            "source": "payment-service",
                        },
                    },
                },
            },
            validate: func(t *testing.T, trigger *kneventingv1.Trigger) {
                g.Expect(trigger.Name).To(Equal("test-func-trigger"))
                g.Expect(trigger.Spec.Broker).To(Equal("custom-broker"))
                g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("type", "order.created"))
                g.Expect(trigger.Spec.Filter.Attributes).To(HaveKeyWithValue("source", "payment-service"))
                
                g.Expect(trigger.Spec.Subscriber.Ref.Kind).To(Equal("Service"))
                g.Expect(trigger.Spec.Subscriber.Ref.Name).To(Equal("test-func"))
                g.Expect(trigger.Spec.Subscriber.Ref.APIVersion).To(Equal("serving.knative.dev/v1"))
            },
        },
        {
            name: "trigger with default broker",
            function: &functionsv1alpha1.Function{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-func",
                    Namespace: "default",
                },
                Spec: functionsv1alpha1.FunctionSpec{
                    Eventing: functionsv1alpha1.EventingSpec{
                        // Broker vazio - deve usar "default"
                        Filters: map[string]string{},
                    },
                },
            },
            validate: func(t *testing.T, trigger *kneventingv1.Trigger) {
                g.Expect(trigger.Spec.Broker).To(Equal("default"))
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := &FunctionReconciler{}
            trigger := r.buildKnativeTrigger(tt.function)
            tt.validate(t, trigger)
        })
    }
}

// Helper function
func findParam(params []tektonv1.Param, name string) *tektonv1.Param {
    for i := range params {
        if params[i].Name == name {
            return &params[i]
        }
    }
    return nil
}
```

### Fase 3: Testes de Reconciliação com envtest (5-8 horas)

**3.1. Expandir `internal/controller/function_controller_test.go`**

```go
var _ = Describe("Function Controller", func() {
    Context("ServiceAccount Management", func() {
        It("should attach registry secret to ServiceAccount", func() {
            ctx := context.Background()
            
            // Criar ServiceAccount default
            sa := &v1.ServiceAccount{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "default",
                    Namespace: "default",
                },
            }
            Expect(k8sClient.Create(ctx, sa)).To(Succeed())
            
            // Criar Secret de registry
            secret := &v1.Secret{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "registry-secret",
                    Namespace: "default",
                },
                Type: v1.SecretTypeDockerConfigJson,
                Data: map[string][]byte{
                    ".dockerconfigjson": []byte(`{"auths":{}}`),
                },
            }
            Expect(k8sClient.Create(ctx, secret)).To(Succeed())
            
            // Criar Function
            function := &functionsv1alpha1.Function{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-function",
                    Namespace: "default",
                },
                Spec: functionsv1alpha1.FunctionSpec{
                    GitRepo: "https://github.com/test/repo",
                    Build: functionsv1alpha1.BuildSpec{
                        RegistrySecretName: "registry-secret",
                        Image:              "registry.io/test:latest",
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
            
            // Reconciliar
            reconciler := &FunctionReconciler{
                Client: k8sClient,
                Scheme: k8sClient.Scheme(),
            }
            
            result, err := reconciler.Reconcile(ctx, reconcile.Request{
                NamespacedName: types.NamespacedName{
                    Name:      "test-function",
                    Namespace: "default",
                },
            })
            
            Expect(err).NotTo(HaveOccurred())
            Expect(result.Requeue).To(BeTrue())
            
            // Verificar que o secret foi adicionado ao SA
            updatedSA := &v1.ServiceAccount{}
            Expect(k8sClient.Get(ctx, types.NamespacedName{
                Name:      "default",
                Namespace: "default",
            }, updatedSA)).To(Succeed())
            
            Expect(updatedSA.ImagePullSecrets).To(ContainElement(
                v1.LocalObjectReference{Name: "registry-secret"},
            ))
        })
        
        It("should not duplicate secret if already attached", func() {
            // Similar ao teste acima, mas SA já tem o secret
            // Verificar que não há duplicação
        })
    })
    
    Context("PipelineRun Lifecycle", func() {
        It("should create PipelineRun when none exists", func() {
            ctx := context.Background()
            
            // Setup: Function + ServiceAccount com secret já configurado
            // ...
            
            // Reconciliar
            // ...
            
            // Verificar que PipelineRun foi criado
            pr := &tektonv1.PipelineRun{}
            Expect(k8sClient.Get(ctx, types.NamespacedName{
                Name:      "test-function-build",
                Namespace: "default",
            }, pr)).To(Succeed())
            
            // Verificar spec do PipelineRun
            Expect(pr.Spec.PipelineSpec.Tasks).To(HaveLen(2))
            
            // Verificar status da Function
            function := &functionsv1alpha1.Function{}
            Expect(k8sClient.Get(ctx, types.NamespacedName{
                Name:      "test-function",
                Namespace: "default",
            }, function)).To(Succeed())
            
            condition := meta.FindStatusCondition(function.Status.Conditions, "Ready")
            Expect(condition).NotTo(BeNil())
            Expect(condition.Status).To(Equal(metav1.ConditionFalse))
            Expect(condition.Reason).To(Equal("Building"))
        })
        
        It("should requeue while PipelineRun is running", func() {
            ctx := context.Background()
            
            // Setup: Function + PipelineRun em execução
            // Definir PipelineRun.Status manualmente para simular "running"
            pr := &tektonv1.PipelineRun{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-function-build",
                    Namespace: "default",
                },
                Spec: tektonv1.PipelineRunSpec{},
                Status: tektonv1.PipelineRunStatus{
                    Status: duckv1.Status{
                        Conditions: duckv1.Conditions{
                            {
                                Type:   "Succeeded",
                                Status: v1.ConditionUnknown,
                                Reason: "Running",
                            },
                        },
                    },
                },
            }
            Expect(k8sClient.Create(ctx, pr)).To(Succeed())
            
            // Reconciliar
            result, err := reconciler.Reconcile(ctx, reconcile.Request{...})
            
            Expect(err).NotTo(HaveOccurred())
            Expect(result.RequeueAfter).To(Equal(30 * time.Second))
        })
        
        It("should update status to BuildFailed when PipelineRun fails", func() {
            // Setup: PipelineRun com status failed
            pr := &tektonv1.PipelineRun{
                // ...
                Status: tektonv1.PipelineRunStatus{
                    Status: duckv1.Status{
                        Conditions: duckv1.Conditions{
                            {
                                Type:   "Succeeded",
                                Status: v1.ConditionFalse,
                                Reason: "Failed",
                            },
                        },
                    },
                },
            }
            // ...
            
            // Verificar condition BuildFailed
            condition := meta.FindStatusCondition(function.Status.Conditions, "NotReady")
            Expect(condition.Reason).To(Equal("BuildFailed"))
        })
        
        It("should extract image digest when PipelineRun succeeds", func() {
            // Setup: PipelineRun succeeded com APP_IMAGE_DIGEST result
            pr := &tektonv1.PipelineRun{
                // ...
                Status: tektonv1.PipelineRunStatus{
                    Status: duckv1.Status{
                        Conditions: duckv1.Conditions{
                            {
                                Type:   "Succeeded",
                                Status: v1.ConditionTrue,
                            },
                        },
                    },
                    Results: []tektonv1.PipelineRunResult{
                        {
                            Name: "APP_IMAGE_DIGEST",
                            Value: tektonv1.ParamValue{
                                Type:      tektonv1.ParamTypeString,
                                StringVal: "registry.io/test@sha256:abc123def456",
                            },
                        },
                    },
                },
            }
            // ...
            
            // Verificar que imageDigest foi salvo no status
            Expect(function.Status.ImageDigest).To(Equal("registry.io/test@sha256:abc123def456"))
            
            // Verificar condition BuildSucceeded
            condition := meta.FindStatusCondition(function.Status.Conditions, "Ready")
            Expect(condition.Reason).To(Equal("BuildSucceeded"))
        })
        
        It("should handle missing APP_IMAGE_DIGEST result", func() {
            // PipelineRun succeeded mas sem resultado
            // Verificar condition BuildImageError
        })
    })
    
    Context("Knative Service Management", func() {
        It("should create Knative Service after successful build", func() {
            // Setup: Function com imageDigest no status
            // Reconciliar
            // Verificar que KService foi criado com imagem correta
        })
        
        It("should update Knative Service when image changes", func() {
            // Setup: KService existente com imagem antiga
            // Function com novo imageDigest
            // Reconciliar
            // Verificar que imagem foi atualizada
        })
        
        It("should update Knative Service when Dapr config changes", func() {
            // Setup: KService sem anotações Dapr
            // Function com Dapr.Enabled = true
            // Reconciliar
            // Verificar que anotações foram adicionadas
        })
        
        It("should not update Knative Service when nothing changed", func() {
            // Setup: KService já sincronizado
            // Reconciliar
            // Verificar que não houve update (pode verificar via resourceVersion)
        })
    })
    
    Context("Knative Trigger Management", func() {
        It("should create Trigger when eventing is configured", func() {
            // Setup: Function com Eventing.Broker definido
            // KService já existe
            // Reconciliar
            // Verificar que Trigger foi criado
        })
        
        It("should not create Trigger when eventing is not configured", func() {
            // Setup: Function sem Eventing.Broker
            // Reconciliar
            // Verificar que Trigger não existe
        })
        
        It("should set Ready condition after Trigger creation", func() {
            // Verificar que condition Ready=True é definida
        })
    })
    
    Context("OwnerReferences", func() {
        It("should set ownerReference on PipelineRun", func() {
            // Verificar que PipelineRun tem ownerReference para Function
        })
        
        It("should set ownerReference on Knative Service", func() {
            // Verificar que KService tem ownerReference para Function
        })
        
        It("should set ownerReference on Trigger", func() {
            // Verificar que Trigger tem ownerReference para Function
        })
    })
})
```

### Fase 4: Configuração de CI e Codecov (1 hora)

**4.1. Atualizar workflow de testes**
```yaml
# .github/workflows/test.yml
name: Tests

on:
  push:
  pull_request:

jobs:
  test:
    name: Run Unit Tests
    runs-on: ubuntu-latest
    steps:
      - name: Clone the code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Running Tests
        run: |
          go mod tidy
          make test

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./cover.out
          flags: unittests
          name: codecov-umbrella
          fail_ci_if_error: false
        env:
          CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}
```

**4.2. Criar token do Codecov**
- Ir para https://codecov.io e adicionar o repositório
- Copiar o token de upload
- Adicionar como secret `CODECOV_TOKEN` no GitHub

**4.3. Adicionar badge no README**
```markdown
[![codecov](https://codecov.io/gh/LucasGois1/zenith-operator/branch/main/graph/badge.svg)](https://codecov.io/gh/LucasGois1/zenith-operator)
```

---

## Cronograma de Implementação

| Fase | Descrição | Tempo Estimado | Prioridade |
|------|-----------|----------------|------------|
| 1 | Configuração e Infraestrutura | 1-2 horas | Alta |
| 2 | Testes dos Métodos Helper | 2-3 horas | Alta |
| 3 | Testes de Reconciliação (envtest) | 5-8 horas | Crítica |
| 4 | Configuração CI e Codecov | 1 hora | Alta |
| **Total** | | **9-14 horas** | |

---

## Cenários de Teste Prioritários

### Alta Prioridade (Implementar Primeiro)

1. **ServiceAccount + Registry Secret**
   - ✅ Secret não existe → adicionar
   - ✅ Secret já existe → não duplicar
   - ✅ ServiceAccount não existe → erro

2. **PipelineRun Lifecycle**
   - ✅ Não existe → criar
   - ✅ Running → requeue
   - ✅ Succeeded com digest → extrair
   - ✅ Succeeded sem digest → erro
   - ✅ Failed → marcar BuildFailed

3. **Knative Service**
   - ✅ Não existe → criar
   - ✅ Imagem desatualizada → atualizar
   - ✅ Dapr mudou → atualizar
   - ✅ Já sincronizado → noop

4. **Status Conditions**
   - ✅ Building
   - ✅ BuildFailed
   - ✅ BuildSucceeded
   - ✅ BuildImageError
   - ✅ Ready

### Média Prioridade (Implementar Depois)

5. **Knative Trigger**
   - ✅ Eventing configurado → criar
   - ✅ Eventing vazio → não criar
   - ✅ Broker customizado vs default

6. **Helpers**
   - ✅ buildPipelineRun com todos os campos
   - ✅ buildPipelineRun com GitRevision vazio
   - ✅ buildKnativeService com Dapr
   - ✅ buildKnativeService sem Dapr
   - ✅ buildKnativeTrigger com filtros

### Baixa Prioridade (Opcional)

7. **Edge Cases**
   - ⚠️ Múltiplas reconciliações simultâneas
   - ⚠️ Recursos deletados durante reconciliação
   - ⚠️ Conflitos de atualização

---

## Métricas de Sucesso

### Cobertura de Código
- ✅ `internal/controller`: 70%+ de cobertura
- ✅ `api/v1alpha1`: 60%+ de cobertura
- ✅ Repositório geral: 60%+ de cobertura

### Qualidade dos Testes
- ✅ Todos os branches críticos de decisão cobertos
- ✅ Todas as transições de status testadas
- ✅ Testes executam em < 30 segundos
- ✅ Testes são determinísticos (não flaky)

### CI/CD
- ✅ Codecov reporta cobertura corretamente
- ✅ Badge de cobertura no README
- ✅ CI falha se cobertura cair significativamente

---

## Considerações Importantes

### ⚠️ Limitações do envtest

1. **Controllers não rodam**: Tekton e Knative controllers não estão ativos
   - Solução: Definir `Status` manualmente nos testes

2. **Validações de CRD**: Apenas validações OpenAPI são aplicadas
   - Solução: Suficiente para nossos casos

3. **Webhooks**: Não são executados no envtest
   - Solução: OK, não temos webhooks implementados

### ✅ Boas Práticas

1. **Usar table-driven tests**: Facilita adicionar novos casos
2. **Helpers de asserção**: Criar funções para verificar conditions
3. **Cleanup**: Sempre limpar recursos após testes
4. **Nomes descritivos**: Testes devem ser auto-documentados
5. **Focar em comportamento**: Não testar implementação interna

### 🔄 Refatoração Opcional

Se durante a implementação os testes ficarem muito complexos, considerar:

1. **Extrair helpers privados do Reconcile()**:
   ```go
   func (r *FunctionReconciler) reconcileServiceAccount(ctx, function) error
   func (r *FunctionReconciler) reconcileBuild(ctx, function) (ctrl.Result, error)
   func (r *FunctionReconciler) reconcileKnativeService(ctx, function) (ctrl.Result, error)
   func (r *FunctionReconciler) reconcileEventing(ctx, function) (ctrl.Result, error)
   ```

2. **Benefícios**:
   - Testes mais focados e simples
   - Melhor separação de responsabilidades
   - Mais fácil de manter

3. **Quando fazer**:
   - Apenas se testes ficarem muito difíceis
   - Não bloquear implementação de testes

---

## Próximos Passos

1. ✅ **Revisar este plano** com o time
2. ⏭️ **Implementar Fase 1**: Configuração (1-2h)
3. ⏭️ **Implementar Fase 2**: Testes de helpers (2-3h)
4. ⏭️ **Implementar Fase 3**: Testes de reconciliação (5-8h)
5. ⏭️ **Implementar Fase 4**: CI e Codecov (1h)
6. ⏭️ **Validar cobertura**: Verificar que codecov reporta corretamente
7. ⏭️ **Documentar**: Atualizar README com instruções de testes

---

## Referências

- [Controller Runtime Testing](https://book.kubebuilder.io/reference/testing.html)
- [Envtest Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [Tekton PipelineRun API](https://tekton.dev/docs/pipelines/pipelineruns/)
- [Knative Serving API](https://knative.dev/docs/serving/)
- [Codecov Documentation](https://docs.codecov.com/)
