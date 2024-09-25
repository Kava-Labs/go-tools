package grpc

import (
	"crypto/tls"
	"fmt"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewGrpcConnection parses a GRPC endpoint and creates a connection to it
func NewGrpcConnection(endpoint string) (*grpc.ClientConn, error) {
	grpcURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grpc url: %w", err)
	}

	var secureOpt grpc.DialOption
	switch grpcURL.Scheme {
	case "http":
		secureOpt = grpc.WithInsecure()
	case "https":
		creds := credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})
		secureOpt = grpc.WithTransportCredentials(creds)
	default:
		return nil, fmt.Errorf("unknown grpc url scheme: %s", grpcURL.Scheme)
	}

	grpcConn, err := grpc.Dial(grpcURL.Host, secureOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to dial grpc: %w", err)
	}

	return grpcConn, nil
}
