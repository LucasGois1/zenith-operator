# Zenith Operator - Comprehensive Architecture Diagram

## Overview

Zenith Operator is a Kubernetes operator that provides a serverless function platform by orchestrating builds, deployments, and event-driven invocations. It abstracts away the complexity of integrating Tekton Pipelines (for builds), Knative Serving (for deployments), and Knative Eventing (for event routing) behind a simple `Function` custom resource.

---

## 1. High-Level Architecture

```mermaid
graph TB
    subgraph "Developer Experience"
        DEV[Developer]
        FUNC_YAML[Function CR YAML]
        DEV -->|1. Creates| FUNC_YAML
    end
    
    subgraph "Kubernetes Cluster"
        subgraph "Zenith Operator System"
            OPERATOR[Zenith Operator<br/>Controller Manager]
            CRD[Function CRD<br/>functions.zenith.com/v1alpha1]
        end
        
        subgraph "Build System - Tekton"
            PIPELINERUN[PipelineRun]
            TASK_CLONE[Task: git-clone]
            TASK_BUILD[Task: buildpacks-phases]
            WORKSPACE[Shared Workspace<br/>emptyDir]
        end
        
        subgraph "Deployment System - Knative Serving"
            KSVC[Knative Service]
            REVISION[Revision]
            ROUTE[Route]
            POD[Pod with Function]
            DAPR[Dapr Sidecar<br/>optional]
        end
        
        subgraph "Event System - Knative Eventing"
            BROKER[Event Broker]
            TRIGGER[Trigger<br/>with filters]
        end
        
        subgraph "External Resources"
            GIT[Git Repository<br/>Function Source Code]
            REGISTRY[Container Registry<br/>Docker Hub, etc.]
        end
    end
    
    FUNC_YAML -->|2. kubectl apply| CRD
    CRD -->|3. Reconcile Event| OPERATOR
    
    OPERATOR -->|4. Creates| PIPELINERUN
    PIPELINERUN -->|5. Executes| TASK_CLONE
    TASK_CLONE -->|6. Clones from| GIT
    TASK_CLONE -->|7. Writes to| WORKSPACE
    WORKSPACE -->|8. Reads from| TASK_BUILD
    TASK_BUILD -->|9. Builds & Pushes| REGISTRY
    TASK_BUILD -->|10. Returns digest| OPERATOR
    
    OPERATOR -->|11. Creates/Updates| KSVC
    KSVC -->|12. Creates| REVISION
    REVISION -->|13. Deploys| POD
    POD -.->|14. Optional| DAPR
    KSVC -->|15. Exposes via| ROUTE
    
    OPERATOR -->|16. Creates optional| TRIGGER
    TRIGGER -->|17. Subscribes to| BROKER
    BROKER -->|18. Routes events to| KSVC
    
    OPERATOR -->|19. Updates Status| CRD
    
    style OPERATOR fill:#4CAF50,stroke:#2E7D32,stroke-width:3px,color:#fff
    style CRD fill:#2196F3,stroke:#1565C0,stroke-width:2px,color:#fff
    style FUNC_YAML fill:#FF9800,stroke:#E65100,stroke-width:2px,color:#fff
    style KSVC fill:#9C27B0,stroke:#6A1B9A,stroke-width:2px,color:#fff
    style PIPELINERUN fill:#F44336,stroke:#C62828,stroke-width:2px,color:#fff
```

---

## 2. Function Custom Resource Structure

```mermaid
classDiagram
    class Function {
        +TypeMeta
        +ObjectMeta
        +FunctionSpec spec
        +FunctionStatus status
    }
    
    class FunctionSpec {
        +string gitRepo
        +string gitRevision
        +BuildSpec build
        +DeploySpec deploy
        +EventingSpec eventing
    }
    
    class BuildSpec {
        +string registrySecretName
        +string image
    }
    
    class DeploySpec {
        +DaprConfig dapr
    }
    
    class DaprConfig {
        +bool enabled
        +string appID
        +int appPort
    }
    
    class EventingSpec {
        +string broker
        +map~string,string~ filters
    }
    
    class FunctionStatus {
        +Condition[] conditions
        +string imageDigest
        +string url
        +int64 observedGeneration
    }
    
    Function --> FunctionSpec
    Function --> FunctionStatus
    FunctionSpec --> BuildSpec
    FunctionSpec --> DeploySpec
    FunctionSpec --> EventingSpec
    DeploySpec --> DaprConfig
```

---

## 3. Operator Reconciliation Flow

```mermaid
sequenceDiagram
    participant K8s as Kubernetes API
    participant Op as Zenith Operator
    participant SA as ServiceAccount
    participant PR as PipelineRun
    participant KS as Knative Service
    participant TR as Trigger
    participant Reg as Container Registry
    
    K8s->>Op: Function CR Created/Updated
    
    rect rgb(255, 240, 240)
        Note over Op,SA: Phase 1: ServiceAccount Setup
        Op->>SA: Get ServiceAccount "default"
        Op->>SA: Add imagePullSecret reference
        Op->>K8s: Update ServiceAccount
        Op->>K8s: Update Status: "Configuring"
    end
    
    rect rgb(240, 255, 240)
        Note over Op,PR: Phase 2: Build Pipeline
        Op->>PR: Create PipelineRun
        Op->>K8s: Update Status: "Building"
        
        PR->>PR: Execute git-clone task
        PR->>PR: Execute buildpacks-phases task
        PR->>Reg: Push image
        PR->>Op: Return image digest
        
        Op->>K8s: Update Status: "BuildSucceeded"<br/>Set imageDigest
    end
    
    rect rgb(240, 240, 255)
        Note over Op,KS: Phase 3: Deployment
        Op->>KS: Create/Update Knative Service
        Note over KS: - Set image to digest<br/>- Add Dapr annotations if enabled<br/>- Configure scaling
        KS->>KS: Create Revision
        KS->>KS: Deploy Pods
        KS->>Op: Return public URL
        Op->>K8s: Update Status: set URL
    end
    
    rect rgb(255, 255, 240)
        Note over Op,TR: Phase 4: Event Subscription (Optional)
        alt Eventing configured
            Op->>TR: Create Trigger
            Note over TR: - Subscribe to Broker<br/>- Apply filters<br/>- Route to Knative Service
            Op->>K8s: Update Status: "Ready"
        else No eventing
            Op->>K8s: Update Status: "Ready"
        end
    end
```

---

## 4. Complete Developer Experience Flow

```mermaid
graph TB
    subgraph "Step 1: Environment Setup"
        A1[Install Kubernetes Cluster<br/>kind/minikube/GKE/EKS]
        A2[Install Tekton Pipelines<br/>kubectl apply -f tekton.yaml]
        A3[Install Knative Serving<br/>kubectl apply -f knative-serving.yaml]
        A4[Install Knative Eventing<br/>optional]
        A5[Install Envoy Gateway<br/>for Gateway API]
        A6[Install Dapr<br/>optional]
        
        A1 --> A2
        A2 --> A3
        A3 --> A4
        A4 --> A5
        A5 --> A6
    end
    
    subgraph "Step 2: Install Zenith Operator"
        B1[Build Operator Image<br/>make docker-build IMG=...]
        B2[Push to Registry<br/>make docker-push IMG=...]
        B3[Install CRDs<br/>make install]
        B4[Deploy Operator<br/>make deploy IMG=...]
        B5[Verify Operator Running<br/>kubectl get pods -n zenith-operator-system]
        
        B1 --> B2
        B2 --> B3
        B3 --> B4
        B4 --> B5
    end
    
    subgraph "Step 3: Prepare Function Code"
        C1[Create Git Repository<br/>with function code]
        C2[Add Procfile or<br/>buildpack configuration]
        C3[Push to Git<br/>GitHub/GitLab/etc]
        
        C1 --> C2
        C2 --> C3
    end
    
    subgraph "Step 4: Configure Secrets"
        D1[Create Registry Secret<br/>kubectl create secret docker-registry]
        D2[Create Git Auth Secret<br/>optional, for private repos]
        
        D1 --> D2
    end
    
    subgraph "Step 5: Deploy Function"
        E1[Create Function YAML<br/>Define gitRepo, build, deploy, eventing]
        E2[Apply Function CR<br/>kubectl apply -f function.yaml]
        E3[Operator Reconciles<br/>Creates PipelineRun]
        E4[Build Executes<br/>git-clone + buildpacks]
        E5[Image Pushed<br/>to registry with digest]
        E6[Knative Service Created<br/>Deploys function pods]
        E7[Route Exposed<br/>Public URL available]
        E8[Trigger Created<br/>optional, for events]
        
        E1 --> E2
        E2 --> E3
        E3 --> E4
        E4 --> E5
        E5 --> E6
        E6 --> E7
        E7 --> E8
    end
    
    subgraph "Step 6: Function Ready"
        F1[Check Status<br/>kubectl get function]
        F2[View URL<br/>status.url field]
        F3[Test Function<br/>curl URL or send events]
        F4[Monitor Logs<br/>kubectl logs]
        F5[Auto-scaling Active<br/>scale-to-zero enabled]
        
        F1 --> F2
        F2 --> F3
        F3 --> F4
        F4 --> F5
    end
    
    A6 --> B1
    B5 --> C1
    C3 --> D1
    D2 --> E1
    E8 --> F1
    
    style A1 fill:#E3F2FD,stroke:#1976D2,stroke-width:2px
    style B1 fill:#E8F5E9,stroke:#388E3C,stroke-width:2px
    style C1 fill:#FFF3E0,stroke:#F57C00,stroke-width:2px
    style D1 fill:#FCE4EC,stroke:#C2185B,stroke-width:2px
    style E1 fill:#F3E5F5,stroke:#7B1FA2,stroke-width:2px
    style F1 fill:#E0F2F1,stroke:#00796B,stroke-width:2px
```

---

## 5. Key Features

```mermaid
mindmap
    root((Zenith Operator<br/>Features))
        Build System
            Git Integration
                Clone from any Git repo
                Support branches/tags/commits
                Private repo authentication
            Cloud Native Buildpacks
                Auto-detect language
                No Dockerfile needed
                Optimized layers
            Registry Management
                Push to any registry
                Immutable digests
                Secret-based auth
        
        Deployment System
            Knative Serving
                Auto-scaling
                Scale-to-zero
                Traffic splitting
                Blue-green deployments
            Dapr Integration
                Service mesh
                Pub/sub
                State management
                Service discovery
            Resource Management
                CPU/Memory limits
                Concurrency control
                Timeout configuration
        
        Event System
            Knative Eventing
                Event-driven architecture
                Broker subscription
                Attribute filtering
                CloudEvents support
            Trigger Management
                Automatic creation
                Filter configuration
                Multiple subscriptions
        
        Developer Experience
            Simple API
                Single CR definition
                Declarative configuration
                GitOps friendly
            Status Tracking
                Build progress
                Deployment status
                Public URL
                Error reporting
            Lifecycle Management
                Automatic updates
                OwnerReference cleanup
                Reconciliation loops
```

---

## 6. Tekton Build Pipeline Details

```mermaid
graph LR
    subgraph "PipelineRun Execution"
        START[PipelineRun Created]
        
        subgraph "Task 1: git-clone"
            T1_START[Start git-clone]
            T1_CLONE[Clone repository<br/>url: spec.gitRepo<br/>revision: spec.gitRevision]
            T1_OUTPUT[Write to workspace]
            T1_END[Complete]
            
            T1_START --> T1_CLONE
            T1_CLONE --> T1_OUTPUT
            T1_OUTPUT --> T1_END
        end
        
        subgraph "Task 2: buildpacks-phases"
            T2_START[Start buildpacks]
            T2_DETECT[Detect buildpacks<br/>analyze source code]
            T2_BUILD[Build image layers<br/>compile/package]
            T2_EXPORT[Export image]
            T2_PUSH[Push to registry<br/>spec.build.image]
            T2_DIGEST[Return digest<br/>APP_IMAGE_DIGEST]
            T2_END[Complete]
            
            T2_START --> T2_DETECT
            T2_DETECT --> T2_BUILD
            T2_BUILD --> T2_EXPORT
            T2_EXPORT --> T2_PUSH
            T2_PUSH --> T2_DIGEST
            T2_DIGEST --> T2_END
        end
        
        START --> T1_START
        T1_END -->|Shared Workspace| T2_START
        T2_END --> FINISH[PipelineRun Succeeded]
    end
    
    subgraph "Authentication"
        SA[ServiceAccount: default]
        SECRET[Secret: registrySecretName]
        
        SA -.->|imagePullSecrets| SECRET
        SA -.->|Used by| START
    end
    
    style START fill:#4CAF50,stroke:#2E7D32,stroke-width:2px,color:#fff
    style FINISH fill:#4CAF50,stroke:#2E7D32,stroke-width:2px,color:#fff
    style T1_CLONE fill:#2196F3,stroke:#1565C0,stroke-width:2px,color:#fff
    style T2_BUILD fill:#FF9800,stroke:#E65100,stroke-width:2px,color:#fff
    style T2_PUSH fill:#F44336,stroke:#C62828,stroke-width:2px,color:#fff
```

---

## 7. Knative Service Deployment Details

```mermaid
graph TB
    subgraph "Knative Service Creation"
        KSVC_CREATE[Knative Service Created]
        
        subgraph "Configuration"
            CONFIG[ConfigurationSpec]
            TEMPLATE[RevisionTemplateSpec]
            PODSPEC[PodSpec]
            CONTAINER[Container]
            
            CONFIG --> TEMPLATE
            TEMPLATE --> PODSPEC
            PODSPEC --> CONTAINER
        end
        
        subgraph "Container Configuration"
            IMAGE[Image: status.imageDigest<br/>immutable reference]
            PORT[Port: spec.deploy.dapr.appPort]
            RESOURCES[Resources: CPU/Memory]
            
            CONTAINER --> IMAGE
            CONTAINER --> PORT
            CONTAINER --> RESOURCES
        end
        
        subgraph "Dapr Integration (Optional)"
            DAPR_ENABLED{Dapr Enabled?}
            DAPR_ANNOTATIONS[Pod Annotations:<br/>dapr.io/enabled: true<br/>dapr.io/app-id: appID<br/>dapr.io/app-port: appPort]
            DAPR_SIDECAR[Dapr Sidecar Injected]
            
            DAPR_ENABLED -->|Yes| DAPR_ANNOTATIONS
            DAPR_ANNOTATIONS --> DAPR_SIDECAR
        end
        
        subgraph "Revision Management"
            REVISION[Revision Created<br/>immutable snapshot]
            DEPLOYMENT[Deployment Created]
            PODS[Pods Deployed]
            
            REVISION --> DEPLOYMENT
            DEPLOYMENT --> PODS
        end
        
        subgraph "Routing"
            ROUTE[Route Created]
            INGRESS[Ingress/Gateway]
            PUBLIC_URL[Public URL<br/>status.url]
            
            ROUTE --> INGRESS
            INGRESS --> PUBLIC_URL
        end
        
        KSVC_CREATE --> CONFIG
        TEMPLATE --> DAPR_ENABLED
        PODSPEC --> REVISION
        PODS -.-> DAPR_SIDECAR
        REVISION --> ROUTE
    end
    
    subgraph "Auto-Scaling"
        AUTOSCALER[Knative Autoscaler]
        METRICS[Monitor Metrics<br/>requests/concurrency]
        SCALE_UP[Scale Up Pods]
        SCALE_ZERO[Scale to Zero<br/>after idle period]
        
        AUTOSCALER --> METRICS
        METRICS --> SCALE_UP
        METRICS --> SCALE_ZERO
        
        AUTOSCALER -.->|Controls| PODS
    end
    
    style KSVC_CREATE fill:#9C27B0,stroke:#6A1B9A,stroke-width:3px,color:#fff
    style IMAGE fill:#4CAF50,stroke:#2E7D32,stroke-width:2px,color:#fff
    style DAPR_SIDECAR fill:#FF9800,stroke:#E65100,stroke-width:2px,color:#fff
    style PUBLIC_URL fill:#2196F3,stroke:#1565C0,stroke-width:2px,color:#fff
```

---

## 8. Event-Driven Architecture (Optional)

```mermaid
graph TB
    subgraph "Event Sources"
        SOURCE1[HTTP Source]
        SOURCE2[Kafka Source]
        SOURCE3[Custom Source]
    end
    
    subgraph "Knative Eventing"
        BROKER[Event Broker<br/>spec.eventing.broker]
        
        subgraph "Trigger Configuration"
            TRIGGER[Trigger<br/>function-name-trigger]
            FILTERS[Attribute Filters<br/>spec.eventing.filters]
            
            TRIGGER --> FILTERS
        end
        
        subgraph "Event Routing"
            FILTER_ENGINE[Filter Engine]
            MATCH{Filters Match?}
            
            FILTER_ENGINE --> MATCH
        end
    end
    
    subgraph "Function Deployment"
        KSVC[Knative Service<br/>function-name]
        POD[Function Pod]
        HANDLER[Event Handler Code]
        
        KSVC --> POD
        POD --> HANDLER
    end
    
    subgraph "Event Processing"
        RECEIVE[Receive CloudEvent]
        PROCESS[Process Event]
        RESPOND[Send Response]
        
        RECEIVE --> PROCESS
        PROCESS --> RESPOND
    end
    
    SOURCE1 --> BROKER
    SOURCE2 --> BROKER
    SOURCE3 --> BROKER
    
    BROKER --> TRIGGER
    FILTERS --> FILTER_ENGINE
    
    MATCH -->|Yes| KSVC
    MATCH -->|No| DROP[Drop Event]
    
    KSVC --> RECEIVE
    RESPOND --> BROKER
    
    style BROKER fill:#FF9800,stroke:#E65100,stroke-width:3px,color:#fff
    style TRIGGER fill:#9C27B0,stroke:#6A1B9A,stroke-width:2px,color:#fff
    style KSVC fill:#4CAF50,stroke:#2E7D32,stroke-width:2px,color:#fff
    style MATCH fill:#F44336,stroke:#C62828,stroke-width:2px,color:#fff
```

---

## 9. Status Conditions and Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Configuring: Function CR Created
    
    Configuring --> Building: ServiceAccount Updated
    
    Building --> BuildSucceeded: PipelineRun Succeeded
    Building --> BuildFailed: PipelineRun Failed
    
    BuildFailed --> [*]: Manual Intervention Required
    
    BuildSucceeded --> Deploying: Knative Service Created
    
    Deploying --> Ready: Service Ready & URL Available
    Deploying --> DeployFailed: Service Creation Failed
    
    DeployFailed --> [*]: Manual Intervention Required
    
    Ready --> Updating: Spec Changed
    Ready --> [*]: Function Deleted
    
    Updating --> Building: New Build Required
    Updating --> Deploying: Only Deploy Changed
    
    note right of Configuring
        Status:
        - Type: Ready
        - Status: False
        - Reason: Configuring
    end note
    
    note right of Building
        Status:
        - Type: Ready
        - Status: False
        - Reason: Building
        - Message: Pipeline running
    end note
    
    note right of BuildSucceeded
        Status:
        - Type: Ready
        - Status: False
        - Reason: BuildSucceeded
        - imageDigest: set
    end note
    
    note right of Ready
        Status:
        - Type: Ready
        - Status: True
        - Reason: Ready
        - imageDigest: set
        - url: set
    end note
```

---

## 10. Example Function Definition

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: hello-world
  namespace: default
spec:
  # Git repository containing function source code
  gitRepo: https://github.com/LucasGois1/zenith-test-functions
  gitRevision: main
  
  # Build configuration
  build:
    # Container registry secret for authentication
    registrySecretName: registry-credentials
    # Target image name (digest will be appended)
    image: docker.io/myorg/hello-world
  
  # Deployment configuration
  deploy:
    # Optional Dapr sidecar injection
    dapr:
      enabled: true
      appID: hello-world
      appPort: 8080
  
  # Optional event subscription
  eventing:
    # Knative Broker to subscribe to
    broker: default
    # CloudEvents attribute filters
    filters:
      type: com.example.greeting
      source: greeting-service

---
# After reconciliation, status will be:
status:
  conditions:
  - type: Ready
    status: "True"
    reason: Ready
    message: Deployed and ready to accept requests
  imageDigest: docker.io/myorg/hello-world@sha256:abc123...
  url: https://hello-world.default.example.com
  observedGeneration: 1
```

---

## 11. Operator Components and RBAC

```mermaid
graph TB
    subgraph "Operator Deployment"
        MANAGER[Controller Manager Pod<br/>zenith-operator-controller-manager]
        
        subgraph "Controllers"
            FUNC_CTRL[Function Controller<br/>FunctionReconciler]
            WATCH[Watch Resources]
            RECONCILE[Reconciliation Loop]
            
            FUNC_CTRL --> WATCH
            FUNC_CTRL --> RECONCILE
        end
        
        subgraph "Webhooks (Optional)"
            VALIDATING[Validating Webhook]
            MUTATING[Mutating Webhook]
        end
        
        MANAGER --> FUNC_CTRL
        MANAGER -.-> VALIDATING
        MANAGER -.-> MUTATING
    end
    
    subgraph "RBAC Permissions"
        SA_OP[ServiceAccount<br/>zenith-operator-controller-manager]
        ROLE[ClusterRole<br/>zenith-operator-manager-role]
        BINDING[ClusterRoleBinding]
        
        SA_OP --> BINDING
        ROLE --> BINDING
        
        subgraph "Permissions"
            PERM1[functions.zenith.com/*<br/>get, list, watch, create, update, patch, delete]
            PERM2[tekton.dev/pipelineruns<br/>get, list, watch, create, update, patch, delete]
            PERM3[serving.knative.dev/services<br/>get, list, watch, create, update, patch, delete]
            PERM4[eventing.knative.dev/triggers<br/>get, list, watch, create, update, patch, delete]
            PERM5[core/serviceaccounts<br/>get, list, watch, update, patch]
            PERM6[core/secrets<br/>get, list, watch]
        end
        
        ROLE --> PERM1
        ROLE --> PERM2
        ROLE --> PERM3
        ROLE --> PERM4
        ROLE --> PERM5
        ROLE --> PERM6
    end
    
    MANAGER --> SA_OP
    
    subgraph "Observability"
        METRICS[Metrics Server<br/>:8080/metrics]
        HEALTH[Health Probes<br/>:8081/healthz, /readyz]
        LOGS[Structured Logging]
        
        MANAGER --> METRICS
        MANAGER --> HEALTH
        MANAGER --> LOGS
    end
    
    style MANAGER fill:#4CAF50,stroke:#2E7D32,stroke-width:3px,color:#fff
    style FUNC_CTRL fill:#2196F3,stroke:#1565C0,stroke-width:2px,color:#fff
    style ROLE fill:#FF9800,stroke:#E65100,stroke-width:2px,color:#fff
```

---

## 12. Integration Points Summary

| Component | Purpose | Integration Method |
|-----------|---------|-------------------|
| **Tekton Pipelines** | Build container images from Git repos | Creates PipelineRun resources with git-clone and buildpacks tasks |
| **Knative Serving** | Deploy and scale functions | Creates Service resources with auto-scaling and routing |
| **Knative Eventing** | Event-driven invocations | Creates Trigger resources for broker subscriptions |
| **Dapr** | Service mesh capabilities | Injects sidecar via pod annotations |
| **Kong/Gateway API** | Ingress and routing | Used by Knative for external access |
| **Container Registry** | Store function images | Authenticated via Kubernetes secrets |
| **Git Repositories** | Source code storage | Cloned by Tekton git-clone task |

---

## 13. Key Design Principles

1. **Declarative API**: Single Function CR defines entire lifecycle
2. **GitOps Friendly**: All configuration in version control
3. **Immutable Deployments**: Uses image digests for reproducibility
4. **Owner References**: Automatic cleanup of child resources
5. **Status Conditions**: Standard Kubernetes condition reporting
6. **Reconciliation Loops**: Continuous state synchronization
7. **Extensibility**: Pluggable build and deployment systems
8. **Security**: Secret-based authentication, non-root containers
9. **Observability**: Metrics, health checks, structured logging
10. **Developer Experience**: Simple API, automatic builds, auto-scaling

---

## 14. Troubleshooting Flow

```mermaid
graph TD
    START[Function Not Working]
    
    CHECK_STATUS{Check Function Status}
    
    STATUS_BUILDING[Status: Building]
    STATUS_FAILED[Status: BuildFailed]
    STATUS_READY[Status: Ready]
    STATUS_UNKNOWN[Status: Unknown]
    
    CHECK_STATUS --> STATUS_BUILDING
    CHECK_STATUS --> STATUS_FAILED
    CHECK_STATUS --> STATUS_READY
    CHECK_STATUS --> STATUS_UNKNOWN
    
    STATUS_BUILDING --> CHECK_PR[Check PipelineRun<br/>kubectl get pipelinerun]
    CHECK_PR --> PR_LOGS[View PipelineRun logs<br/>kubectl logs]
    
    STATUS_FAILED --> CHECK_PR_FAILED[Check PipelineRun failure<br/>kubectl describe pipelinerun]
    CHECK_PR_FAILED --> FIX_BUILD[Fix build issues:<br/>- Git auth<br/>- Registry auth<br/>- Buildpack errors]
    
    STATUS_READY --> CHECK_URL{URL accessible?}
    CHECK_URL -->|No| CHECK_KSVC[Check Knative Service<br/>kubectl get ksvc]
    CHECK_KSVC --> CHECK_PODS[Check Pods<br/>kubectl get pods]
    CHECK_PODS --> POD_LOGS[View Pod logs<br/>kubectl logs]
    
    CHECK_URL -->|Yes| CHECK_EVENTS{Events working?}
    CHECK_EVENTS -->|No| CHECK_TRIGGER[Check Trigger<br/>kubectl get trigger]
    CHECK_TRIGGER --> CHECK_BROKER[Check Broker<br/>kubectl get broker]
    
    STATUS_UNKNOWN --> CHECK_OPERATOR[Check Operator logs<br/>kubectl logs -n zenith-operator-system]
    CHECK_OPERATOR --> CHECK_RBAC[Verify RBAC permissions]
    
    style START fill:#F44336,stroke:#C62828,stroke-width:3px,color:#fff
    style STATUS_READY fill:#4CAF50,stroke:#2E7D32,stroke-width:2px,color:#fff
    style STATUS_FAILED fill:#F44336,stroke:#C62828,stroke-width:2px,color:#fff
```

---

## Summary

The Zenith Operator provides a complete serverless function platform on Kubernetes by:

1. **Abstracting Complexity**: Single Function CR instead of managing multiple resources
2. **Automating Builds**: Tekton pipelines with Cloud Native Buildpacks
3. **Enabling Scale**: Knative Serving with auto-scaling and scale-to-zero
4. **Supporting Events**: Knative Eventing for event-driven architectures
5. **Integrating Service Mesh**: Optional Dapr sidecar injection
6. **Ensuring Security**: Secret-based authentication and RBAC
7. **Providing Observability**: Status conditions, metrics, and logs
8. **Simplifying Operations**: Automatic reconciliation and lifecycle management

The operator follows Kubernetes best practices and provides a production-ready platform for deploying and managing serverless functions at scale.
