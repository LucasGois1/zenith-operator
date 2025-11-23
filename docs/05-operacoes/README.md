# Operações

Configuração e gerenciamento do Zenith Operator em ambientes de produção.

## Conteúdo

### [Helm Chart](helm-chart.md)
Documentação completa do Helm chart para instalação do Zenith Operator e stack completa.

**Tópicos abordados:**
- Instalação via Helm
- Configuração de values
- Profiles (standard vs dev)
- Componentes da stack (Tekton, Knative, Envoy Gateway)
- Customização e troubleshooting

**Use este documento quando:**
- Estiver instalando o operator pela primeira vez
- Precisar configurar o ambiente de produção
- Quiser entender os componentes da stack
- Precisar customizar a instalação

### [Configuração de Registry](configuracao-registry.md)
Guia completo para configurar container registries em produção.

**Tópicos abordados:**
- Detecção automática de registries inseguros
- Docker Hub (recomendado para começar)
- Registries customizados (Harbor, Nexus, ECR, GCR)
- Registry in-cluster para produção
- Autenticação e secrets
- Troubleshooting de problemas de registry

**Use este documento quando:**
- Estiver configurando um registry de produção
- Tiver problemas com push/pull de imagens
- Quiser usar um registry privado
- Precisar configurar autenticação de registry

## Configuração de Produção

### Checklist de Produção

Antes de usar o Zenith Operator em produção, certifique-se de:

- [ ] **Cluster Kubernetes** configurado e estável
- [ ] **Tekton Pipelines** instalado e funcionando
- [ ] **Knative Serving** instalado com Gateway configurado
- [ ] **Container Registry** configurado com autenticação
- [ ] **Secrets de Git** criados para repositórios privados
- [ ] **Monitoring** configurado (Prometheus, Grafana)
- [ ] **Logging** centralizado configurado
- [ ] **Backup** e disaster recovery planejados
- [ ] **RBAC** configurado adequadamente
- [ ] **Network Policies** aplicadas conforme necessário

### Boas Práticas

1. **Use HTTPS** para todos os registries em produção
2. **Rotacione credenciais** regularmente (Git tokens, registry passwords)
3. **Configure resource limits** para funções
4. **Implemente monitoring** e alertas
5. **Use namespaces separados** para diferentes ambientes
6. **Configure backup** de Functions CRs
7. **Documente** suas configurações customizadas
8. **Teste** em staging antes de produção

### Ambientes

Recomendamos manter ambientes separados:

- **Development**: Cluster local (kind/minikube) com profile `dev`
- **Staging**: Cluster dedicado com configuração similar à produção
- **Production**: Cluster dedicado com alta disponibilidade e monitoring

## Próximos Passos

- **[Guias](../02-guias/)** - Aprenda a criar funções
- **[Referência](../04-referencia/)** - Consulte a API completa
- **[Troubleshooting](../04-referencia/troubleshooting.md)** - Resolva problemas
