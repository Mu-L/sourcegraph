package client

import (
	"context"
	"fmt"
	"time"
)

type EmbeddingsClient interface {
	// GetEmbeddings returns embeddings for the given texts.
	GetEmbeddings(ctx context.Context, texts []string) ([]float32, error)
	// GetDimensions returns the dimensionality of the embedding space.
	GetDimensions() (int, error)
	// GetModelIdentifier returns the identifier of the model used to generate embeddings. The format is
	// "provider/name", for example "openai/text-embedding-ada-002".
	GetModelIdentifier() string
}

func NewRateLimitExceededError(retryAfter time.Time) error {
	return &RateLimitExceededError{
		retryAfter: retryAfter,
	}
}

type RateLimitExceededError struct {
	retryAfter time.Time
}

func (e RateLimitExceededError) Error() string { return "rate limit exceeded" }

func (e RateLimitExceededError) RetryAfter() time.Time { return e.retryAfter }

type PartialError struct {
	Err   error
	Index int
}

func (p PartialError) Error() string {
	return fmt.Sprintf("partial error: input %d: %s", p.Index, p.Err)
}
