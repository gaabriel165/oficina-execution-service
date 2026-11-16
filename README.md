# Tech Challenge — Oficina Mecânica · Execution Service

Microsserviço de **execução e produção** da oficina mecânica. Mantém o catálogo de serviços e peças, controla o estoque, gerencia a fila de diagnóstico e de execução e comunica a finalização do reparo.

Parte do Tech Challenge da pós-graduação em Arquitetura de Software (FIAP SOAT) — Fase 4.

## Repositórios do projeto

| Repositório | Papel |
|---|---|
| [`oficina-os-service`](https://github.com/gaabriel165/oficina-os-service) | Ordens de serviço, clientes, veículos e orquestração da saga |
| [`oficina-billing-service`](https://github.com/gaabriel165/oficina-billing-service) | Orçamento e pagamento |
| **`oficina-execution-service`** (este) | Diagnóstico, execução, catálogo e estoque |
| [`oficina-lambda-auth`](https://github.com/gaabriel165/oficina-lambda-auth) | Autenticação por CPF e API Gateway |
| [`oficina-infra-k8s`](https://github.com/gaabriel165/oficina-infra-k8s) | VPC, EKS e base compartilhada |
| [`oficina-infra-db`](https://github.com/gaabriel165/oficina-infra-db) | Banco do OS Service |

## Stack

- Go 1.26, Gin
- Amazon DynamoDB (banco NoSQL exclusivo deste serviço)
- Amazon SQS para comandos e respostas da saga
- OpenTelemetry (traces) e `slog` em JSON (logs)

## Comunicação

O serviço não expõe chamadas síncronas para os outros microsserviços. Ele consome comandos da fila `oficina-execution-commands` e responde na fila `oficina-saga-replies`. As mensagens de saída são gravadas na tabela de outbox na mesma transação da mudança de estado e publicadas em seguida; o consumidor registra cada mensagem processada para ignorar repetições.

Os contratos estão em [`messaging-contracts.md`](https://github.com/gaabriel165/oficina-os-service/blob/main/docs/architecture/messaging-contracts.md).

## Como executar

```bash
cp .env.example .env
docker-compose up --build
```

Sobem a aplicação (porta `8082`), o DynamoDB Local e o ElasticMQ, que emula o SQS localmente. Com `DYNAMODB_CREATE_TABLES=true` as tabelas são criadas no startup.

```bash
curl http://localhost:8082/health
curl http://localhost:8082/ready
```

## Testes

```bash
make test/unit
make test/integration
make test/coverage
```

Os testes de integração usam testcontainers e precisam do Docker em execução.

## Estrutura de pastas

```
cmd/api                  ponto de entrada
configs                  leitura de variáveis de ambiente
internal/domain          entidades, portas e contratos de mensagens
internal/application     casos de uso e relay do outbox
internal/infrastructure  DynamoDB, SQS e observabilidade
internal/api             handlers, middlewares e DTOs
```
