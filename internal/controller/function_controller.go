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
	"os"
	"strconv"
	"strings"
	"time"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kneventingv1 "knative.dev/eventing/pkg/apis/eventing/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	knservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	functionsv1alpha1 "github.com/lucasgois1/zenith-operator/api/v1alpha1"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=functions.zenith.com,resources=functions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=functions.zenith.com,resources=functions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=functions.zenith.com,resources=functions/finalizers,verbs=update
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventing.knative.dev,resources=triggers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Function object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
//
//nolint:gocyclo // Monolithic reconcile function; to be refactored into phases in a follow-up
func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. Obter o recurso 'Function' que acionou esta reconciliação
	var function functionsv1alpha1.Function
	if err := r.Get(ctx, req.NamespacedName, &function); err != nil {
		log.Error(err, "Não foi possível buscar o recurso Function")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	saName := function.Name + "-sa"
	serviceAccount := &v1.ServiceAccount{}
	saKey := types.NamespacedName{Name: saName, Namespace: function.Namespace}

	err := r.Get(ctx, saKey, serviceAccount)
	if err != nil && errors.IsNotFound(err) {
		// ServiceAccount não existe, criar um novo
		log.Info("Criando ServiceAccount dedicado para Function", "ServiceAccountName", saName)
		serviceAccount = &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: function.Namespace,
			},
		}

		if err := controllerutil.SetControllerReference(&function, serviceAccount, r.Scheme); err != nil {
			log.Error(err, "Falha ao definir OwnerReference no ServiceAccount")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, serviceAccount); err != nil {
			log.Error(err, "Falha ao criar ServiceAccount")
			return ctrl.Result{}, err
		}

		log.Info("ServiceAccount criado com sucesso")
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Falha ao buscar ServiceAccount")
		return ctrl.Result{}, err
	}

	// 3. Configurar secrets no ServiceAccount
	needsUpdate := false

	gitSecretName := function.Spec.GitAuthSecretName
	if gitSecretName != "" {
		// Verificar se o secret existe
		gitSecret := &v1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: gitSecretName, Namespace: function.Namespace}, gitSecret); err != nil {
			if errors.IsNotFound(err) {
				log.Error(err, "Git auth secret não encontrado", "SecretName", gitSecretName)
				// Atualizar status com erro
				gitAuthMissingCondition := metav1.Condition{
					Type:    "Ready",
					Status:  metav1.ConditionFalse,
					Reason:  "GitAuthMissing",
					Message: "Git authentication secret não encontrado: " + gitSecretName,
				}
				meta.SetStatusCondition(&function.Status.Conditions, gitAuthMissingCondition)
				if err := r.Status().Update(ctx, &function); err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{RequeueAfter: time.Second * 30}, nil
			}
			return ctrl.Result{}, err
		}

		// Adicionar secret à lista de secrets do ServiceAccount (não imagePullSecrets)
		found := false
		for _, secretRef := range serviceAccount.Secrets {
			if secretRef.Name == gitSecretName {
				found = true
				break
			}
		}
		if !found {
			log.Info("Adicionando Git auth secret ao ServiceAccount", "SecretName", gitSecretName)
			serviceAccount.Secrets = append(serviceAccount.Secrets, v1.ObjectReference{Name: gitSecretName})
			needsUpdate = true
		}
	}

	registrySecretName := function.Spec.Build.RegistrySecretName
	if registrySecretName != "" {
		// Adicionar à lista imagePullSecrets
		found := false
		for _, secretRef := range serviceAccount.ImagePullSecrets {
			if secretRef.Name == registrySecretName {
				found = true
				break
			}
		}
		if !found {
			log.Info("Adicionando Registry secret ao ServiceAccount", "SecretName", registrySecretName)
			serviceAccount.ImagePullSecrets = append(serviceAccount.ImagePullSecrets, v1.LocalObjectReference{Name: registrySecretName})
			needsUpdate = true
		}
	}

	// 3.3. Atualizar ServiceAccount se necessário
	if needsUpdate {
		if err := r.Update(ctx, serviceAccount); err != nil {
			log.Error(err, "Falha ao atualizar ServiceAccount")
			return ctrl.Result{}, err
		}
		log.Info("ServiceAccount atualizado com sucesso")
		return ctrl.Result{Requeue: true}, nil
	}

	pipelineRunName := function.Name + "-build"
	pipelineRun := &tektonv1.PipelineRun{}

	// Tenta obter o PipelineRun que gerenciamos
	err = r.Get(ctx, types.NamespacedName{Name: pipelineRunName, Namespace: function.Namespace}, pipelineRun)

	// Verifica se o PipelineRun não existe
	if err != nil && errors.IsNotFound(err) {
		log.Info("PipelineRun não encontrado. Criando um novo...", "PipelineRun.Name", pipelineRunName)

		// 1. Construir o objeto PipelineRun em Go
		newPipelineRun := r.buildPipelineRun(&function)

		// 2. Definir o OwnerReference [2]
		// Isso torna o 'Function' dono do 'PipelineRun'.
		if err := controllerutil.SetControllerReference(&function, newPipelineRun, r.Scheme); err != nil {
			log.Error(err, "Falha ao definir OwnerReference no PipelineRun")
			return ctrl.Result{}, err
		}

		// 3. Criar o PipelineRun no cluster
		if err := r.Create(ctx, newPipelineRun); err != nil {
			log.Error(err, "Falha ao criar PipelineRun")
			return ctrl.Result{}, err
		}

		// 4. Atualizar o Status para "Building" e solicitar nova fila (requeue)
		log.Info("PipelineRun criado com sucesso. Atualizando status para 'Building'.")

		newCondition := metav1.Condition{
			Type:    "Ready", // Tipo de condição padrão
			Status:  metav1.ConditionFalse,
			Reason:  "Building",
			Message: "Pipeline de build iniciado",
		}

		// Usa a função 'meta.SetStatusCondition' correta do pacote 'k8s.io/apimachinery/pkg/api/meta'
		meta.SetStatusCondition(&function.Status.Conditions, newCondition)
		function.Status.ObservedGeneration = function.Generation

		// Atualiza o sub-recurso de status [3]
		if err := r.Status().Update(ctx, &function); err != nil {
			log.Error(err, "Falha ao atualizar o status da Função para 'Building'")
			return ctrl.Result{}, err
		}

		// Retorna com Requeue: true para que possamos começar a monitorar
		// o status do PipelineRun na próxima reconciliação.
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		// Algum outro erro ocorreu ao tentar obter o PipelineRun
		log.Error(err, "Falha ao obter o PipelineRun")
		return ctrl.Result{}, err
	}

	// Se chegamos aqui, 'err' foi 'nil', o que significa que o PipelineRun já existe.
	log.Info("PipelineRun já existe, passando para a fase de monitoramento.")

	// --- FIM DA LÓGICA DO PASSO 3.2.2 ---

	// 1. Verificar se o PipelineRun terminou
	if !pipelineRun.IsDone() {
		log.Info("PipelineRun is still running", "PipelineRun.Name", pipelineRun.Name)
		// Ainda em execução, verificar novamente em 30 segundos
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// 2. Verificar se falhou
	if pipelineRun.IsFailure() {
		log.Error(nil, "PipelineRun failed", "PipelineRun.Name", pipelineRun.Name)
		// (Atualizar Status para "BuildFailed" e parar)
		buildFailedCondition := metav1.Condition{
			Type:    "Ready", // Usar tipo "Ready" consistentemente
			Status:  metav1.ConditionFalse,
			Reason:  "BuildFailed",
			Message: "O build falhou",
		}
		meta.SetStatusCondition(&function.Status.Conditions, buildFailedCondition)
		function.Status.ObservedGeneration = function.Generation
		if err := r.Status().Update(ctx, &function); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil // Não requeue em falha
	}

	// 3. Sucesso! Extrair o ImageDigest.
	log.Info("PipelineRun succeeded", "PipelineRun.Name", pipelineRun.Name)
	imageDigest := ""
	for _, result := range pipelineRun.Status.Results {
		// O nome 'APP_IMAGE_DIGEST' é definido pela Task 'buildpacks-phases'
		if result.Name == "APP_IMAGE_DIGEST" { //
			imageDigest = result.Value.StringVal
			break
		}
	}

	if imageDigest == "" {
		imageErrorCondition := metav1.Condition{
			Type:    "Ready", // Tipo de condição padrão
			Status:  metav1.ConditionFalse,
			Reason:  "BuildImageError",
			Message: "Ocorreu um erro ao gerar o digest da imagem",
		}
		meta.SetStatusCondition(&function.Status.Conditions, imageErrorCondition)
		function.Status.ObservedGeneration = function.Generation

		if err := r.Status().Update(ctx, &function); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil // Não requeue - erro permanente
	}

	// 4. Salvar o digest no Status e passar para a próxima fase.
	// Construir a referência completa da imagem com o digest
	imageWithDigest := function.Spec.Build.Image + "@" + imageDigest
	function.Status.ImageDigest = imageWithDigest
	buildSucceededCondition := metav1.Condition{
		Type:    "Ready", // Tipo de condição padrão
		Status:  metav1.ConditionFalse,
		Reason:  "BuildSucceeded",
		Message: "Imagem gerada com sucesso",
	}
	meta.SetStatusCondition(&function.Status.Conditions, buildSucceededCondition)
	function.Status.ObservedGeneration = function.Generation

	if err := r.Status().Update(ctx, &function); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Iniciando Fase 3.4: Reconciliação do Knative Service")

	knativeServiceName := function.Name
	knativeService := &knservingv1.Service{}

	// 1. Construir o estado DESEJADO do Knative Service
	// Fazemos isso primeiro para que possamos usá-lo tanto para criar quanto para comparar/atualizar.
	desiredKsvc := r.buildKnativeService(&function)

	// 2. Tentar obter o estado ATUAL do Knative Service no cluster
	err = r.Get(ctx, types.NamespacedName{Name: knativeServiceName, Namespace: function.Namespace}, knativeService)

	if err != nil && errors.IsNotFound(err) {
		// --- CAMINHO DE CRIAÇÃO ---
		log.Info("Knative Service não encontrado. Criando...", "KnativeService.Name", knativeServiceName)

		// 3. Definir o OwnerReference
		// Isso é CRÍTICO para que o 'Function' gerencie o ciclo de vida do 'KnativeService' [1, 2]
		if err := controllerutil.SetControllerReference(&function, desiredKsvc, r.Scheme); err != nil {
			log.Error(err, "Falha ao definir OwnerReference no Knative Service")
			return ctrl.Result{}, err
		}

		// 4. Criar o Knative Service no cluster
		if err := r.Create(ctx, desiredKsvc); err != nil {
			log.Error(err, "Falha ao criar Knative Service")
			return ctrl.Result{}, err
		}

		log.Info("Knative Service criado com sucesso.")
		// Re-enfileira a requisição. A próxima reconciliação irá monitorar
		// o status do ksvc recém-criado (e eventualmente passar para a Fase 3.5).
		return ctrl.Result{Requeue: true}, nil

	} else if err != nil {
		// Erro real ao tentar o Get
		log.Error(err, "Falha ao obter Knative Service")
		return ctrl.Result{}, err
	}

	// --- CAMINHO DE ATUALIZAÇÃO ---
	// Se chegamos aqui, o Knative Service FOI encontrado.
	log.Info("Knative Service encontrado. Verificando se há atualizações...")

	needsUpdate = false

	// 1. Verificar se a imagem está desatualizada
	// Compara a imagem no cluster com a imagem que acabamos de construir
	// NOTA: O Knative Service cria um PodSpec com UM contêiner.
	currentImage := ""
	if len(knativeService.Spec.Template.Spec.Containers) > 0 {
		currentImage = knativeService.Spec.Template.Spec.Containers[0].Image
	}
	desiredImage := desiredKsvc.Spec.Template.Spec.Containers[0].Image

	if currentImage != desiredImage {
		log.Info("ImageDigest desatualizado, marcando para atualização.", "Atual", currentImage, "Desejado", desiredImage)
		needsUpdate = true
	}

	// 2. Verificar se as anotações do Dapr mudaram
	// (Uma verificação 'reflect.DeepEqual' é mais robusta, mas isso cobre o caso principal)
	if len(knativeService.Spec.Template.Annotations) != len(desiredKsvc.Spec.Template.Annotations) {
		needsUpdate = true
	} else {
		for k, v := range desiredKsvc.Spec.Template.Annotations {
			if knativeService.Spec.Template.Annotations[k] != v {
				log.Info("Anotações do Dapr mudaram, marcando para atualização.")
				needsUpdate = true
				break
			}
		}
	}

	// 3. Executar a atualização se necessário
	if needsUpdate {
		log.Info("Atualizando Knative Service...")
		// Atualiza o spec do objeto existente com o spec desejado
		knativeService.Spec = desiredKsvc.Spec
		if err := r.Update(ctx, knativeService); err != nil {
			log.Error(err, "Falha ao atualizar Knative Service")
			return ctrl.Result{}, err
		}
		log.Info("Knative Service atualizado com sucesso.")
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Knative Service está sincronizado.")
	// --- FIM DA FASE 3.4 ---

	// Atualizar URL se disponível
	if knativeService.Status.URL != nil {
		function.Status.URL = knativeService.Status.URL.String()
	}

	// Se 'eventing' não estiver configurado, marcar como Ready e parar.
	if function.Spec.Eventing.Broker == "" {
		readyCondition := metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Function deployed and ready to accept requests",
		}
		meta.SetStatusCondition(&function.Status.Conditions, readyCondition)
		function.Status.ObservedGeneration = function.Generation
		if err := r.Status().Update(ctx, &function); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	triggerName := function.Name + "-trigger"
	trigger := &kneventingv1.Trigger{}

	err = r.Get(ctx, types.NamespacedName{Name: triggerName, Namespace: function.Namespace}, trigger)
	if err != nil && errors.IsNotFound(err) {
		// Trigger não encontrado, vamos criar.

		// 1. Construir o Trigger
		newTrigger := r.buildKnativeTrigger(&function) // Função helper (ver abaixo)

		// 2. Definir OwnerReference
		if err := controllerutil.SetControllerReference(&function, newTrigger, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		// 3. Criar
		log.Info("Creating new Knative Trigger", "Trigger.Name", newTrigger.Name)
		if err := r.Create(ctx, newTrigger); err != nil {
			return ctrl.Result{}, err
		}

		readyCondition := metav1.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "Ready",
			Message: "Function deployed with eventing and ready to accept requests",
		}
		// 4. Atualizar Status para "Ready"
		meta.SetStatusCondition(&function.Status.Conditions, readyCondition)
		function.Status.ObservedGeneration = function.Generation
		if err := r.Status().Update(ctx, &function); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil // Fim!
	} else if err != nil {
		// Erro real ao tentar o Get
		log.Error(err, "Falha ao obter Knative Trigger")
		return ctrl.Result{}, err
	}

	// --- CAMINHO DE ATUALIZAÇÃO DO TRIGGER ---
	// Se chegamos aqui, o Trigger FOI encontrado.
	log.Info("Knative Trigger encontrado. Verificando se há atualizações...")

	// Construir o Trigger desejado
	desiredTrigger := r.buildKnativeTrigger(&function)

	needsUpdate = false

	// 1. Verificar se o broker mudou
	if trigger.Spec.Broker != desiredTrigger.Spec.Broker {
		log.Info("Broker mudou, marcando para atualização.", "Atual", trigger.Spec.Broker, "Desejado", desiredTrigger.Spec.Broker)
		needsUpdate = true
	}

	// 2. Verificar se os filtros mudaram
	// Comparar os atributos do filtro
	currentFilters := trigger.Spec.Filter
	desiredFilters := desiredTrigger.Spec.Filter

	if currentFilters == nil && desiredFilters != nil {
		needsUpdate = true
	} else if currentFilters != nil && desiredFilters == nil {
		needsUpdate = true
	} else if currentFilters != nil && desiredFilters != nil {
		// Comparar os atributos
		if len(currentFilters.Attributes) != len(desiredFilters.Attributes) {
			needsUpdate = true
		} else {
			for k, v := range desiredFilters.Attributes {
				if currentFilters.Attributes[k] != v {
					log.Info("Filtros do Trigger mudaram, marcando para atualização.")
					needsUpdate = true
					break
				}
			}
		}
	}

	// 3. Executar a atualização se necessário
	if needsUpdate {
		log.Info("Atualizando Knative Trigger...")
		// Atualiza o spec do objeto existente com o spec desejado
		trigger.Spec = desiredTrigger.Spec
		if err := r.Update(ctx, trigger); err != nil {
			log.Error(err, "Falha ao atualizar Knative Trigger")
			return ctrl.Result{}, err
		}
		log.Info("Knative Trigger atualizado com sucesso.")
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Knative Trigger está sincronizado.")

	readyCondition := metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "Ready",
		Message: "Function deployed with eventing and ready to accept requests",
	}
	meta.SetStatusCondition(&function.Status.Conditions, readyCondition)
	function.Status.ObservedGeneration = function.Generation
	if err := r.Status().Update(ctx, &function); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil // Tudo pronto.

}

/*
buildPipelineParams constrói os parâmetros para a task de buildpacks.
Implementa lógica inteligente para detectar quando usar registries inseguros:
 1. Verifica variável de ambiente INSECURE_REGISTRIES para configuração explícita
 2. Detecta automaticamente registries locais/cluster-internal baseado no hostname da imagem
 3. Suporta múltiplos registries inseguros separados por vírgula
*/
func (r *FunctionReconciler) buildPipelineParams(function *functionsv1alpha1.Function) []tektonv1.Param {
	params := []tektonv1.Param{
		{Name: "APP_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: function.Spec.Build.Image}},
		{Name: "CNB_BUILDER_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "paketobuildpacks/builder-jammy-base:latest"}},
		{Name: "CNB_PROCESS_TYPE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
	}

	// Determinar registries inseguros usando lógica inteligente
	insecureRegistries := r.detectInsecureRegistries(function.Spec.Build.Image)
	if insecureRegistries != "" {
		params = append(params, tektonv1.Param{
			Name:  "CNB_INSECURE_REGISTRIES",
			Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: insecureRegistries},
		})
	}

	return params
}

/*
detectInsecureRegistries implementa lógica inteligente para detectar registries inseguros.
Prioridade de detecção:
 1. Variável de ambiente INSECURE_REGISTRIES (configuração explícita do usuário)
 2. Auto-detecção baseada no hostname da imagem:
    - Registries cluster-internal (.svc.cluster.local)
    - Registries localhost (localhost, 127.0.0.1)
    - Registries com portas não-padrão (indicam ambiente de desenvolvimento)
 3. Retorna string vazia para registries públicos conhecidos (docker.io, gcr.io, etc.)
*/
func (r *FunctionReconciler) detectInsecureRegistries(imageURL string) string {
	// 1. Verificar configuração explícita via variável de ambiente
	if envInsecure := os.Getenv("INSECURE_REGISTRIES"); envInsecure != "" {
		return envInsecure
	}

	parts := strings.Split(imageURL, "/")
	if len(parts) == 0 {
		return ""
	}

	if len(parts) == 1 {
		return ""
	}

	potentialRegistry := parts[0]

	if strings.Contains(potentialRegistry, ".svc.cluster.local") {
		return potentialRegistry
	}

	if strings.HasPrefix(potentialRegistry, "localhost") ||
		strings.HasPrefix(potentialRegistry, "127.0.0.1") {
		return potentialRegistry
	}

	if strings.Contains(potentialRegistry, ":") {
		// Verificar se não é um registry público conhecido
		if !strings.Contains(potentialRegistry, "docker.io") &&
			!strings.Contains(potentialRegistry, "gcr.io") &&
			!strings.Contains(potentialRegistry, "ghcr.io") &&
			!strings.Contains(potentialRegistry, "quay.io") &&
			!strings.Contains(potentialRegistry, "registry.k8s.io") {
			return potentialRegistry
		}
	}

	// Retorna string vazia (sem CNB_INSECURE_REGISTRIES)
	return ""
}

/*
buildPipelineRun constrói um *tektonv1.PipelineRun em memória.
Este PipelineRun é projetado para:
 1. Clonar um repositório Git usando a Task 'git-clone'.
 2. Construir uma imagem de contêiner usando Cloud Native Buildpacks com a Task 'buildpacks-phases'.
 3. Enviar a imagem para o registry especificado.
*/
func (r *FunctionReconciler) buildPipelineRun(function *functionsv1alpha1.Function) *tektonv1.PipelineRun {
	pipelineRunName := function.Name + "-build"

	// Define 'main' como padrão para a revisão do git se não for especificado
	gitRevision := function.Spec.GitRevision
	if gitRevision == "" {
		gitRevision = "main"
	}

	serviceAccountName := function.Name + "-sa"
	const sharedWorkspaceName = "source-workspace"

	return &tektonv1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: function.Namespace,
		},
		Spec: tektonv1.PipelineRunSpec{
			// Vincula o PipelineRun ao ServiceAccount dedicado que tem as credenciais
			TaskRunTemplate: tektonv1.PipelineTaskRunTemplate{
				ServiceAccountName: serviceAccountName,
			},

			// 'pipelineSpec' define um pipeline embutido [4]
			PipelineSpec: &tektonv1.PipelineSpec{
				// 1. DECLARAÇÃO DE WORKSPACE:
				// Declara os workspaces que as Tasks deste pipeline precisarão.
				// Isto é um slice de 'PipelineWorkspaceDeclaration'.
				Workspaces: []tektonv1.PipelineWorkspaceDeclaration{
					{Name: sharedWorkspaceName, Description: "", Optional: false},
				},

				// Declara os resultados que o pipeline irá expor a partir das tasks.
				Results: []tektonv1.PipelineResult{
					{
						Name:        "APP_IMAGE_DIGEST",
						Description: "The digest of the built application image",
						Value:       tektonv1.ResultValue{Type: tektonv1.ParamTypeString, StringVal: "$(tasks.build-and-push.results.APP_IMAGE_DIGEST)"},
					},
				},

				// 2. DEFINIÇÃO DAS TASKS:
				// Isto é um slice de 'PipelineTask'.
				Tasks: []tektonv1.PipelineTask{
					// --- Task 1: Git Clone ---
					{
						Name: "fetch-source",
						TaskRef: &tektonv1.TaskRef{
							Name: "git-clone", // Refere-se à Task 'git-clone' instalada [5]
							Kind: "Task",
						},
						// 'Workspaces' aqui é um slice de 'WorkspacePipelineTaskBinding'
						Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
							{
								Name:      "output",            // A task 'git-clone' define seu workspace de saída como 'output'
								Workspace: sharedWorkspaceName, // Mapeia para o workspace 'source-workspace' do pipeline
							},
						},
						// 'Params' é um slice de 'Param'
						Params: []tektonv1.Param{
							{Name: "url", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: function.Spec.GitRepo}},
							{Name: "revision", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: gitRevision}},
						},
					},
					// --- Task 2: Buildpacks ---
					{
						Name: "build-and-push",
						TaskRef: &tektonv1.TaskRef{
							Name: "buildpacks-phases", // Refere-se à Task 'buildpacks-phases' instalada [5]
							Kind: "Task",
						},
						// 'RunAfter' é um slice de 'string' [5]
						RunAfter: []string{"fetch-source"}, // Garante que o clone termine antes do build começar

						// 'Workspaces' aqui é um slice de 'WorkspacePipelineTaskBinding'
						Workspaces: []tektonv1.WorkspacePipelineTaskBinding{
							{
								Name:      "source",            // A task 'buildpacks' define seu workspace de entrada como 'source' [5]
								Workspace: sharedWorkspaceName, // Mapeia para o mesmo workspace
							},
						},
						Params: r.buildPipelineParams(function),
					},
				},
			},

			// 3. VINCULAÇÃO DE WORKSPACE (Workspace Binding):
			// Esta seção 'Workspaces' está no nível 'spec', não 'pipelineSpec'.
			// Ela *cumpre* a declaração de workspace feita acima.
			// Isto é um slice de 'WorkspaceBinding'.
			Workspaces: []tektonv1.WorkspaceBinding{
				{
					Name: sharedWorkspaceName, // Corresponde ao nome em 'pipelineSpec.workspaces'

					VolumeClaimTemplate: &v1.PersistentVolumeClaim{
						Spec: v1.PersistentVolumeClaimSpec{
							AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
							Resources: v1.VolumeResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
		},
	}
}

/*
buildKnativeService constrói um objeto *knservingv1.Service em memória
baseado no Spec da Função e no ImageDigest do Status.
Ele adere à API 'v1' do Knative Serving, onde o ServiceSpec
contém ConfigurationSpec e RouteSpec embutidos.
*/
func (r *FunctionReconciler) buildKnativeService(function *functionsv1alpha1.Function) *knservingv1.Service {

	// --- Ponto de Integração do Dapr ---
	// Estas anotações devem ser aplicadas ao TEMPLATE do Pod.[3, 4]
	podAnnotations := make(map[string]string)
	if function.Spec.Deploy.Dapr.Enabled {
		podAnnotations["dapr.io/enabled"] = "true"
		podAnnotations["dapr.io/app-id"] = function.Spec.Deploy.Dapr.AppID
		podAnnotations["dapr.io/app-port"] = strconv.Itoa(function.Spec.Deploy.Dapr.AppPort)
		// (Adicione outras anotações Dapr conforme necessário, ex: config, log-level) [4]
	}
	// ------------------------------------

	// Determinar a porta do container
	containerPort := int32(8080)
	if function.Spec.Deploy.Dapr.Enabled && function.Spec.Deploy.Dapr.AppPort > 0 {
		containerPort = int32(function.Spec.Deploy.Dapr.AppPort)
	}

	// Construir variáveis de ambiente
	envVars := []v1.EnvVar{
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	}
	for _, e := range function.Spec.Deploy.Env {
		envVars = append(envVars, v1.EnvVar{
			Name:  e.Name,
			Value: e.Value,
		})
	}

	// Construir a definição do container
	container := v1.Container{
		// Usa o digest do build bem-sucedido da Fase 3.3
		Image: function.Status.ImageDigest,
		Ports: []v1.ContainerPort{ // Ports é um slice
			{
				// Informa ao Knative a porta que o contêiner da aplicação escuta
				// Isso é importante para o Dapr saber para onde encaminhar [4, 5]
				ContainerPort: containerPort,
			},
		},
		Env: envVars,
	}

	// Construir o Service object
	ksvc := &knservingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      function.Name,
			Namespace: function.Namespace,
			Labels: map[string]string{
				"networking.knative.dev/visibility": "cluster-local",
			},
		},
		// O Spec 'v1' do Knative Service [6]
		Spec: knservingv1.ServiceSpec{

			// 1. 'ConfigurationSpec' é embutido (inlined) [5]
			// É aqui que a definição do Pod (Template) reside.
			ConfigurationSpec: knservingv1.ConfigurationSpec{
				Template: knservingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						// As anotações do Dapr vão aqui para injeção do sidecar [4]
						Annotations: podAnnotations,
					},
					// 2. 'Spec' é o 'RevisionSpec'
					Spec: knservingv1.RevisionSpec{

						// --- CORREÇÃO CRÍTICA ---
						// RevisionSpec incorpora (inlines) um corev1.PodSpec [6]
						// Nós preenchemos os campos relevantes do PodSpec aqui.
						PodSpec: v1.PodSpec{
							Containers: []v1.Container{container}, // Containers é um slice [6]
							// Outros campos do PodSpec podem ser definidos aqui se necessário
						},
						// ------------------------

						// (Campos opcionais do RevisionSpec podem ser definidos aqui)
						// ContainerConcurrency: &int64{100},
						// TimeoutSeconds: &int64{300},
					},
				},
			},

			// 3. 'RouteSpec' também é embutido [5]
			// Deixamos em branco para usar o comportamento padrão do Knative:
			// 100% do tráfego para a "latestReadyRevision".
			RouteSpec: knservingv1.RouteSpec{},
		},
	}

	return ksvc
}

func (r *FunctionReconciler) buildKnativeTrigger(function *functionsv1alpha1.Function) *kneventingv1.Trigger {
	brokerName := "default" // Padrão
	if function.Spec.Eventing.Broker != "" {
		brokerName = function.Spec.Eventing.Broker
	}

	return &kneventingv1.Trigger{
		ObjectMeta: metav1.ObjectMeta{
			Name:      function.Name + "-trigger",
			Namespace: function.Namespace,
		},
		Spec: kneventingv1.TriggerSpec{
			Broker: brokerName,
			// Filtra eventos com base no Spec [8, 56]
			Filter: &kneventingv1.TriggerFilter{
				Attributes: function.Spec.Eventing.Filters,
			},
			// Define o Knative Service como o destino (sink) [9]
			Subscriber: duckv1.Destination{
				Ref: &duckv1.KReference{
					Kind:       "Service",
					Namespace:  function.Namespace,
					Name:       function.Name,
					APIVersion: "serving.knative.dev/v1",
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&functionsv1alpha1.Function{}).
		Owns(&tektonv1.PipelineRun{}).
		Owns(&knservingv1.Service{}).
		Owns(&kneventingv1.Trigger{}).
		Owns(&v1.ServiceAccount{}).
		Named("function").
		Complete(r)
}
