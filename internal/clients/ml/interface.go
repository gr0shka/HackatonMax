package ml

import (
	"context"
)

// MLClient defines the contract for communicating with the external route optimization ML service.
type MLClient interface {
	OptimizeRoute(ctx context.Context, req *OptimizeRequest) (*OptimizeResponse, error)
}
