package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

const envFilePath = "/var/envshare/.env"

func main() {
	ssmPath := os.Getenv("SSM_PATH")
	secretsPath := os.Getenv("SECRETS_MANAGER_PATH")
	refreshInterval := os.Getenv("ENV_REFRESH")
	interval := 0
	if refreshInterval != "" {
		fmt.Sscanf(refreshInterval, "%d", &interval)
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	run := func() {
		secrets := make(map[string]string)
		if ssmPath != "" {
			ssmClient := ssm.NewFromConfig(cfg)
			out, err := fetchSSMParams(ctx, ssmClient, ssmPath)
			if err != nil {
				log.Printf("error fetching SSM parameters: %v", err)
			} else {
				for k, v := range out {
					secrets[k] = v
				}
			}
		}
		if secretsPath != "" {
			smClient := secretsmanager.NewFromConfig(cfg)
			out, err := fetchSecrets(ctx, smClient, secretsPath)
			if err != nil {
				log.Printf("error fetching secrets: %v", err)
			} else {
				for k, v := range out {
					secrets[k] = v
				}
			}
		}
		writeEnvFile(secrets)
	}

	run()
	if interval > 0 {
		for {
			time.Sleep(time.Duration(interval) * time.Second)
			run()
		}
	} else {
		select {}
	}
}

func fetchSSMParams(ctx context.Context, client *ssm.Client, path string) (map[string]string, error) {
	params := map[string]string{}
	nextToken := ""
	for {
		out, err := client.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
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

func fetchSecrets(ctx context.Context, client *secretsmanager.Client, prefix string) (map[string]string, error) {
	secrets := map[string]string{}
	list, err := client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		return nil, err
	}
	for _, s := range list.SecretList {
		if s.Name != nil && strings.HasPrefix(*s.Name, prefix) {
			out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
				SecretId: s.Name,
			})
			if err != nil {
				continue
			}
			key := strings.TrimPrefix(*s.Name, prefix)
			key = strings.TrimPrefix(key, "/")
			secrets[key] = aws.ToString(out.SecretString)
		}
	}
	return secrets, nil
}

func writeEnvFile(envs map[string]string) {
	f, err := os.Create(envFilePath)
	if err != nil {
		log.Printf("failed to write env file: %v", err)
		return
	}
	defer f.Close()
	for k, v := range envs {
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}
	log.Printf("env file written with %d entries", len(envs))
}
