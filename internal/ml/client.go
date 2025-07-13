package ml

import (
	"context"
	"diabetify/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// Enhanced MLClient interface for RabbitMQ-only operations
type MLClient interface {
	// Asynchronous operations (RabbitMQ)
	PredictAsync(ctx context.Context, jobID string, features []float64) error
	HealthCheckAsync(ctx context.Context) error
	// Common
	Close() error
}

// fireAndForgetMLClient implements pure fire-and-forget communication
type fireAndForgetMLClient struct {
	// RabbitMQ components (publishing only)
	rabbitConn    *amqp.Connection
	rabbitChannel *amqp.Channel
	requestQueue  string
	responseQueue string
	healthQueue   string

	closed bool

	// Configuration
	rabbitURL string

	// Debug tracking
	debugEnabled bool
	messagesSent int64
}

// NewFireAndForgetMLClient creates a client that supports fire-and-forget communication
func NewAsyncMLClient(rabbitURL, responseQueue string) (MLClient, error) {
	if responseQueue == "" {
		responseQueue = "ml.prediction.hybrid_response"
	}

	client := &fireAndForgetMLClient{
		rabbitURL:     rabbitURL,
		requestQueue:  "ml.prediction.request",
		responseQueue: responseQueue,
		healthQueue:   "ml.health.request",
		closed:        false,
		debugEnabled:  true,
		messagesSent:  0,
	}

	if err := client.initRabbitMQ(); err != nil {
		return nil, fmt.Errorf("failed to initialize RabbitMQ: %w", err)
	}

	return client, nil
}

func (c *fireAndForgetMLClient) initRabbitMQ() error {
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
			true,  // durable = true
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

// ============ FIRE-AND-FORGET OPERATIONS ============

// SubmitPredictionFireAndForget sends a prediction request and immediately returns
func (c *fireAndForgetMLClient) PredictAsync(ctx context.Context, jobID string, features []float64) error {
	if c.closed || c.rabbitChannel == nil {
		return errors.New("RabbitMQ client not available")
	}

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

	// Fire-and-forget: publish message and return immediately
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
		return fmt.Errorf("failed to publish fire-and-forget request: %w", err)
	}

	c.messagesSent++
	return nil
}

// HealthCheckFireAndForget sends a health check message and returns immediately
func (c *fireAndForgetMLClient) HealthCheckAsync(ctx context.Context) error {
	if c.closed || c.rabbitChannel == nil {
		return errors.New("RabbitMQ client not available")
	}

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

	// Fire-and-forget: publish message and return immediately
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
		return fmt.Errorf("failed to publish fire-and-forget health check: %w", err)
	}

	return nil
}

func (c *fireAndForgetMLClient) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true

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

	if len(errs) > 0 {
		return fmt.Errorf("errors closing fire-and-forget client: %v", errs)
	}

	return nil
}

// ============ VALIDATION ============

func (c *fireAndForgetMLClient) validateFeatures(features []float64) error {
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

func (c *fireAndForgetMLClient) isBinary(val float64) bool {
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
