package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Estrutura para armazenar contagens por tipo de operação
type OperationCounter struct {
    mu     sync.RWMutex
    counts map[string]map[string]int // Contagem por tipo de operação e usuário
    users  map[string]struct{}      // Armazena usuários processados para evitar duplicação de ID de mensagem
}

func NewOperationCounter() *OperationCounter {
    return &OperationCounter{
        counts: make(map[string]map[string]int),
        users:  make(map[string]struct{}),
    }
}

func (c *OperationCounter) Increment(operation, userID string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.counts[operation] == nil {
        c.counts[operation] = make(map[string]int)
    }
    c.counts[operation][userID]++
}

func (c *OperationCounter) GetCounts() map[string]map[string]int {
    c.mu.RLock()
    defer c.mu.RUnlock()
    countsCopy := make(map[string]map[string]int)
    for op, userCounts := range c.counts {
        countsCopy[op] = make(map[string]int)
        for user, count := range userCounts {
            countsCopy[op][user] = count
        }
    }
    return countsCopy
}

func saveOperationStats(operation string, counts map[string]int) error {
    // Cria o diretório se não existir
    err := os.MkdirAll("data", os.ModePerm)
    if err != nil {
        return fmt.Errorf("erro ao criar pasta 'data': %v", err)
    }
    
    // Prepara os dados para salvar
    stats := struct {
        Timestamp string            `json:"timestamp"`
        Counts    map[string]int    `json:"counts"`
        Operation string            `json:"operation"`
    }{
        Timestamp: time.Now().Format(time.RFC3339),
        Counts:    counts,
        Operation: operation,
    }
    
    // Nome do arquivo baseado na operação
    filename := fmt.Sprintf("data/%s_stats.json", operation)
    
    // Garante permissões adequadas para o arquivo
    f, err := os.Create(filename)
    if err != nil {
        return fmt.Errorf("erro ao criar arquivo %s: %v", filename, err)
    }
    defer f.Close()
    
    // Usa json.NewEncoder para escrever diretamente no arquivo
    encoder := json.NewEncoder(f)
    encoder.SetIndent("", "  ") // Formatação bonita
    
    if err := encoder.Encode(&stats); err != nil {
        return fmt.Errorf("erro ao salvar arquivo %s: %v", filename, err)
    }
    
    log.Printf("Estatísticas de %s salvas em %s", operation, filename)
    return nil
}

func main() {
    // Configuração da conexão RabbitMQ
    conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    ch, err := conn.Channel()
    if err != nil {
        log.Fatal(err)
    }
    defer ch.Close()
    
    // Declaração da fila
    q, err := ch.QueueDeclare(
        "eventcountertest",
        true,   // durable
        false,  // delete when unused
        false,  // exclusive
        false,  // no-wait
        nil,    // arguments
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Inicialização do contador
    counter := NewOperationCounter()
    
    // Canais para os tipos de eventos
    createdChan := make(chan amqp.Delivery)
    updatedChan := make(chan amqp.Delivery)
    deletedChan := make(chan amqp.Delivery)
    
    // Configuração do consumidor
    msgs, err := ch.Consume(
        q.Name,
        "",
        true,    // auto-ack
        false,   // exclusive
        false,   // no-local
        false,   // no-wait
        nil,     // args
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // WaitGroup para garantir que todas as mensagens foram processadas
    var wg sync.WaitGroup
    
    // Canal para controle de inatividade
    inactivityChan := make(chan struct{})
    
    // Adiciona tratamento de sinais para encerramento limpo
    stopChan := make(chan os.Signal, 1)
    signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
    
    // Timer de inatividade com contexto
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        timer := time.NewTimer(5 * time.Second)
        defer timer.Stop()
        
        for {
            select {
            case <-timer.C:
                log.Println("5 segundos sem mensagens recebidas. Encerrando aplicação...")
                
                // Obtém as contagens finais
                counts := counter.GetCounts()
                
                // Salva cada tipo de operação em um arquivo separado
                for operation, userCounts := range counts {
                    if err := saveOperationStats(operation, userCounts); err != nil {
                        log.Printf("Erro ao salvar estatísticas: %v", err)
                    }
                }
                
                // Aguarda todas as goroutines terminarem
                wg.Wait()
                
                // Encerra a execução após salvar os dados
                os.Exit(0)
            
            case <-inactivityChan:
                if !timer.Stop() {
                    <-timer.C
                }
                timer.Reset(5 * time.Second)
                
            case <-ctx.Done():
                return
            }
        }
    }()
    
    // Função para processar mensagens de um canal
    processMessages := func(eventChan chan amqp.Delivery, operation string) {
        defer wg.Done()
        for msg := range eventChan {
            inactivityChan <- struct{}{}
            parts := strings.Split(msg.RoutingKey, ".")
            if len(parts) < 3 {
                log.Printf("Mensagem com routing_key inválido. Ignorando mensagem.")
                continue
            }
            userID := parts[0]
            counter.Increment(operation, userID)
            log.Printf("Processada mensagem do tipo: %s (user: %s, message_id: %s)",
                operation, userID, msg.Body)
        }
    }
    
    // Distribui as mensagens para os canais corretos
    go func() {
        for msg := range msgs {
            switch {
            case strings.HasSuffix(msg.RoutingKey, ".created"):
                createdChan <- msg
            case strings.HasSuffix(msg.RoutingKey, ".updated"):
                updatedChan <- msg
            case strings.HasSuffix(msg.RoutingKey, ".deleted"):
                deletedChan <- msg
            }
        }
    }()
    
    // Inicia as goroutines para processar as mensagens
    wg.Add(3)
    go processMessages(createdChan, "created")
    go processMessages(updatedChan, "updated")
    go processMessages(deletedChan, "deleted")
    
    log.Println("Microserviço de contagem iniciado. Encerrará após 5 segundos sem mensagens.")
    
    // Aguarda sinais de encerramento
    <-stopChan
    log.Println("Encerrando aplicação...")
    
    // Obtém as contagens finais
    counts := counter.GetCounts()
    
    // Salva cada tipo de operação em um arquivo separado
    for operation, userCounts := range counts {
        if err := saveOperationStats(operation, userCounts); err != nil {
            log.Printf("Erro ao salvar estatísticas: %v", err)
        }
    }
    
    // Aguarda todas as goroutines terminarem
    wg.Wait()
    
    // Encerra a execução após salvar os dados
    os.Exit(0)
}