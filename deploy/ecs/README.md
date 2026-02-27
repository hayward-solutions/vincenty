# SitAware — AWS ECS Deployment

Deploy SitAware on AWS ECS Fargate with ALB, RDS (PostgreSQL + PostGIS), ElastiCache (Redis), and S3.

> **Redis Cluster Mode**: If your ElastiCache cluster has cluster mode enabled, set `REDIS_CLUSTER=true` in the API task definition. The API will use a Redis Cluster client that discovers shard topology automatically via the configuration endpoint.

## Architecture

```
Internet → ALB (TLS) → ECS Fargate
                          ├── sitaware-api (port 8080) → RDS PostgreSQL
                          │                            → ElastiCache Redis
                          │                            → S3 bucket
                          └── sitaware-web (port 3000) → api (Service Connect)
```

- **ALB** handles TLS termination (no Caddy needed)
- **Service Connect** gives the web container a stable DNS name (`api.sitaware.local`) to reach the API internally
- **S3** is used directly via IAM task role — no static credentials needed
- **Fargate** provides serverless container hosting — no EC2 instances to manage

## Prerequisites

Before deploying, you need the following AWS resources:

| Resource | Purpose |
|---|---|
| **VPC** with 2+ private subnets | Container networking |
| **ALB** with HTTPS listener | TLS termination, routing |
| **ALB target group: `sitaware-api`** | Routes `/api/*`, `/healthz`, `/readyz`, `/ws` to API |
| **ALB target group: `sitaware-web`** | Routes everything else to web |
| **RDS PostgreSQL 16** with PostGIS | Database (enable `postgis` extension) |
| **ElastiCache Redis** | Pub/sub messaging (enable transit encryption) |
| **S3 bucket** | File/tile storage |
| **ECR repositories** | `sitaware/api` and `sitaware/web` |
| **CloudWatch log groups** | `/ecs/sitaware-api` and `/ecs/sitaware-web` |
| **ECS cluster** named `sitaware` | Container orchestration |
| **Cloud Map namespace** `sitaware.local` | Service Connect (service-to-service discovery) |
| **IAM execution role** | Pull ECR images, read SSM parameters, write CloudWatch logs |
| **IAM API task role** | S3 access (GetObject, PutObject, DeleteObject, ListBucket) |
| **IAM web task role** | Minimal (no special permissions needed) |
| **SSM Parameter Store** secrets | Sensitive config values |

## Step 1: Create SSM Parameters

Store secrets in SSM Parameter Store (SecureString):

```bash
aws ssm put-parameter --name /sitaware/admin-password --type SecureString --value "YOUR_ADMIN_PASSWORD"
aws ssm put-parameter --name /sitaware/jwt-secret      --type SecureString --value "YOUR_JWT_SECRET"
aws ssm put-parameter --name /sitaware/db-password      --type SecureString --value "YOUR_DB_PASSWORD"
aws ssm put-parameter --name /sitaware/redis-password   --type SecureString --value "YOUR_REDIS_PASSWORD"

# Optional: KMS key ARN for encrypting TOTP secrets (omit to use HKDF from JWT_SECRET)
# aws ssm put-parameter --name /sitaware/mfa-kms-key-arn --type SecureString --value "arn:aws:kms:REGION:ACCOUNT_ID:key/YOUR_KEY_ID"
```

> **WebAuthn configuration**: The API task definition includes `WEBAUTHN_RP_ID` and `WEBAUTHN_RP_ORIGINS` which must match your production domain. Update `sitaware.example.com` to your actual domain in the task definition before deploying.

## Step 2: Create S3 Bucket and IAM Roles

Create the S3 bucket, apply the bucket policy, and set up IAM roles:

```bash
# Create the bucket
aws s3api create-bucket --bucket sitaware-files --region $REGION

# Apply the strict bucket policy (denies non-HTTPS, restricts to task role, requires SSE-S3)
# NOTE: Replace ACCOUNT_ID in s3-bucket-policy.json before applying
aws s3api put-bucket-policy --bucket sitaware-files --policy file://deploy/ecs/s3-bucket-policy.json

# Create the API task role (allows ECS tasks to assume it)
aws iam create-role \
  --role-name sitaware-api-task-role \
  --assume-role-policy-document file://deploy/ecs/api-task-role-trust-policy.json

# Attach the S3 access policy to the task role
aws iam put-role-policy \
  --role-name sitaware-api-task-role \
  --policy-name sitaware-api-s3-access \
  --policy-document file://deploy/ecs/api-task-role-policy.json

# Create the web task role (no special permissions needed)
aws iam create-role \
  --role-name sitaware-web-task-role \
  --assume-role-policy-document file://deploy/ecs/api-task-role-trust-policy.json

# Create the ECS execution role
aws iam create-role \
  --role-name sitaware-ecs-execution-role \
  --assume-role-policy-document file://deploy/ecs/api-task-role-trust-policy.json
```

> **Bucket policy**: The policy files contain `ACCOUNT_ID` placeholders. Run the `sed` command in Step 5 before applying, or replace manually. After replacing placeholders, apply the bucket policy with: `aws s3api put-bucket-policy --bucket sitaware-files --policy file://deploy/ecs/s3-bucket-policy.json`

## Step 3: Create CloudWatch Log Groups

```bash
aws logs create-log-group --log-group-name /ecs/sitaware-api
aws logs create-log-group --log-group-name /ecs/sitaware-web
```

## Step 4: Build and Push Docker Images

```bash
# Authenticate with ECR
aws ecr get-login-password --region REGION | docker login --username AWS --password-stdin ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com

# Build and push API
docker build -t sitaware/api:latest -f services/api/Dockerfile .
docker tag sitaware/api:latest ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com/sitaware/api:latest
docker push ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com/sitaware/api:latest

# Build and push Web
docker build -t sitaware/web:latest -f clients/web/Dockerfile .
docker tag sitaware/web:latest ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com/sitaware/web:latest
docker push ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com/sitaware/web:latest
```

## Step 5: Update Placeholder Values

Before registering task definitions, replace placeholders in the JSON files:

| Placeholder | Replace with |
|---|---|
| `ACCOUNT_ID` | Your AWS account ID (e.g., `123456789012`) |
| `REGION` | Your AWS region (e.g., `us-east-1`) |
| `sitaware-db.cluster-xxxx.REGION.rds.amazonaws.com` | Your RDS endpoint |
| `sitaware-redis.xxxx.REGION.cache.amazonaws.com` | Your ElastiCache endpoint |
| `sitaware.example.com` (in `WEBAUTHN_RP_ID` and `WEBAUTHN_RP_ORIGINS`) | Your actual domain |

> **Redis TLS**: The task definition sets `REDIS_TLS=true` because ElastiCache requires transit encryption. If your ElastiCache cluster has transit encryption disabled, set this to `false`.

> **Redis Cluster**: The task definition defaults `REDIS_CLUSTER` to `false`. If your ElastiCache cluster has cluster mode enabled, change this to `true`. The single `REDIS_HOST` address serves as the cluster configuration endpoint.
| `sitaware.example.com` | Your actual domain |
| `subnet-PRIVATE_1`, `subnet-PRIVATE_2` | Your private subnet IDs |
| `sg-API_SG`, `sg-WEB_SG` | Your security group IDs |
| Target group ARNs | Your actual ALB target group ARNs |

You can use `sed` for this:

```bash
export ACCOUNT_ID=123456789012
export REGION=us-east-1

for f in deploy/ecs/*.json; do
  sed -i "s/ACCOUNT_ID/$ACCOUNT_ID/g; s/REGION/$REGION/g" "$f"
done
```

## Step 6: Register Task Definitions

```bash
aws ecs register-task-definition --cli-input-json file://deploy/ecs/api-task-definition.json
aws ecs register-task-definition --cli-input-json file://deploy/ecs/web-task-definition.json
```

## Step 7: Create ECS Cluster (if not already created)

```bash
aws ecs create-cluster --cluster-name sitaware --service-connect-defaults namespace=sitaware.local
```

## Step 8: Create Services

```bash
aws ecs create-service --cli-input-json file://deploy/ecs/api-service.json
aws ecs create-service --cli-input-json file://deploy/ecs/web-service.json
```

## Step 9: Configure ALB Listener Rules

Create listener rules on your HTTPS (443) listener:

```bash
# API routes (higher priority = checked first)
aws elbv2 create-rule \
  --listener-arn LISTENER_ARN \
  --priority 10 \
  --conditions Field=path-pattern,Values='/api/*' \
  --actions Type=forward,TargetGroupArn=API_TG_ARN

aws elbv2 create-rule \
  --listener-arn LISTENER_ARN \
  --priority 11 \
  --conditions Field=path-pattern,Values='/healthz' \
  --actions Type=forward,TargetGroupArn=API_TG_ARN

aws elbv2 create-rule \
  --listener-arn LISTENER_ARN \
  --priority 12 \
  --conditions Field=path-pattern,Values='/readyz' \
  --actions Type=forward,TargetGroupArn=API_TG_ARN

aws elbv2 create-rule \
  --listener-arn LISTENER_ARN \
  --priority 13 \
  --conditions Field=path-pattern,Values='/ws' \
  --actions Type=forward,TargetGroupArn=API_TG_ARN

# Web routes (default action — lowest priority)
aws elbv2 modify-listener \
  --listener-arn LISTENER_ARN \
  --default-actions Type=forward,TargetGroupArn=WEB_TG_ARN
```

> **WebSocket note**: ALB natively supports WebSocket upgrades. Ensure the API target group has stickiness enabled if you run multiple API tasks and need session affinity.

## Step 10: Verify Deployment

```bash
# Check service status
aws ecs describe-services --cluster sitaware --services sitaware-api sitaware-web

# Watch task status
aws ecs list-tasks --cluster sitaware --service-name sitaware-api
aws ecs describe-tasks --cluster sitaware --tasks TASK_ARN

# Tail logs
aws logs tail /ecs/sitaware-api --follow
aws logs tail /ecs/sitaware-web --follow

# Test health endpoint
curl https://sitaware.example.com/healthz
```

## Step 11: Update Deployment

To deploy a new version:

```bash
# Build and push new images (Step 4)

# Force new deployment (pulls latest image)
aws ecs update-service --cluster sitaware --service sitaware-api --force-new-deployment
aws ecs update-service --cluster sitaware --service sitaware-web --force-new-deployment
```

## IAM Policy Reference

### Execution Role (shared)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ssm:GetParameters",
        "ssm:GetParameter"
      ],
      "Resource": "arn:aws:ssm:REGION:ACCOUNT_ID:parameter/sitaware/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:REGION:ACCOUNT_ID:log-group:/ecs/sitaware-*:*"
    }
  ]
}
```

### API Task Role

See [`api-task-role-policy.json`](api-task-role-policy.json) — grants `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject` on objects and `s3:ListBucket` on the bucket.

### API Task Role Trust Policy

See [`api-task-role-trust-policy.json`](api-task-role-trust-policy.json) — allows `ecs-tasks.amazonaws.com` to assume the role.

### S3 Bucket Policy

See [`s3-bucket-policy.json`](s3-bucket-policy.json) — enforces three restrictions:

| Statement | Effect | Purpose |
|---|---|---|
| `DenyNonHTTPS` | Deny all `s3:*` | Blocks any request not using TLS |
| `DenyUnauthorizedPrincipals` | Deny all `s3:*` | Only the API task role and account root can access the bucket |

## Security Groups

### API Security Group (`sg-API_SG`)

| Direction | Port | Source | Purpose |
|---|---|---|---|
| Inbound | 8080 | ALB SG | HTTP from load balancer |
| Inbound | 8080 | Web SG | Service Connect (internal) |
| Outbound | 5432 | RDS SG | PostgreSQL |
| Outbound | 6379 | Redis SG | ElastiCache |
| Outbound | 443 | 0.0.0.0/0 | S3, SSM, CloudWatch |

### Web Security Group (`sg-WEB_SG`)

| Direction | Port | Source | Purpose |
|---|---|---|---|
| Inbound | 3000 | ALB SG | HTTP from load balancer |
| Outbound | 8080 | API SG | Service Connect to API |
| Outbound | 443 | 0.0.0.0/0 | SSM, CloudWatch |
