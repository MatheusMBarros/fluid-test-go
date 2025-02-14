# Variáveis
APP_NAME = fluid-test-go
SRC = main.go
BINARY = $(APP_NAME)

default: build

build:
	@echo "🔧 Compilando o projeto..."
	go build -o $(BINARY) $(SRC)

run: build
	@echo "🚀 Executando o projeto..."
	./$(BINARY)


install-deps:
	@echo "📦 Instalando dependências..."
	go mod tidy

run-rabbitmq:
	@echo "📡 Iniciando RabbitMQ..."
	docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:management

stop-rabbitmq:
	@echo "🛑 Parando RabbitMQ..."
	docker stop rabbitmq && docker rm rabbitmq

.PHONY: default build run test fmt clean lint install-deps run-rabbitmq stop-rabbitmq
