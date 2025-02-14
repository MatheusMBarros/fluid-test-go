# Contagem de Operações com RabbitMQ

Este projeto é um microserviço para contagem de operações (criação, atualização, exclusão) de usuários processadas a partir de mensagens RabbitMQ. O serviço grava as estatísticas de operação em arquivos JSON após um tempo de inatividade ou quando a aplicação é encerrada.

## Pré-requisitos

- Ter o Docker instalado na máquina

- Ter o Go instalado

Antes de rodar o programa, siga os seguintes passos para configurar o ambiente:

### 1. Clonar o Repositório

Primeiro, clone este repositório para sua máquina local onde nele voce ira gerar as mensagems e subir o ambiente do RabbitMQ:

```bash
git clone https://github.com/fluidapi/teste-go-eventcounter.git
cd teste-go-eventcounter
```

### 2. Configuração do Ambiente

Para configurar o ambiente, execute o seguinte comando:

```bash
make env-up
```

Este comando irá subir os contêineres necessários, incluindo RabbitMQ e outros serviços, para garantir que tudo funcione corretamente.

## 3. Gerar os Dados de Publicação (Publish Generator)

Antes de rodar o microserviço, é necessário gerar os dados de publicação. Para isso, execute:

```bash
make publish-generator
```

Este comando irá preparar as mensagens que serão enviadas para o RabbitMQ, simulando eventos de criação, atualização e exclusão.

## Rodando o Microserviço

```bash
git clone
https://github.com/MatheusMBarros/fluid-test-go.git

cd fluid-test-go.
```

Após configurar o ambiente e gerar os dados de publicação, você pode rodar o microserviço de contagem de operações com o seguinte comando:

```bash
make run
```

O serviço ficará ouvindo o RabbitMQ e processará as mensagens. Ele vai contar quantas operações de criação, atualização e exclusão foram feitas para cada usuário e, depois de 5 segundos sem novas mensagens, irá salvar as estatísticas em arquivos JSON.

## Arquivos Gerados

As estatísticas serão salvas na pasta `data`, com o formato de arquivo:

```bash
data/{operation}_stats.json
```

Onde `{operation}` pode ser:

- `created`: Operações de criação
- `updated`: Operações de atualização
- `deleted`: Operações de exclusão

## Finalizando o Serviço

O serviço ficará em execução até que não haja mensagens por 5 segundos ou que seja encerrado manualmente. Após isso, ele irá salvar as estatísticas e finalizar a execução.

## Dependências

- Go 1.18+
- Docker
- RabbitMQ
