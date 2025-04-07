package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types as ssmtypes"
)

// Mock SSM client

type mockSSMClient struct {
	params map[string]string
	error  error
}

func (m *mockSSMClient) GetParametersByPath(ctx context.Context, input *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	if m.error != nil {
		return nil, m.error
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

// Mock Secrets Manager client

type mockSecretsManagerClient struct {
	secrets map[string]string
	error   error
}

func (m *mockSecretsManagerClient) ListSecrets(ctx context.Context, input *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	if m.error != nil {
		return nil, m.error
	}
	out := &secretsmanager.ListSecretsOutput{}
	for name := range m.secrets {
		secretName := name
		out.SecretList = append(out.SecretList, types.SecretListEntry{Name: &secretName})
	}
	return out, nil
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.error != nil {
		return nil, m.error
	}
	val, ok := m.secrets[*input.SecretId]
	if !ok {
		return nil, errors.New("not found")
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: &val}, nil
}

// ========== Unit Tests ==========

func TestFetchSSMParams_Empty(t *testing.T) {
	mock := &mockSSMClient{params: map[string]string{}}
	out, err := fetchSSMParams(context.Background(), mock, "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty map, got %d entries", len(out))
	}
}

func TestFetchSSMParams_Error(t *testing.T) {
	expectedErr := errors.New("ssm failure")
	mock := &mockSSMClient{error: expectedErr}
	_, err := fetchSSMParams(context.Background(), mock, "/test")
	if err == nil || !strings.Contains(err.Error(), "ssm failure") {
		t.Errorf("expected error, got %v", err)
	}
}

func TestFetchSecrets_PrefixFilter(t *testing.T) {
	mock := &mockSecretsManagerClient{
		secrets: map[string]string{
			"/prod/db/password": "secret123",
			"/prod/db/user":     "admin",
			"/other/skip":       "nope",
		},
	}
	out, err := fetchSecrets(context.Background(), mock, "/prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(out))
	}
	if out["db/password"] != "secret123" {
		t.Errorf("expected secret123, got %s", out["db/password"])
	}
}

func TestFetchSecrets_Error(t *testing.T) {
	expectedErr := errors.New("secrets error")
	mock := &mockSecretsManagerClient{error: expectedErr}
	_, err := fetchSecrets(context.Background(), mock, "/any")
	if err == nil || !strings.Contains(err.Error(), "secrets error") {
		t.Errorf("expected error, got %v", err)
	}
}
