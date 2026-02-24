# SitAware — AWS ECS Deployment

Deploy SitAware on AWS ECS Fargate with ALB, NLB, RDS (PostgreSQL + PostGIS), ElastiCache (Redis), and S3.

## Architecture

```
Internet → ALB (TLS/HTTP) → ECS Fargate
             │                 ├── sitaware-api (port 8080) → RDS PostgreSQL
             │                 │                            → ElastiCache Redis
             │                 │                            → S3 bucket
             │                 ├── sitaware-web (port 3000) → api (Service Connect)
             │                 └── sitaware-mediamtx (8889) → api (auth hook)
             │                       ↑ WHIP/WHEP (HTTP)       → S3 (recordings)
             │
         → NLB (UDP/TCP) → ECS Fargate
                             └── sitaware-coturn (3478)
                                   ↑ STUN/TURN (UDP/TCP)
                                     browsers connect directly
```

- **ALB** handles TLS termination for HTTP traffic — API, web, and WHIP/WHEP media signaling
- **NLB** handles non-HTTP traffic — coturn TURN/STUN (UDP/TCP) must be directly reachable by browsers for WebRTC NAT traversal
- **Service Connect** provides internal DNS names (`api.sitaware.local`, `mediamtx.sitaware.local`, `coturn.sitaware.local`)
- **S3** is used directly via IAM task role — no static credentials needed
- **Fargate** provides serverless container hosting — no EC2 instances to manage

### Network Protocol Summary

| Service | Port | Protocol | Accessible by | Load Balancer |
|---|---|---|---|---|
| **API** | 8080 | HTTP | ALB only (internal) | ALB |
| **Web** | 3000 | HTTP | ALB only (internal) | ALB |
| **MediaMTX** (WHIP/WHEP) | 8889 | HTTP | ALB only (internal) | ALB |
| **MediaMTX** (RTSP) | 8554 | TCP | VPC only (hardware devices) | None (Service Connect) |
| **MediaMTX** (RTMP) | 1935 | TCP | VPC only (hardware devices) | None (Service Connect) |
| **coturn** (STUN/TURN) | 3478 | UDP + TCP | **End users (browsers)** | **NLB** |
| **coturn** (relay) | 49152-65535 | UDP | **End users (browsers)** | **NLB** |

> **Why does coturn need an NLB?** TURN/STUN is a binary UDP/TCP protocol, not HTTP. Browsers connect to coturn directly to establish WebRTC media channels through NATs and firewalls. An ALB (HTTP-only) cannot handle this traffic.

## Prerequisites

Before deploying, you need the following AWS resources:

| Resource | Purpose |
|---|---|
| **VPC** with 2+ private subnets | Container networking |
| **ALB** with HTTPS listener | TLS termination, HTTP routing (API, web, WHIP/WHEP) |
| **ALB target group: `sitaware-api`** | Routes `/api/*`, `/healthz`, `/readyz`, `/ws` to API |
| **ALB target group: `sitaware-web`** | Routes everything else to web |
| **ALB target group: `sitaware-mediamtx`** | Routes `/whip/*`, `/whep/*` to MediaMTX |
| **NLB** with Elastic IP | UDP/TCP routing for coturn (TURN/STUN) |
| **NLB target group: `sitaware-coturn-udp`** | UDP 3478 → coturn |
| **NLB target group: `sitaware-coturn-tcp`** | TCP 3478 → coturn |
| **NLB target group: `sitaware-coturn-relay`** | UDP 49152-65535 → coturn (relay ports) |
| **RDS PostgreSQL 16** with PostGIS | Database (enable `postgis` extension) |
| **ElastiCache Redis** | Pub/sub messaging (enable transit encryption) |
| **S3 bucket** | File/tile/recording storage |
| **ECR repositories** | `sitaware/api` and `sitaware/web` |
| **CloudWatch log groups** | `/ecs/sitaware-api`, `/ecs/sitaware-web`, `/ecs/sitaware-mediamtx`, `/ecs/sitaware-coturn` |
| **ECS cluster** named `sitaware` | Container orchestration |
| **Cloud Map namespace** `sitaware.local` | Service Connect (service-to-service discovery) |
| **IAM execution role** | Pull ECR images, read SSM parameters, write CloudWatch logs |
| **IAM API task role** | S3 access (GetObject, PutObject, DeleteObject, ListBucket) |
| **IAM mediamtx task role** | S3 access for recordings (PutObject) |
| **IAM coturn task role** | Minimal (no special permissions needed) |
| **IAM web task role** | Minimal (no special permissions needed) |
| **SSM Parameter Store** secrets | Sensitive config values |

## Step 1: Create SSM Parameters

Store secrets in SSM Parameter Store (SecureString):

```bash
aws ssm put-parameter --name /sitaware/admin-password --type SecureString --value "YOUR_ADMIN_PASSWORD"
aws ssm put-parameter --name /sitaware/jwt-secret      --type SecureString --value "YOUR_JWT_SECRET"
aws ssm put-parameter --name /sitaware/db-password      --type SecureString --value "YOUR_DB_PASSWORD"
aws ssm put-parameter --name /sitaware/redis-password   --type SecureString --value "YOUR_REDIS_PASSWORD"
aws ssm put-parameter --name /sitaware/turn-password    --type SecureString --value "YOUR_TURN_PASSWORD"

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
aws logs create-log-group --log-group-name /ecs/sitaware-mediamtx
aws logs create-log-group --log-group-name /ecs/sitaware-coturn
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
| `NLB_ELASTIC_IP` (in coturn `EXTERNAL_IP`) | The Elastic IP attached to your NLB |
| `subnet-PRIVATE_1`, `subnet-PRIVATE_2` | Your private subnet IDs |
| `sg-API_SG`, `sg-WEB_SG`, `sg-MEDIAMTX_SG`, `sg-COTURN_SG` | Your security group IDs |
| Target group ARNs | Your ALB and NLB target group ARNs |

> **Redis TLS**: The task definition sets `REDIS_TLS=true` because ElastiCache requires transit encryption. If your ElastiCache cluster has transit encryption disabled, set this to `false`.

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
aws ecs register-task-definition --cli-input-json file://deploy/ecs/mediamtx-task-definition.json
aws ecs register-task-definition --cli-input-json file://deploy/ecs/coturn-task-definition.json
```

## Step 7: Create ECS Cluster (if not already created)

```bash
aws ecs create-cluster --cluster-name sitaware --service-connect-defaults namespace=sitaware.local
```

## Step 8: Create Services

```bash
aws ecs create-service --cli-input-json file://deploy/ecs/api-service.json
aws ecs create-service --cli-input-json file://deploy/ecs/web-service.json
aws ecs create-service --cli-input-json file://deploy/ecs/mediamtx-service.json
aws ecs create-service --cli-input-json file://deploy/ecs/coturn-service.json
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

# MediaMTX WHIP/WHEP routes (video streaming signaling)
aws elbv2 create-rule \
  --listener-arn LISTENER_ARN \
  --priority 14 \
  --conditions Field=path-pattern,Values='/whip/*' \
  --actions Type=forward,TargetGroupArn=MEDIAMTX_TG_ARN

aws elbv2 create-rule \
  --listener-arn LISTENER_ARN \
  --priority 15 \
  --conditions Field=path-pattern,Values='/whep/*' \
  --actions Type=forward,TargetGroupArn=MEDIAMTX_TG_ARN

# Web routes (default action — lowest priority)
aws elbv2 modify-listener \
  --listener-arn LISTENER_ARN \
  --default-actions Type=forward,TargetGroupArn=WEB_TG_ARN
```

> **WebSocket note**: ALB natively supports WebSocket upgrades. Ensure the API target group has stickiness enabled if you run multiple API tasks and need session affinity.

## Step 9b: Configure NLB for coturn

coturn requires an NLB because TURN/STUN uses UDP/TCP, not HTTP. Browsers connect to coturn directly for WebRTC NAT traversal.

### Create NLB with Elastic IP

```bash
# Allocate an Elastic IP (stable public address for TURN)
aws ec2 allocate-address --domain vpc
# Note the AllocationId (e.g., eipalloc-xxxx)

# Create the NLB with the Elastic IP
aws elbv2 create-load-balancer \
  --name sitaware-coturn-nlb \
  --type network \
  --subnet-mappings SubnetId=subnet-PUBLIC_1,AllocationId=eipalloc-xxxx
# Note: Use a PUBLIC subnet for the NLB so it gets a public IP
```

> **Important:** Update the `EXTERNAL_IP` value in `coturn-task-definition.json` with the Elastic IP address (not the allocation ID) so coturn includes the correct public IP in STUN responses.

### Create Target Groups

```bash
# STUN/TURN signaling (UDP)
aws elbv2 create-target-group \
  --name sitaware-coturn-udp \
  --protocol UDP \
  --port 3478 \
  --vpc-id VPC_ID \
  --target-type ip \
  --health-check-protocol TCP \
  --health-check-port 3478

# STUN/TURN signaling (TCP fallback)
aws elbv2 create-target-group \
  --name sitaware-coturn-tcp \
  --protocol TCP \
  --port 3478 \
  --vpc-id VPC_ID \
  --target-type ip

# TURN relay ports (UDP range)
aws elbv2 create-target-group \
  --name sitaware-coturn-relay \
  --protocol UDP \
  --port 49152 \
  --vpc-id VPC_ID \
  --target-type ip \
  --health-check-protocol TCP \
  --health-check-port 3478
```

### Create NLB Listeners

```bash
# UDP listener for STUN/TURN
aws elbv2 create-listener \
  --load-balancer-arn NLB_ARN \
  --protocol UDP \
  --port 3478 \
  --default-actions Type=forward,TargetGroupArn=COTURN_UDP_TG_ARN

# TCP listener for TURN fallback
aws elbv2 create-listener \
  --load-balancer-arn NLB_ARN \
  --protocol TCP \
  --port 3478 \
  --default-actions Type=forward,TargetGroupArn=COTURN_TCP_TG_ARN

# UDP listener for relay port range
aws elbv2 create-listener \
  --load-balancer-arn NLB_ARN \
  --protocol UDP \
  --port 49152-65535 \
  --default-actions Type=forward,TargetGroupArn=COTURN_RELAY_TG_ARN
```

> **Relay ports:** The relay port range (49152-65535) is used by coturn to relay media between WebRTC peers. NLB supports port range listeners. The target group health check uses TCP 3478 since the relay ports are dynamically allocated.

## Step 10: Verify Deployment

```bash
# Check service status
aws ecs describe-services --cluster sitaware \
  --services sitaware-api sitaware-web sitaware-mediamtx sitaware-coturn

# Watch task status
aws ecs list-tasks --cluster sitaware --service-name sitaware-api
aws ecs describe-tasks --cluster sitaware --tasks TASK_ARN

# Tail logs
aws logs tail /ecs/sitaware-api --follow
aws logs tail /ecs/sitaware-web --follow
aws logs tail /ecs/sitaware-mediamtx --follow
aws logs tail /ecs/sitaware-coturn --follow

# Test health endpoint
curl https://sitaware.example.com/healthz

# Test TURN connectivity (replace NLB_DNS with your NLB's DNS name)
# Use a STUN client or browser WebRTC internals to verify coturn is reachable
```

## Step 11: Update Deployment

To deploy a new version:

```bash
# Build and push new images (Step 4)

# Force new deployment (pulls latest image)
aws ecs update-service --cluster sitaware --service sitaware-api --force-new-deployment
aws ecs update-service --cluster sitaware --service sitaware-web --force-new-deployment

# MediaMTX and coturn use public images — force redeploy to pick up config changes
aws ecs update-service --cluster sitaware --service sitaware-mediamtx --force-new-deployment
aws ecs update-service --cluster sitaware --service sitaware-coturn --force-new-deployment
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

### MediaMTX Security Group (`sg-MEDIAMTX_SG`)

| Direction | Port | Source | Purpose |
|---|---|---|---|
| Inbound | 8889 | ALB SG | WHIP/WHEP signaling from load balancer |
| Inbound | 8554 | VPC CIDR | RTSP from hardware devices (internal) |
| Inbound | 1935 | VPC CIDR | RTMP from hardware devices (internal) |
| Inbound | 9997 | VPC CIDR | Health check (management API) |
| Outbound | 8080 | API SG | Auth hook + recording callback to API |
| Outbound | 3478 | coturn SG | ICE candidates via TURN |
| Outbound | 443 | 0.0.0.0/0 | CloudWatch |

### coturn Security Group (`sg-COTURN_SG`)

| Direction | Port | Protocol | Source | Purpose |
|---|---|---|---|---|
| Inbound | 3478 | UDP | 0.0.0.0/0 | STUN/TURN from browsers |
| Inbound | 3478 | TCP | 0.0.0.0/0 | TURN TCP fallback from browsers |
| Inbound | 49152-65535 | UDP | 0.0.0.0/0 | TURN relay media from browsers |
| Outbound | All | All | 0.0.0.0/0 | Relay media to peers |

> **coturn inbound from 0.0.0.0/0**: This is required because end users' browsers connect to coturn directly from arbitrary IPs. The TURN authentication mechanism (long-term credentials) protects against unauthorized use.
