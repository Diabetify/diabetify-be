package ml

import (
	"context"
	"diabetify/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// Enhanced MLClient interface for RabbitMQ-only operations
type MLClient interface {
	// Asynchronous operations (RabbitMQ)
	PredictAsync(ctx context.Context, jobID string, features []float64) error
	GetAsyncResult(ctx context.Context, correlationID string) (*models.PredictionResponse, error)

	// Consumer registration (for job worker)
	RegisterPendingCall(correlationID string, responseCh chan *models.PredictionResponse)
	UnregisterPendingCall(correlationID string)

	// Health check via RabbitMQ
	HealthCheckAsync(ctx context.Context) error

	// Common
	Close() error
}

// asyncMLClient implements RabbitMQ-only communication
type asyncMLClient struct {
	// RabbitMQ components (publishing only)
	rabbitConn    *amqp.Connection
	rabbitChannel *amqp.Channel
	requestQueue  string
	responseQueue string
	healthQueue   string

	// Shared pending calls (used by job worker's consumer)
	pendingCalls map[string]chan *models.PredictionResponse
	pendingMutex sync.RWMutex

	closed     bool
	closeMutex sync.RWMutex

	// Configuration
	rabbitURL string

	// Debug tracking
	debugEnabled bool
	messagesSent int64
}

// NewAsyncMLClient creates a client that supports RabbitMQ-only communication
func NewAsyncMLClient(rabbitURL, responseQueue string) (MLClient, error) {
	if responseQueue == "" {
		responseQueue = "ml.prediction.hybrid_response"
	}

	client := &asyncMLClient{
		rabbitURL:     rabbitURL,
		requestQueue:  "ml.prediction.request",
		responseQueue: responseQueue,
		healthQueue:   "ml.health.request",
		pendingCalls:  make(map[string]chan *models.PredictionResponse),
		closed:        false,
		debugEnabled:  true,
		messagesSent:  0,
	}

	if err := client.initRabbitMQ(); err != nil {
		return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
	}

	return client, nil
}

// NewHybridMLClient creates a backward-compatible async-only client
// This maintains compatibility with existing code that uses the old constructor name
func NewHybridMLClient(grpcAddress, rabbitURL, responseQueue string) (MLClient, error) {
	return NewAsyncMLClient(rabbitURL, responseQueue)
}

// Initialize RabbitMQ connection (publishing only - no consumer)
func (c *asyncMLClient) initRabbitMQ() error {
	if c.rabbitURL == "" {
		c.rabbitURL = "amqp://admin:password123@localhost:5672/"
	}

	conn, err := amqp.Dial(c.rabbitURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ at %s: %w", c.rabbitURL, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	queues := []string{c.requestQueue, c.responseQueue, c.healthQueue, "ml.health.response"}

	for _, queue := range queues {
		_, err := ch.QueueDeclare(
			queue,
			true,  // durable = true (MUST match Python)
			false, // delete when unused = false
			false, // exclusive = false
			false, // no-wait = false
			nil,   // arguments = nil
		)
		if err != nil {
			ch.Close()
			conn.Close()
			return fmt.Errorf("failed to declare queue %s: %w", queue, err)
		}
	}

	c.rabbitConn = conn
	c.rabbitChannel = ch

	return nil
}

// ============ ASYNCHRONOUS OPERATIONS (RabbitMQ) ============

func (c *asyncMLClient) PredictAsync(ctx context.Context, jobID string, features []float64) error {
	c.closeMutex.RLock()
	if c.closed || c.rabbitChannel == nil {
		c.closeMutex.RUnlock()
		return errors.New("RabbitMQ client not available")
	}
	c.closeMutex.RUnlock()

	if err := c.validateFeatures(features); err != nil {
		return err
	}

	correlationID := jobID

	request := PredictionRequest{
		Features:      features,
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	err = c.rabbitChannel.Publish(
		"",
		c.requestQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			CorrelationId: correlationID,
			ReplyTo:       c.responseQueue,
			Timestamp:     time.Now(),
			DeliveryMode:  amqp.Persistent,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish async request: %w", err)
	}

	c.messagesSent++
	return nil
}

func (c *asyncMLClient) GetAsyncResult(ctx context.Context, correlationID string) (*models.PredictionResponse, error) {
	c.closeMutex.RLock()
	if c.closed {
		c.closeMutex.RUnlock()
		return nil, errors.New("client is closed")
	}
	c.closeMutex.RUnlock()

	c.pendingMutex.RLock()
	responseCh, exists := c.pendingCalls[correlationID]
	c.pendingMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no pending call registered for correlation ID %s", correlationID)
	}

	timeout := 2 * time.Second

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("async result not ready yet for correlation ID %s", correlationID)
	case response := <-responseCh:
		if response == nil {
			return nil, fmt.Errorf("received nil response for correlation ID %s", correlationID)
		}
		return response, nil
	}
}

// HealthCheckAsync performs health check via RabbitMQ (async operation)
func (c *asyncMLClient) HealthCheckAsync(ctx context.Context) error {
	c.closeMutex.RLock()
	if c.closed || c.rabbitChannel == nil {
		c.closeMutex.RUnlock()
		return errors.New("RabbitMQ client not available")
	}
	c.closeMutex.RUnlock()

	correlationID := fmt.Sprintf("health_%d", time.Now().UnixNano())
	responseQueue := "ml.health.response"

	request := HealthCheckRequest{
		CorrelationID: correlationID,
		Timestamp:     time.Now(),
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal health check request: %w", err)
	}

	err = c.rabbitChannel.Publish(
		"",
		c.healthQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			CorrelationId: correlationID,
			ReplyTo:       responseQueue,
			Timestamp:     time.Now(),
			DeliveryMode:  amqp.Persistent,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish health check: %w", err)
	}

	return nil
}

// ============ CONSUMER REGISTRATION (for job worker) ============

func (c *asyncMLClient) RegisterPendingCall(correlationID string, responseCh chan *models.PredictionResponse) {
	c.pendingMutex.Lock()
	c.pendingCalls[correlationID] = responseCh
	c.pendingMutex.Unlock()
}

func (c *asyncMLClient) UnregisterPendingCall(correlationID string) {
	c.pendingMutex.Lock()
	delete(c.pendingCalls, correlationID)
	c.pendingMutex.Unlock()
}

// DeliverResponse delivers a response to a waiting goroutine (used by unified consumer)
func (c *asyncMLClient) DeliverResponse(correlationID string, response *models.PredictionResponse) bool {
	c.pendingMutex.RLock()
	responseCh, exists := c.pendingCalls[correlationID]
	c.pendingMutex.RUnlock()

	if !exists {
		return false
	}

	select {
	case responseCh <- response:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// ============ CLEANUP ============

func (c *asyncMLClient) Close() error {
	c.closeMutex.Lock()
	if c.closed {
		c.closeMutex.Unlock()
		return nil
	}
	c.closed = true
	c.closeMutex.Unlock()

	var errs []error

	if c.rabbitChannel != nil {
		if err := c.rabbitChannel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close RabbitMQ channel: %w", err))
		}
	}

	if c.rabbitConn != nil {
		if err := c.rabbitConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close RabbitMQ connection: %w", err))
		}
	}

	c.pendingMutex.Lock()
	for _, ch := range c.pendingCalls {
		close(ch)
	}
	c.pendingCalls = make(map[string]chan *models.PredictionResponse)
	c.pendingMutex.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("errors closing async client: %v", errs)
	}

	return nil
}

// ============ SHARED VALIDATION ============

func (c *asyncMLClient) validateFeatures(features []float64) error {
	if len(features) != 9 {
		return errors.New("incorrect number of features: expected 9")
	}
	if features[0] <= 0 {
		return errors.New("age must be positive")
	}
	if features[1] < 0 || features[1] > 2 {
		return errors.New("smoking status must be 0, 1, or 2")
	}
	if !c.isBinary(features[2]) {
		return errors.New("cholesterol status must be 0 or 1")
	}
	if features[3] < 0 || features[3] > 2 {
		return errors.New("macrosomic baby must be 0, 1, or 2")
	}
	if features[4] < 0 {
		return errors.New("physical activity frequency cannot be negative")
	}
	if !c.isBinary(features[5]) {
		return errors.New("bloodline status must be 0 or 1")
	}
	if features[6] < 0 || features[6] > 3 {
		return errors.New("brinkman index must be between 0 and 3")
	}
	if features[7] < 10 || features[7] > 60 {
		return errors.New("BMI out of typical range (10-60)")
	}
	if !c.isBinary(features[8]) {
		return errors.New("hypertension status must be 0 or 1")
	}
	return nil
}

func (c *asyncMLClient) isBinary(val float64) bool {
	return val == 0 || val == 1
}

// ============ MESSAGE TYPES ============

type PredictionRequest struct {
	Features      []float64 `json:"features"`
	CorrelationID string    `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
}

type FlexibleTime struct {
	time.Time
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	str := string(b)
	if str == "null" {
		ft.Time = time.Time{}
		return nil
	}

	str = str[1 : len(str)-1]

	formats := []string{
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}

	var err error
	for _, format := range formats {
		ft.Time, err = time.Parse(format, str)
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("unable to parse timestamp %s with any known format", str)
}

type PredictionResponse struct {
	Prediction    float64                           `json:"prediction"`
	Explanation   map[string]models.ExplanationItem `json:"explanation"`
	ElapsedTime   float64                           `json:"elapsed_time"`
	Timestamp     FlexibleTime                      `json:"timestamp"`
	CorrelationID string                            `json:"correlation_id"`
	Error         *string                           `json:"error"`
}

type HealthCheckRequest struct {
	CorrelationID string    `json:"correlation_id"`
	Timestamp     time.Time `json:"timestamp"`
}

type HealthCheckResponse struct {
	Status        string       `json:"status"`
	Timestamp     FlexibleTime `json:"timestamp"`
	CorrelationID string       `json:"correlation_id"`
	Error         *string      `json:"error"`
}
