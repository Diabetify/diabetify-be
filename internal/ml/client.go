package ml

import (
	"context"
	"diabetify/internal/models"
	pb "diabetify/internal/proto/prediction"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// grpcMLClient implements MLClient using gRPC
type grpcMLClient struct {
	conn   *grpc.ClientConn
	client pb.PredictionServiceClient
}

type MLClient interface {
	Predict(ctx context.Context, features []float64) (*models.PredictionResponse, error)
	UpdateModel(ctx context.Context, req *models.UpdateModelRequest) (*models.UpdateModelResponse, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

// NewGRPCMLClient creates a new ML client using gRPC
func NewGRPCMLClient(address string) (MLClient, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ML service: %w", err)
	}

	// Test the connection by creating a client and doing a health check
	client := pb.NewPredictionServiceClient(conn)

	return &grpcMLClient{
		conn:   conn,
		client: client,
	}, nil
}

// Predict sends features to the ML service via gRPC and returns the prediction
func (c *grpcMLClient) Predict(ctx context.Context, features []float64) (*models.PredictionResponse, error) {
	// Validate features
	if err := c.validateFeatures(features); err != nil {
		return nil, err
	}

	// Create gRPC request
	req := &pb.PredictionRequest{
		Features: features,
	}

	// Make gRPC call with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := c.client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gRPC prediction call failed: %w", err)
	}

	explanationItems := make(map[string]models.ExplanationItem)

	if resp.Explanation != nil {
		for featureName, featureExplanation := range resp.Explanation {
			if featureExplanation != nil {
				explanationItems[featureName] = models.ExplanationItem{
					Contribution: featureExplanation.Contribution,
					Impact:       int(featureExplanation.Impact),
				}
			}
		}
	}

	timestamp, err := time.Parse(time.RFC3339, resp.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	return &models.PredictionResponse{
		Prediction:  resp.Prediction,
		Explanation: explanationItems,
		ElapsedTime: resp.ElapsedTime,
		Timestamp:   timestamp,
	}, nil
}

// UpdateModel sends model update request via gRPC
func (c *grpcMLClient) UpdateModel(ctx context.Context, req *models.UpdateModelRequest) (*models.UpdateModelResponse, error) {
	// Convert to gRPC format
	xNew := make([]*pb.FeatureVector, len(req.XNew))
	for i, features := range req.XNew {
		xNew[i] = &pb.FeatureVector{Values: features}
	}

	xVal := make([]*pb.FeatureVector, len(req.XVal))
	for i, features := range req.XVal {
		xVal[i] = &pb.FeatureVector{Values: features}
	}

	grpcReq := &pb.UpdateModelRequest{
		XNew:   xNew,
		YNew:   req.YNew,
		XVal:   xVal,
		YVal:   req.YVal,
		Epochs: int32(req.Epochs),
	}

	// Make gRPC call with longer timeout for model updates
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	resp, err := c.client.UpdateModel(ctx, grpcReq)
	if err != nil {
		return nil, fmt.Errorf("gRPC model update call failed: %w", err)
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, resp.Timestamp)
	if err != nil {
		timestamp = time.Now() // Fallback
	}

	return &models.UpdateModelResponse{
		Status:      resp.Status,
		AUCBefore:   resp.AucBefore,
		AUCAfter:    resp.AucAfter,
		PRAUCBefore: resp.PrAucBefore,
		PRAUCAfter:  resp.PrAucAfter,
		ElapsedTime: resp.ElapsedTime,
		Timestamp:   timestamp,
	}, nil
}

// HealthCheck performs a health check via gRPC
func (c *grpcMLClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &pb.HealthCheckRequest{}
	resp, err := c.client.HealthCheck(ctx, req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Status != "healthy" {
		return fmt.Errorf("service unhealthy: %s", resp.Status)
	}

	return nil
}

// Close closes the gRPC connection
func (c *grpcMLClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// validateFeatures checks if the feature array is valid
func (c *grpcMLClient) validateFeatures(features []float64) error {
	// Check if we have exactly 9 features
	if len(features) != 9 {
		return errors.New("incorrect number of features: expected 7")
	}

	// Validate age (should be positive)
	if features[0] <= 0 {
		return errors.New("age must be positive")
	}

	// Validate binary features (0/1)
	if !c.isBinary(features[2]) || !c.isBinary(features[3]) || !c.isBinary(features[5]) || !c.isBinary(features[8]) {
		return errors.New("is_cholesterol, is_macrosomic_baby, is_bloodline, and is_hypertension must be 0 or 1")
	}

	// Validate BMI (typical range)
	if features[7] < 10 || features[7] > 60 {
		return errors.New("BMI out of typical range (10-60)")
	}

	// Validate physical activity (non-negative)
	if features[4] < 0 {
		return errors.New("physical activity frequency cannot be negative")
	}

	return nil
}

// isBinary checks if a value is 0 or 1
func (c *grpcMLClient) isBinary(val float64) bool {
	return val == 0 || val == 1
}
