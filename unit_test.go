package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// fetchSSMParamsMockable is the core logic extracted for testing
func fetchSSMParamsMockable(ctx context.Context, getParams func(context.Context, *ssm.GetParametersByPathInput, ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error), path string) (map[string]string, error) {
	params := map[string]string{}
	nextToken := ""
	for {
		out, err := getParams(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(path),
			WithDecryption: aws.Bool(true),
			Recursive:      aws.Bool(true),
			NextToken:      aws.String(nextToken),
		})
		if err != nil {
			return nil, err
		}
		for _, p := range out.Parameters {
			key := strings.TrimPrefix(*p.Name, path)
			key = strings.TrimPrefix(key, "/")
			params[key] = *p.Value
		}
		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		nextToken = *out.NextToken
	}
	return params, nil
}

// mockSSMClient is a minimal mock for SSM
type mockSSMClient struct {
	params map[string]string
	err    error
}

// GetParametersByPath mocks the SSM client's method
func (m *mockSSMClient) GetParametersByPath(ctx context.Context, input *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := &ssm.GetParametersByPathOutput{}
	for k, v := range m.params {
		out.Parameters = append(out.Parameters, ssmtypes.Parameter{
			Name:  aws.String(*input.Path + "/" + k),
			Value: aws.String(v),
		})
	}
	return out, nil
}

// ========== Unit Tests ==========

func TestFetchSSMParams_Empty(t *testing.T) {
	mock := &mockSSMClient{params: map[string]string{}}
	out, err := fetchSSMParamsMockable(context.Background(), mock.GetParametersByPath, "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty map, got %d entries", len(out))
	}
}

func TestFetchSSMParams_Error(t *testing.T) {
	expectedErr := errors.New("ssm failure")
	mock := &mockSSMClient{err: expectedErr}
	_, err := fetchSSMParamsMockable(context.Background(), mock.GetParametersByPath, "/test")
	if err == nil || !strings.Contains(err.Error(), "ssm failure") {
		t.Errorf("expected error, got %v", err)
	}
}