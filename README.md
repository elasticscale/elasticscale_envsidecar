# elasticscale_envsidecar

This is a lightweight Go application designed to fetch environment variables from AWS Systems Manager (SSM) Parameter Store and writing them to a .env file for use by other processes. It’s ideal for sidecar containers in cloud-native environments, providing a simple way to manage and refresh parameters easily.

## Features
- Fetches parameters from AWS SSM Parameter Store.
- Writes key-value pairs to /var/envshare/.env.
- Supports periodic refresh of environment variables based on a configurable interval.
- Unit-tested with mocks for reliable behavior.

## Environment Variables
- SSM_PATH: The path in SSM Parameter Store to fetch parameters from (e.g., /myapp/config).
- ENV_REFRESH: (Optional) Interval in seconds to refresh the .env file (e.g., 300 for 5 minutes). If unset or 0, runs once and waits indefinitely.

## How It Works
- Initialization: Loads AWS configuration using the default SDK chain (IAM roles, credentials file, etc.).
- Fetching: Queries SSM Parameter Store for all parameters under SSM_PATH (recursive, decrypted).
- Writing: Combines results into a .env file at /var/envshare/.env.
- Refresh: If ENV_REFRESH is set, repeats the process at the specified interval.

## ECS Task Definition Example
```
{
    ...
    "volumes": [
        {
            "name": "envvolume",
            "host": {}
        }
    ],
    "containerDefinitions": [
        {
            "name": "ubuntu",
            "image": "ubuntu:22.04",
            "essential": true,
            "command": [
                "tail",
                "-f",
                "/dev/null"
            ],
            "mountPoints": [
                {
                    "sourceVolume": "envvolume",
                    "containerPath": "/var/envshare",
                    "readOnly": true
                }
            ],
            "dependsOn": [
                {
                    "containerName": "envsidecar",
                    "condition": "START"
                }
            ],
        },
        {
            "name": "envsidecar",
            "image": "public.ecr.aws/elasticscale/elastic-staging-envsidecar:latest",
            "essential": true,
            "environment": [
                {
                    "name": "SSM_PATH",
                    "value": "/elastic/staging/shared/"
                },
                {
                    "name": "ENV_REFRESH",
                    "value": "60"
                }
            ],
            "mountPoints": [
                {
                    "sourceVolume": "envvolume",
                    "containerPath": "/var/envshare",
                    "readOnly": false
                }
            ]
        }
    ],
    ...
}
```

## Contact
For issues or questions, reach out to the ElasticScale team.