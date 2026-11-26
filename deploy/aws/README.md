# AWS Hosting Guide — PromisedLand (Single EC2 Instance)

> **Comprehensive step-by-step guide.** Covers VPC, EC2, EBS, Docker, TLS, backups, S3, and all troubleshooting encountered during a real deployment on Amazon Linux 2023 (eu-north-1 / Stockholm). Written so a deleted instance can be rebuilt from scratch.

---

## Table of Contents

1. [What You Are Deploying](#1-what-you-are-deploying)
2. [Prerequisites](#2-prerequisites)
3. [Naming Convention](#3-naming-convention)
4. [VPC Setup](#4-vpc-setup)
5. [Security Groups](#5-security-groups)
6. [EC2 Instance Launch](#6-ec2-instance-launch)
7. [IAM Instance Profile (S3 + CloudWatch)](#7-iam-instance-profile-s3--cloudwatch)
8. [EBS Volume Mount](#8-ebs-volume-mount)
9. [Host Setup (Docker, Git, Clone)](#9-host-setup-docker-git-clone)
10. [Configure `.env`](#10-configure-env)
11. [TLS Setup (Caddy + Let's Encrypt)](#11-tls-setup-caddy--lets-encrypt)
12. [First Deploy](#12-first-deploy)
13. [Verification](#13-verification)
14. [Backups (Local + S3)](#14-backups-local--s3)
15. [Monitoring](#15-monitoring)
16. [Updates & Rollback](#16-updates--rollback)
17. [Cost & Housekeeping](#17-cost--housekeeping)
18. [Security Checklist](#18-security-checklist)
19. [Troubleshooting](#19-troubleshooting)
20. [Known Gaps](#20-known-gaps)

---

## 1. What You Are Deploying

Three images, built from this repo:

| Image | Source | Role |
|---|---|---|
| `promisedland-api:ci` | `backend/Dockerfile` | Go API on `:8080`, runs as UID 100 (`app`) |
| `promisedland-web:ci` | `frontend/Dockerfile` | Static SPA + nginx on `:80` (UID 101) |
| `promisedland-edge:ci` | `deploy/nginx/Dockerfile` | **Production entrypoint**: SPA + nginx proxying `/api/`, `/ws/`, `/healthz`, `/readyz`, `/media/` to the API. Listens on `:80`, UID 101 |

The prod compose override (`deploy/compose/prod.yml`) activates the edge `proxy` profile, publishes **only port 80** on the host, sets `APP_ENV=production`, and refuses to start without `CORS_ORIGIN` and `SESSION_SECRET`. The API is reachable on `127.0.0.1:8080` for diagnostics only.

SQLite lives on an EBS-backed host path (`/srv/promisedland/data` → container `/data`); user media lands in the same directory under `media/`. The API runs as UID 100, so that directory must be owned by `100:101`.

---

## 2. Prerequisites

- An AWS account with a billing budget/alert set (set this first).
- A domain name (optional but recommended; TLS is painful without one). Free subdomains (DuckDNS) work for testing.
- Local tools: AWS CLI v2, `ssh`, `docker`.
- This repository checked out locally.

---

## 3. Naming Convention

Use a consistent prefix `promisedland-<env>-<purpose>`.

| Resource | Example Name |
|---|---|
| EC2 instance | `promisedland-prod-app` |
| Security Group | `promisedland-prod-sg` |
| EBS volume | `promisedland-prod-data` |
| S3 bucket | `promisedland-<yourname>-data` (globally unique) |
| ECR repos | `promisedland/api`, `promisedland/web`, `promisedland/edge` |
| IAM role/profile | `promisedland-prod-instance-role` / `promisedland-prod-instance-profile` |
| CloudWatch log group | `/promisedland/prod` |
| VPC | `promisedland-prod-vpc` |
| Subnets | `promisedland-prod-public-1a`, `-public-1b`, `-private-1a`, `-private-1b` |
| Route tables | `promisedland-prod-public-rt`, `promisedland-prod-private-rt` |
| Internet Gateway | `promisedland-prod-igw` |
| NAT Gateway | `promisedland-prod-nat` |
| Key pair | `promisedland-prod-key` |

---

## 4. VPC Setup

### 4.1 Quick Option — VPC Wizard (Console)

1. VPC Dashboard → **Create VPC** → choose **VPC and more**.
2. Settings:
   - Name tag auto-generation: `promisedland-prod`
   - IPv4 CIDR: `10.0.0.0/16`
   - Number of AZs: **2**
   - Number of public subnets: **2**
   - Number of private subnets: **2**
   - NAT gateways: **In 1 AZ** (cheaper) or 1 per AZ (HA)
   - DNS hostnames: ✅, DNS resolution: ✅
3. Click **Create VPC**. Wait ~2 minutes for NAT Gateway.

### 4.2 Manual Option — AWS CLI

```bash
export AWS_REGION=eu-north-1   # or us-east-1, etc.
export AZA=${AWS_REGION}a
export AZB=${AWS_REGION}b

# VPC
VPC_ID=$(aws ec2 create-vpc --cidr-block 10.0.0.0/16 \
  --tag-specifications "ResourceType=vpc,Tags=[{Key=Name,Value=promisedland-prod-vpc}]" \
  --query 'Vpc.VpcId' --output text)
aws ec2 modify-vpc-attribute --vpc-id $VPC_ID --enable-dns-hostnames
aws ec2 modify-vpc-attribute --vpc-id $VPC_ID --enable-dns-support

# Subnets
PUBA=$(aws ec2 create-subnet --vpc-id $VPC_ID --cidr-block 10.0.1.0/24 --availability-zone $AZA \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=promisedland-prod-public-1a}]" \
  --query 'Subnet.SubnetId' --output text)
PUBB=$(aws ec2 create-subnet --vpc-id $VPC_ID --cidr-block 10.0.2.0/24 --availability-zone $AZB \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=promisedland-prod-public-1b}]" \
  --query 'Subnet.SubnetId' --output text)
PRIVA=$(aws ec2 create-subnet --vpc-id $VPC_ID --cidr-block 10.0.11.0/24 --availability-zone $AZA \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=promisedland-prod-private-1a}]" \
  --query 'Subnet.SubnetId' --output text)
PRIVB=$(aws ec2 create-subnet --vpc-id $VPC_ID --cidr-block 10.0.12.0/24 --availability-zone $AZB \
  --tag-specifications "ResourceType=subnet,Tags=[{Key=Name,Value=promisedland-prod-private-1b}]" \
  --query 'Subnet.SubnetId' --output text)

aws ec2 modify-subnet-attribute --subnet-id $PUBA --map-public-ip-on-launch
aws ec2 modify-subnet-attribute --subnet-id $PUBB --map-public-ip-on-launch

# Internet Gateway
IGW=$(aws ec2 create-internet-gateway \
  --tag-specifications "ResourceType=internet-gateway,Tags=[{Key=Name,Value=promisedland-prod-igw}]" \
  --query 'InternetGateway.InternetGatewayId' --output text)
aws ec2 attach-internet-gateway --internet-gateway-id $IGW --vpc-id $VPC_ID

# NAT Gateway (in public subnet A)
NAT_EIP=$(aws ec2 allocate-address --domain vpc \
  --tag-specifications "ResourceType=elastic-ip,Tags=[{Key=Name,Value=promisedland-prod-nat-eip}]" \
  --query 'AllocationId' --output text)
NAT=$(aws ec2 create-nat-gateway --subnet-id $PUBA --allocation-id $NAT_EIP \
  --tag-specifications "ResourceType=natgateway,Tags=[{Key=Name,Value=promisedland-prod-nat}]" \
  --query 'NatGateway.NatGatewayId' --output text)
aws ec2 wait nat-gateway-available --nat-gateway-ids $NAT

# Public route table
PUB_RT=$(aws ec2 create-route-table --vpc-id $VPC_ID \
  --tag-specifications "ResourceType=route-table,Tags=[{Key=Name,Value=promisedland-prod-public-rt}]" \
  --query 'RouteTable.RouteTableId' --output text)
aws ec2 create-route --route-table-id $PUB_RT --destination-cidr-block 0.0.0.0/0 --gateway-id $IGW
aws ec2 associate-route-table --route-table-id $PUB_RT --subnet-id $PUBA
aws ec2 associate-route-table --route-table-id $PUB_RT --subnet-id $PUBB

# Private route table
PRIV_RT=$(aws ec2 create-route-table --vpc-id $VPC_ID \
  --tag-specifications "ResourceType=route-table,Tags=[{Key=Name,Value=promisedland-prod-private-rt}]" \
  --query 'RouteTable.RouteTableId' --output text)
aws ec2 create-route --route-table-id $PRIV_RT --destination-cidr-block 0.0.0.0/0 --nat-gateway-id $NAT
aws ec2 associate-route-table --route-table-id $PRIV_RT --subnet-id $PRIVA
aws ec2 associate-route-table --route-table-id $PRIV_RT --subnet-id $PRIVB
```

### 4.3 Route Table → Subnet Mapping

| Route Table | Subnets | `0.0.0.0/0` Target |
|---|---|---|
| `promisedland-prod-public-rt` | `public-1a`, `public-1b` | Internet Gateway |
| `promisedland-prod-private-rt` | `private-1a`, `private-1b` | NAT Gateway |
| Main (default) | *(none — fallback)* | *(none — no internet)* |

**Key rules:**
- Both public subnets share one public route table.
- Both private subnets share one private route table (single NAT).
- Every subnet must be explicitly associated. A subnet with no association falls back to the main route table (no internet) — common "why can't my instance reach the internet" bug.
- Leave the main route table empty (no IGW/NAT route) so new subnets default to "no internet" (safe failure mode).

### 4.4 VPC Verification

```bash
aws ec2 describe-route-tables --filters "Name=vpc-id,Values=$VPC_ID" \
  --query 'RouteTables[*].[RouteTableId,Tags[?Key==`Name`].Value|[0],Associations[*].SubnetId,Routes[*].DestinationCidrBlock]'
```

---

## 5. Security Groups

### Public SG (`promisedland-prod-public-sg`)

| Direction | Type | Protocol | Port | Source |
|---|---|---|---|---|
| Inbound | SSH | TCP | 22 | My IP (`x.x.x.x/32`) |
| Inbound | HTTP | TCP | 80 | Anywhere (`0.0.0.0/0`) |
| Inbound | HTTPS | TCP | 443 | Anywhere (`0.0.0.0/0`) |
| Outbound | All traffic | All | All | `0.0.0.0/0` (default) |

### Private SG (`promisedland-prod-private-sg`)

| Direction | Type | Protocol | Port | Source |
|---|---|---|---|---|
| Inbound | All traffic | All | All | `promisedland-prod-public-sg` (SG reference, not CIDR) |
| Outbound | All traffic | All | All | `0.0.0.0/0` (default) |

### CLI

```bash
PUB_SG=$(aws ec2 create-security-group --group-name promisedland-prod-public-sg \
  --description "PromisedLand public: HTTP/HTTPS from anywhere, SSH from my IP" \
  --vpc-id $VPC_ID --query 'GroupId' --output text)
MY_IP=$(curl -s https://checkip.amazonaws.com)/32
aws ec2 authorize-security-group-ingress --group-id $PUB_SG --protocol tcp --port 22  --cidr $MY_IP
aws ec2 authorize-security-group-ingress --group-id $PUB_SG --protocol tcp --port 80  --cidr 0.0.0.0/0
aws ec2 authorize-security-group-ingress --group-id $PUB_SG --protocol tcp --port 443 --cidr 0.0.0.0/0

PRIV_SG=$(aws ec2 create-security-group --group-name promisedland-prod-private-sg \
  --description "PromisedLand private: reachable only from public SG" \
  --vpc-id $VPC_ID --query 'GroupId' --output text)
aws ec2 authorize-security-group-ingress --group-id $PRIV_SG --protocol -1 --source-group $PUB_SG
```

---

## 6. EC2 Instance Launch

### Console

EC2 → Instances → Launch instance:
- Name: `promisedland-prod-app`
- AMI: Amazon Linux 2023 (or Ubuntu 24.04 LTS)
- Instance type: `t3.small` (or `t3.micro`)
- Key pair: create/select `promisedland-prod-key`
- Network settings → Edit:
  - VPC: `promisedland-prod-vpc`
  - Subnet: `promisedland-prod-public-1a`
  - Auto-assign public IP: **Enable**
  - Security group: `promisedland-prod-public-sg`
- Storage: add a second volume (EBS, gp3, e.g. 20 GiB) for `/srv/promisedland/data` — **uncheck "Delete on termination"** for the data volume
- Advanced details → IAM instance profile: `promisedland-prod-instance-profile`
- Launch

### CLI

```bash
AMI_ID=ami-xxxxxxxxxxxxxxxxx   # AL2023 or Ubuntu 24.04 for your region
KEY_NAME=promisedland-prod-key

INSTANCE_ID=$(aws ec2 run-instances \
  --image-id $AMI_ID --instance-type t3.small --key-name $KEY_NAME \
  --subnet-id $PUBA --security-group-ids $PUB_SG \
  --iam-instance-profile Name=promisedland-prod-instance-profile \
  --block-device-mappings 'DeviceName=/dev/sdf,Ebs={VolumeSize=20,VolumeType=gp3,DeleteOnTermination=false}' \
  --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=promisedland-prod-app}]" \
  --query 'Instances[0].InstanceId' --output text)
aws ec2 wait instance-running --instance-ids $INSTANCE_ID
aws ec2 describe-instances --instance-ids $INSTANCE_ID \
  --query 'Reservations[0].Instances[0].PublicIpAddress' --output text
```

### Elastic IP (optional, for stable IP)

```bash
EIP_ALLOC=$(aws ec2 allocate-address --domain vpc \
  --tag-specifications "ResourceType=elastic-ip,Tags=[{Key=Name,Value=promisedland-prod-eip}]" \
  --query 'AllocationId' --output text)
aws ec2 associate-address --instance-id $INSTANCE_ID --allocation-id $EIP_ALLOC
```

---

## 7. IAM Instance Profile (S3 + CloudWatch)

### Create policy, role, and instance profile

```bash
cat > /tmp/pl-s3-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:PutObject","s3:GetObject","s3:ListBucket"],
      "Resource": ["arn:aws:s3:::BUCKET_NAME","arn:aws:s3:::BUCKET_NAME/*"]
    },
    {
      "Effect": "Allow",
      "Action": ["logs:CreateLogStream","logs:PutLogEvents"],
      "Resource": "arn:aws:logs:*:*:log-group:/promisedland/prod:*"
    }
  ]
}
EOF
sed -i "s/BUCKET_NAME/promisedland-aswanidev-data/g" /tmp/pl-s3-policy.json

POLICY_ARN=$(aws iam create-policy --policy-name promisedland-prod-s3-policy \
  --policy-document file:///tmp/pl-s3-policy.json --query 'Policy.Arn' --output text)

aws iam create-role --role-name promisedland-prod-instance-role \
  --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}'

aws iam attach-role-policy --role-name promisedland-prod-instance-role --policy-arn $POLICY_ARN

aws iam create-instance-profile --instance-profile-name promisedland-prod-instance-profile
aws iam add-role-to-instance-profile --instance-profile-name promisedland-prod-instance-profile --role-name promisedland-prod-instance-role
```

### Attach to instance

```bash
aws ec2 associate-iam-instance-profile \
  --instance-id $INSTANCE_ID \
  --iam-instance-profile Name=promisedland-prod-instance-profile
```

If a profile is already attached, use `replace-iam-instance-profile-association` instead.

### Verify on the instance (after ~60s propagation)

```bash
aws sts get-caller-identity
aws s3 ls s3://promisedland-aswanidev-data
```

---

## 8. EBS Volume Mount

### Identify the volume

```bash
lsblk
sudo file -s /dev/nvme1n1   # "data" = unformatted; "ext4 filesystem" = already formatted
```

On Nitro instances the data volume is usually `/dev/nvme1n1` (root is `/dev/nvme0n1`). On older Xen instances it's `/dev/xvdf`. Always confirm with `lsblk`.

### Format (only on first use — DESTRUCTIVE)

```bash
sudo mkfs.ext4 -L promisedland-data /dev/nvme1n1
```

### Mount

```bash
sudo mkdir -p /srv/promisedland/data
sudo mount LABEL=promisedland-data /srv/promisedland/data
echo 'LABEL=promisedland-data /srv/promisedland/data ext4 defaults,nofail 0 2' | sudo tee -a /etc/fstab
sudo mount -a
```

### Set ownership (API = UID 100, edge = UID 101)

```bash
sudo chown 100:101 /srv/promisedland/data
sudo chmod 775 /srv/promisedland/data
ls -ld /srv/promisedland/data
# Expected: drwxrwxr-x ... 100 101 ... /srv/promisedland/data
```

### Verify UID 100 can write

```bash
docker run --rm -v /srv/promisedland/data:/data --user 100:101 alpine \
  sh -c 'touch /data/.writetest && rm /data/.writetest && echo "UID 100 write OK"'
```

### Re-verify mount persists across reboot

```bash
sudo umount /srv/promisedland/data
sudo mount -a
df -h /srv/promisedland/data
sudo findmnt --verify
```

---

## 9. Host Setup (Docker, Git, Clone)

### Amazon Linux 2023

```bash
sudo dnf install -y docker git
sudo systemctl enable --now docker
sudo usermod -aG docker ec2-user
# log out and back in for group to take effect

# Docker Compose v2 plugin
sudo mkdir -p /usr/local/lib/docker/cli-plugins
sudo curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 \
  -o /usr/local/lib/docker/cli-plugins/docker-compose
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
docker compose version

# Buildx (required by compose build; AL2023 ships an old version)
ARCH=$(uname -m); case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac
VER=$(curl -s https://api.github.com/repos/docker/buildx/releases/latest | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')
[ -z "$VER" ] && VER=0.19.3
sudo rm -f /usr/local/lib/docker/cli-plugins/docker-buildx
sudo curl -fSL "https://github.com/docker/buildx/releases/download/v${VER}/buildx-v${VER}.linux-${ARCH}" \
  -o /usr/local/lib/docker/cli-plugins/docker-buildx
sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-buildx
docker buildx version   # should be >= 0.17.0

# Clone
sudo mkdir -p /opt/promisedland && sudo chown ec2-user:ec2-user /opt/promisedland
git clone https://github.com/Aswanidev-vs/Promiseland.git /opt/promisedland
cd /opt/promisedland
```

### Ubuntu 24.04 (alternative)

```bash
sudo apt update && sudo apt install -y docker.io docker-compose-v2 git
sudo usermod -aG docker $USER && newgrp docker
sudo mkdir -p /opt/promisedland && sudo chown $USER:$USER /opt/promisedland
git clone https://github.com/Aswanidev-vs/Promiseland.git /opt/promisedland
cd /opt/promisedland
```

### sqlite3 (for backups)

```bash
# AL2023
sudo dnf install -y sqlite
# Ubuntu
sudo apt install -y sqlite3
```

### cron (AL2023 only — not installed by default)

```bash
sudo dnf install -y cronie
sudo systemctl enable --now crond
```

---

## 10. Configure `.env`

Create `/opt/promisedland/.env` (mode `600`, outside source control):

```bash
cat > /opt/promisedland/.env <<'EOF'
APP_ENV=production
PORT=8080
DATABASE_PATH=/data/promisedland.db
MEDIA_PATH=/data/media
CORS_ORIGIN=https://promisedland.aswanidev.xyz
SESSION_SECRET=<paste: openssl rand -hex 32>
FEATURE_ADULT_DATING=false
ADULT_AGE_ASSURANCE=none
PLATFORM_MODERATOR_EMAIL=<your email, optional>
EOF
chmod 600 /opt/promisedland/.env
```

Generate the secret:
```bash
openssl rand -hex 32
```

**Rules:**
- `SESSION_SECRET` must not be `change-me` or wrapped in `<...>` angle brackets. The API refuses production with the dev secret.
- `CORS_ORIGIN` must be the exact public HTTPS origin (e.g., `https://promisedland.aswanidev.xyz`), not `*` and not a placeholder like `https://<your-domain>`.
- `FEATURE_ADULT_DATING=false` and `ADULT_AGE_ASSURANCE=none` (R5 is a disabled scaffold).

---

## 11. TLS Setup (Caddy + Let's Encrypt)

The repo ships no TLS. Use Caddy for automatic Let's Encrypt certificates.

### 11.1 DNS

At your registrar, add an A record:

| Type | Name/Host | Value | TTL |
|---|---|---|---|
| A | `promisedland` | `51.21.248.103` (your instance IP) | 300 |

Verify:
```bash
dig +short promisedland.aswanidev.xyz
# Should return: 51.21.248.103
```

### 11.2 Create Caddyfile

```bash
mkdir -p /opt/promisedland/deploy/caddy
cat > /opt/promisedland/deploy/caddy/Caddyfile <<'EOF'
promisedland.aswanidev.xyz {
    reverse_proxy proxy:80
}
EOF
```

### 11.3 Create TLS compose override

```bash
cat > /opt/promisedland/deploy/compose/tls.yml <<'EOF'
name: promisedland

services:
  proxy:
    ports: !reset []

  caddy:
    image: caddy:2-alpine
    container_name: promisedland-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /opt/promisedland/deploy/caddy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
      - caddy_config:/config
    networks:
      - promisedland
    depends_on:
      - proxy

volumes:
  caddy_data:
  caddy_config:
EOF
```

### 11.4 Cloudflare (optional, recommended for production)

If you use Cloudflare DNS (free):
1. Add site → enter `aswanidev.xyz`.
2. Change nameservers at registrar to Cloudflare's.
3. Add A record: `promisedland` → `51.21.248.103`, proxy status **DNS only (grey cloud)** initially.
4. Later, enable orange cloud (proxy) and set SSL/TLS to **Full (strict)**. Install an Origin CA cert on the instance for end-to-end encryption.

---

## 12. First Deploy

```bash
cd /opt/promisedland
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml up -d --build
```

This builds all images, starts the API on `127.0.0.1:8080`, brings up the edge proxy internally, and starts Caddy on `:80`/`:443`. First boot applies all 22 embedded migrations automatically.

Watch build progress:
```bash
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml logs -f
```

---

## 13. Verification

```bash
curl -fsS https://promisedland.aswanidev.xyz/healthz   # {"status":"ok"}
curl -fsS https://promisedland.aswanidev.xyz/readyz     # {"status":"ready"}
curl -sI http://promisedland.aswanidev.xyz/ | head -3   # 301 redirect to HTTPS
curl -fsS https://promisedland.aswanidev.xyz/ | head -c 80  # the SPA

docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml ps
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml logs --tail=50 api
```

Then open `https://promisedland.aswanidev.xyz` in a browser:
- Register an account
- Create a community
- Upload an avatar/banner
- Exercise `/admin` with your staff account

---

## 14. Backups (Local + S3)

### 14.1 S3 Bucket Setup

```bash
export AWS_REGION=eu-north-1
export BUCKET=promisedland-aswanidev-data

aws s3 mb s3://$BUCKET --region $AWS_REGION
aws s3api put-public-access-block --bucket $BUCKET \
  --public-access-block-configuration "BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true"
aws s3api put-bucket-versioning --bucket $BUCKET --versioning-configuration Status=Enabled
```

### 14.2 IAM Policy + Role + Instance Profile

See [Section 7](#7-iam-instance-profile-s3--cloudwatch). Attach the instance profile to the EC2 instance. Verify on the instance:

```bash
aws sts get-caller-identity
aws s3 ls s3://promisedland-aswanidev-data
```

### 14.3 Run Backup

```bash
sudo env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data \
  /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups
```

Output should include:
```
wrote /srv/promisedland/backups/promisedland-<timestamp>.db.gz
wrote s3://promisedland-aswanidev-data/promisedland/promisedland-<timestamp>.db.gz
backup complete
```

The media directory warning (`media directory not found`) is normal on a fresh install with no uploads.

### 14.4 Verify Backup Integrity

```bash
cd /srv/promisedland/backups
gunzip -kf promisedland-*.db.gz
sqlite3 promisedland-*.db 'PRAGMA integrity_check;'
# Expected: ok
```

### 14.5 Schedule (Daily 03:17 UTC)

```bash
(crontab -l 2>/dev/null | grep -v 'promisedland/deploy/backup'; \
 echo "17 3 * * * sudo /usr/bin/env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups >> /var/log/promisedland-backup.log 2>&1") | crontab -
crontab -l
```

### 14.6 Lifecycle Rule (Expire Old Versions)

```bash
cat > /tmp/lifecycle.json <<'EOF'
{
  "Rules": [
    {
      "ID": "promisedland-backup-retention",
      "Status": "Enabled",
      "Filter": { "Prefix": "promisedland/" },
      "NoncurrentVersionExpiration": { "NoncurrentDays": 30 },
      "AbortIncompleteMultipartUpload": { "DaysAfterInitiation": 7 }
    }
  ]
}
EOF
aws s3api put-bucket-lifecycle-configuration --bucket promisedland-aswanidev-data --lifecycle-configuration file:///tmp/lifecycle.json
```

### 14.7 Rehearse a Restore

Before relying on backups, restore one into a scratch volume. See `docs/operations.md` under "Backup and restore". At minimum:

```bash
mkdir -p /tmp/scratch-restore
cp /srv/promisedland/backups/promisedland-<timestamp>.db.gz /tmp/scratch-restore/
cd /tmp/scratch-restore && gunzip *.db.gz
sqlite3 promisedland-*.db 'PRAGMA integrity_check;'
```

---

## 15. Monitoring

- `docker compose ... logs -f api` (structured JSON to stdout; Docker's `json-file` driver rotates by default).
- CloudWatch: install the CloudWatch agent or use the Docker log driver `awslogs` to ship container logs. Alarms on EC2 status check, disk utilisation, and the app health endpoint.
- Application health: `GET /healthz` (liveness) and `GET /readyz` (readiness, returns 503 `database_unavailable` if the DB ping fails).

```bash
curl -fsS https://promisedland.aswanidev.xyz/healthz
curl -fsS https://promisedland.aswanidev.xyz/readyz
```

---

## 16. Updates & Rollback

### Update

```bash
cd /opt/promisedland
git pull
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml up -d --build
```

### Rollback

Keep the previous image tag (or previous git commit). If a bad deploy applied a migration, restore the matching backup before swapping images — migrations are forward-only. See `docs/operations.md` for rollback and `SESSION_SECRET` rotation procedures.

---

## 17. Cost & Housekeeping

- Stop the instance when not actively testing: `aws ec2 stop-instances --instance-ids i-08ff3548150fb5714`.
- Set a billing budget with an email alert at a low threshold.
- S3 lifecycle rules on the backup prefix (expire after N days) and on media (move to cheaper storage after N days).
- The database is a single writer: do not run two API containers against the same volume.
- NAT Gateway costs ~$32/month. If you don't need private subnets, use the single-public-subnet layout to avoid this cost.

---

## 18. Security Checklist (Before Announcing)

- [ ] `SESSION_SECRET` is a fresh random value, not `change-me` or wrapped in `<...>`.
- [ ] `CORS_ORIGIN` is the exact public HTTPS origin, not `*` or a placeholder.
- [ ] `FEATURE_ADULT_DATING=false` and `ADULT_AGE_ASSURANCE=none`.
- [ ] Security Group: only 80/443 (and 22 from your IP, or SSM instead of SSH).
- [ ] No secret values committed (`.env` is gitignored, mode `600`).
- [ ] TLS live and redirecting HTTP → HTTPS.
- [ ] Backup cron installed and one restore rehearsed.
- [ ] Register + login + password reset exercised end to end (password reset requires an email provider — see Known Gaps).
- [ ] `PLATFORM_MODERATOR_EMAIL` set if you want the admin console.
- [ ] Remove stored root AWS credentials from the instance (`rm ~/.aws/credentials`) so it relies solely on the instance profile.

---

## 19. Troubleshooting

### 19.1 `compose build requires buildx 0.17.0 or later`

**Cause:** AL2023 ships an old Buildx (or none). The Compose plugin requires ≥ 0.17.0.

**Fix:** Install current Buildx (see Section 9). The download URL uses `amd64`/`arm64`, not `x86_64`/`aarch64` — map the architecture name.

### 19.2 `permission denied while trying to connect to the Docker API at unix:///var/run/docker.sock`

**Cause:** User not in the `docker` group in the current shell session.

**Fix:**
```bash
newgrp docker   # immediate, no logout
# or log out and back in
```

### 19.3 `No such image: promisedland-api:latest`

**Cause:** Build didn't produce/tag the image. Compose tries to start a container from a non-existent image.

**Fix:** Build explicitly first:
```bash
docker compose -f docker-compose.yml -f deploy/compose/prod.yml build api
docker compose -f docker-compose.yml -f deploy/compose/prod.yml up -d
```
Or build manually:
```bash
docker build -t promisedland-api:ci -f backend/Dockerfile backend/
```

### 19.4 `dependency failed to start: container promisedland-api is unhealthy`

**Cause:** The API container starts but fails its healthcheck. Check logs:
```bash
docker compose -f docker-compose.yml -f deploy/compose/prod.yml logs --tail=60 api
```

Common sub-causes:

#### 19.4a `failed to open database: unable to open database file: out of memory (14)`

**Cause:** `/srv/promisedland/data` is owned by `root:root` (mode 755) instead of `100:101` (mode 775). UID 100 (the API) can't create the SQLite DB file.

**Fix:**
```bash
sudo chown 100:101 /srv/promisedland/data
sudo chmod 775 /srv/promisedland/data
```

Also verify the EBS volume is actually mounted (not just an empty dir on root disk):
```bash
findmnt /srv/promisedland/data
df -h /srv/promisedland/data
```

If not mounted, mount it (see Section 8).

#### 19.4b SELinux blocking (AL2023)

SELinux is `Enforcing` by default on AL2023 and can block container access to bind-mounted host paths.

**Fix:**
```bash
getenforce   # if "Enforcing"
sudo setenforce 0   # temporary test
# if that fixes it, make permanent for the data dir:
sudo chcon -Rt svirt_sandbox_file_t /srv/promisedland/data
```

#### 19.4c `SESSION_SECRET` or `CORS_ORIGIN` issues

`prod.yml` requires both. The API also refuses production with the dev secret `change-me`. Ensure `.env` has real values without `<...>` angle brackets.

### 19.5 `sudo -u '#100' test -w ...` → `sudo: unknown user #100`

**Cause:** No passwd entry for UID 100 (normal for container UIDs). `sudo -u '#100'` doesn't work without a named user.

**Fix:** Use `setpriv` (no passwd entry needed) or verify via ownership/mode:
```bash
stat -c '%u:%g mode=%a' /srv/promisedland/data
# or
docker run --rm -v /srv/promisedland/data:/data --user 100:101 alpine sh -c 'touch /data/.t && rm /data/.t && echo OK'
```

### 19.6 EBS volume ownership reverted after reboot

**Cause:** The volume wasn't mounted (fstab entry missing or wrong), or the mount point directory was recreated.

**Fix:** Use `LABEL=` or `UUID=` in fstab (not `/dev/nvme1n1`), and re-apply ownership after mounting:
```bash
sudo chown 100:101 /srv/promisedland/data
sudo chmod 775 /srv/promisedland/data
```

### 19.7 `make_bucket failed: Unable to locate credentials`

**Cause:** No AWS credentials on the instance (no instance profile attached, or IMDS not reachable).

**Fix:** Attach an instance profile with S3 permissions (Section 7). Verify:
```bash
aws sts get-caller-identity
```

### 19.8 `InvalidInstanceID.Malformed: The instance ID 'None'`

**Cause:** The EC2 instance doesn't have the expected `Name` tag, so the filter returned `None`.

**Fix:** Find the instance by IP or list all instances:
```bash
aws ec2 describe-instances --filters "Name=ip-address,Values=51.21.248.103" --query 'Reservations[0].Instances[0].InstanceId' --output text
# or
aws ec2 describe-instances --query 'Reservations[*].Instances[*].[InstanceId,Tags[?Key==`Name`].Value|[0],PublicIpAddress]' --output table
```

### 19.9 `crontab: command not found`

**Cause:** AL2023 doesn't include cron by default.

**Fix:**
```bash
sudo dnf install -y cronie
sudo systemctl enable --now crond
```

### 19.10 `S3_BUCKET=<your-bucket>` → `your-bucket: No such file or directory`

**Cause:** Angle brackets `<` and `>` are shell redirection operators. Bash interpreted `<your-bucket>` as "read input from file named `your-bucket`".

**Fix:** Remove angle brackets. Use plain names: `S3_BUCKET=promisedland-aswanidev-data`. Quote if spaces: `S3_BUCKET='my bucket'`.

### 19.11 `sudo DATA_DIR=... cmd` drops environment variables

**Cause:** sudo's `env_reset` policy strips variables.

**Fix:** Use `sudo env VAR=val cmd` instead of `sudo VAR=val cmd`.

### 19.12 Caddy fails to obtain Let's Encrypt certificate

**Cause:** Port 80 not reachable from the internet, DNS not propagated, or rate limit hit.

**Fix:**
1. Verify DNS: `dig +short promisedland.aswanidev.xyz` → should return instance IP.
2. Verify port 80 reachable: `curl http://promisedland.aswanidev.xyz/` from outside.
3. Check SG allows 80/443 from `0.0.0.0/0`.
4. Check Caddy logs: `docker compose ... logs caddy`.
5. If rate-limited, wait a week or use Let's Encrypt staging endpoint for testing.

### 19.13 `aws sts get-caller-identity` returns root instead of instance profile

**Cause:** Stored root credentials in `~/.aws/credentials` take precedence over IMDS.

**Fix (optional, for better security):**
```bash
rm -f ~/.aws/credentials
aws sts get-caller-identity   # should now show the instance profile role
```

---

## 20. Known Gaps (Not Yet Automated)

| Gap | Impact | Mitigation Today |
|---|---|---|
| No deploy job / ECR publish | Deploy is `git pull` + `compose up --build` on the box | Manual; PRD §11.3 items 6–9 |
| No TLS automation | HTTPS must be set up by hand (Section 11) | Caddy (done) or Cloudflare/ALB |
| No email provider | Password reset and email verification don't work in production | Keep `DEV_EXPOSE_TOKENS` off; document reset as manual until SMTP is added |
| No CloudWatch agent config | Logs stay in Docker's `json-file` driver | Install the agent; see Section 15 |
| No restore rehearsed | Backup script exists but restore is untested | Run it once before relying on it |
| Media on local disk | No CDN, no quota, no AV scan, no re-encode | Documented as-is in PRD §10.6; S3 + lifecycle for now |
| Single instance | SQLite is single-writer; one box | Fine for beta; revisit if you need HA |
| No IaC | All provisioning is manual | Automate with Terraform/CloudFormation later |

---

## 21. Manual Website Implementation (Non-Docker Deployment)

> **This section covers deploying PromisedLand WITHOUT Docker** — building and running the Go API binary and the frontend SPA directly on the host. Use this if you prefer not to use containers, or need to debug at the process level. For the standard deployment, use Sections 9–13 (Docker).

### 21.1 Architecture (Manual)

```
┌─────────────────────────────────────────────────────┐
│  EC2 Instance (Amazon Linux 2023)                   │
│                                                     │
│  ┌──────────────┐    ┌──────────────────────────┐  │
│  │  Caddy (TLS)  │───▶│  Nginx (edge proxy)      │  │
│  │  :443 / :80   │    │  :80 (localhost only)    │  │
│  └──────────────┘    └──────┬───────────────────┘  │
│                             │                       │
│                    ┌────────┴────────┐              │
│                    │                 │              │
│              ┌─────▼─────┐   ┌──────▼──────┐       │
│              │  Go API   │   │  Static SPA │       │
│              │  :8080    │   │  (nginx)    │       │
│              │  (systemd)│   │             │       │
│              └─────┬─────┘   └─────────────┘       │
│                    │                               │
│              ┌─────▼─────┐                         │
│              │  SQLite   │                         │
│              │  /srv/    │                         │
│              │  promised │                         │
│              │  land/    │                         │
│              │  data/    │                         │
│              └───────────┘                         │
└─────────────────────────────────────────────────────┘
```

### 21.2 Prerequisites (Manual)

Install all build tools and runtime dependencies:

```bash
# AL2023
sudo dnf install -y golang nginx caddy sqlite git
# Install Bun (not Node/npm — this project uses Bun exclusively)
curl -fsSL https://bun.sh/install | bash
echo 'export BUN_INSTALL="$HOME/.bun"' >> ~/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Verify versions
go version          # Go 1.22+
bun --version       # Bun 1.4+
nginx -v
caddy version
```

For Ubuntu 24.04:
```bash
sudo apt install -y golang-go nginx caddy sqlite3 git
curl -fsSL https://bun.sh/install | bash
echo 'export BUN_INSTALL="$HOME/.bun"' >> ~/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### 21.3 Build the Go API Binary

```bash
cd /opt/promisedland

# Download dependencies
cd backend
go mod download

# Build (static binary, no CGO)
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /opt/promisedland/server ./cmd/server

# Verify binary
file /opt/promisedland/server
# Expected: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked

# Make executable
chmod +x /opt/promisedland/server
```

### 21.4 Build the Frontend SPA

```bash
cd /opt/promisedland/frontend

# Install dependencies (Bun — not npm)
bun install --frozen-lockfile

# Build for production
bun run build

# Output is in dist/ — this is the static SPA
ls -la dist/
```

### 21.5 Create the Data Directory

```bash
sudo mkdir -p /srv/promisedland/data/media
sudo chown 100:101 /srv/promisedland/data
sudo chmod 775 /srv/promisedland/data
sudo chown 100:101 /srv/promisedland/data/media
sudo chmod 775 /srv/promisedland/data/media
```

### 21.6 Create the Environment File

```bash
cat > /opt/promisedland/.env <<'EOF'
APP_ENV=production
PORT=8080
DATABASE_PATH=/srv/promisedland/data/promisedland.db
MEDIA_PATH=/srv/promisedland/data/media
CORS_ORIGIN=https://promisedland.aswanidev.xyz
SESSION_SECRET=<paste: openssl rand -hex 32>
FEATURE_ADULT_DATING=false
ADULT_AGE_ASSURANCE=none
PLATFORM_MODERATOR_EMAIL=<your email, optional>
EOF
chmod 600 /opt/promisedland/.env
```

**Note:** In manual mode, `DATABASE_PATH` and `MEDIA_PATH` point directly to host paths (not `/data/...` as in Docker).

### 21.7 Create a Systemd Service for the API

```bash
sudo tee /etc/systemd/system/promisedland-api.service <<'EOF'
[Unit]
Description=PromisedLand Go API
After=network.target

[Service]
Type=simple
User=app
Group=app
WorkingDirectory=/opt/promisedland
EnvironmentFile=/opt/promisedland/.env
ExecStart=/opt/promisedland/server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

Create the `app` user (UID 100) if it doesn't exist:

```bash
# Check if UID 100 exists
id 100 2>/dev/null || sudo useradd -u 100 -s /sbin/nologin app
sudo groupadd -g 101 app 2>/dev/null || true
sudo usermod -g 101 app 2>/dev/null || true

# Ensure data dir ownership
sudo chown -R 100:101 /srv/promisedland/data
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now promisedland-api
sudo systemctl status promisedland-api
```

### 21.8 Configure Nginx (Edge Proxy)

Create the nginx config to serve the SPA and proxy API requests:

```bash
sudo tee /etc/nginx/conf.d/promisedland.conf <<'EOF'
server {
    listen 80;
    server_name promisedland.aswanidev.xyz;

    # Static SPA
    root /opt/promisedland/frontend/dist;
    index index.html;

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API requests
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    # Health checks
    location /healthz {
        proxy_pass http://127.0.0.1:8080;
    }
    location /readyz {
        proxy_pass http://127.0.0.1:8080;
    }

    # Media files
    location /media/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
}
EOF

# Test config
sudo nginx -t

# Enable and start
sudo systemctl enable --now nginx
sudo systemctl reload nginx
```

### 21.9 Configure Caddy (TLS)

```bash
sudo tee /etc/caddy/Caddyfile <<'EOF'
promisedland.aswanidev.xyz {
    reverse_proxy 127.0.0.1:80
}
EOF

sudo systemctl enable --now caddy
sudo systemctl reload caddy
```

Caddy auto-provisions the Let's Encrypt cert and handles HTTP→HTTPS redirect.

### 21.10 Verify Manual Deployment

```bash
# API health
curl -fsS http://127.0.0.1:8080/healthz   # {"status":"ok"}
curl -fsS http://127.0.0.1:8080/readyz     # {"status":"ready"}

# Through nginx
curl -fsS http://localhost/healthz
curl -fsS http://localhost/ | head -c 80

# Through Caddy (HTTPS)
curl -fsS https://promisedland.aswanidev.xyz/healthz
curl -fsS https://promisedland.aswanidev.xyz/

# Service status
sudo systemctl status promisedland-api
sudo systemctl status nginx
sudo systemctl status caddy

# Logs
sudo journalctl -u promisedland-api -f
sudo journalctl -u nginx -f
```

### 21.11 Updating (Manual)

```bash
cd /opt/promisedland
git pull

# Rebuild API
cd backend
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /opt/promisedland/server ./cmd/server
sudo systemctl restart promisedland-api

# Rebuild frontend
cd /opt/promisedland/frontend
bun install --frozen-lockfile && bun run build

# Reload nginx (serves new static files)
sudo systemctl reload nginx
```

### 21.12 Application-Level Configuration

#### Admin Console

Set `PLATFORM_MODERATOR_EMAIL` in `.env` to bootstrap a platform moderator at startup. The user with that email gets moderator privileges on first login.

#### Feature Flags

| Variable | Default | Description |
|---|---|---|
| `FEATURE_ADULT_DATING` | `false` | Enable adult dating scaffold (R5). **Keep false** unless legal/safety review complete. |
| `ADULT_AGE_ASSURANCE` | `none` | Age assurance mode. `none` = deny everyone. `development_stub` = test double only (dev only). |
| `DEV_EXPOSE_TOKENS` | `false` | Return email-verification and password-reset tokens in API responses. **Never enable in production** — allows account takeover. |

#### Database Migrations

Migrations are embedded in the Go binary and run automatically on first boot. All 22 migrations apply on a fresh database. Migrations are forward-only — never downgrade.

#### Media Storage

Uploaded images are stored on disk at `MEDIA_PATH` (default `/data/media` in Docker, `/srv/promisedland/data/media` manually). They are never stored in SQLite. The directory is created lazily on first upload.

### 21.13 Systemd Service Management

```bash
# Start/stop/restart
sudo systemctl start promisedland-api
sudo systemctl stop promisedland-api
sudo systemctl restart promisedland-api

# Enable/disable auto-start on boot
sudo systemctl enable promisedland-api
sudo systemctl disable promisedland-api

# View logs
sudo journalctl -u promisedland-api -f          # follow
sudo journalctl -u promisedland-api --since "1 hour ago"  # last hour
sudo journalctl -u promisedland-api -n 50       # last 50 lines
```

### 21.14 Comparing Docker vs Manual

| Aspect | Docker (Sections 9–13) | Manual (Section 21) |
|---|---|---|
| Setup complexity | Lower (compose handles everything) | Higher (manual build, systemd, nginx config) |
| Isolation | Full (containers) | None (processes on host) |
| Updates | `git pull` + `compose up --build` | Rebuild binary + `systemctl restart` |
| Debugging | `docker logs`, `docker exec` | `journalctl`, direct process access |
| Resource overhead | Slightly higher (container runtime) | Lower (direct processes) |
| Reproducibility | High (Dockerfile pins versions) | Lower (depends on host packages) |
| Recommended for | Production, most users | Development, debugging, minimal environments |

**Recommendation:** Use Docker (Sections 9–13) for production. Use manual deployment for development, debugging, or environments where Docker isn't available.

---

## 22. AWS Console Setup (No CLI — Browser Only)

> **This section covers the entire AWS infrastructure setup using only the AWS Management Console (browser).** No AWS CLI required. Docker is still used for the application deployment (Sections 9–13). Use this if you prefer clicking through the UI instead of running CLI commands.

### 22.1 Region Selection

1. Log in to AWS Console: https://console.aws.amazon.com
2. In the top-right corner, select your region (e.g., **eu-north-1 Stockholm** or **us-east-1 N. Virginia**).
3. **Stay in this region for all steps below.** Mixing regions causes resources to not find each other.

### 22.2 Create the VPC

1. Search for **VPC** in the console search bar → open **VPC Dashboard**.
2. Click **Create VPC**.
3. Choose **VPC and more** (not "VPC only").
4. Fill in:

| Field | Value |
|---|---|
| Name tag auto-generation | `promisedland-prod` |
| IPv4 CIDR block | `10.0.0.0/16` |
| IPv6 CIDR block | No IPv6 CIDR block |
| Tenancy | Default |
| Number of Availability Zones | **2** |
| Number of public subnets | **2** |
| Number of private subnets | **2** |
| NAT gateways | **In 1 AZ** (cheaper) |
| VPC endpoints | None |
| DNS hostnames | ✅ Enable |
| DNS resolution | ✅ Enable |

5. Click **Create VPC**. Wait ~2 minutes for NAT Gateway to become "Available".
6. **Rename resources** (optional): Go to each resource → **Actions → Edit tags** → set `Name` tag:
   - VPC → `promisedland-prod-vpc`
   - Subnets → `promisedland-prod-public-1a`, `-public-1b`, `-private-1a`, `-private-1b`
   - IGW → `promisedland-prod-igw`
   - NAT Gateway → `promisedland-prod-nat`
   - Route tables → `promisedland-prod-public-rt`, `promisedland-prod-private-rt`

### 22.3 Verify Route Table Associations

1. Go to **VPC → Route tables**.
2. Click **`promisedland-prod-public-rt`**:
   - **Routes tab**: should show `10.0.0.0/16 → local` and `0.0.0.0/0 → igw-xxxx`.
   - **Subnet associations tab**: should list both public subnets. If missing → **Edit subnet associations** → check both → Save.
3. Click **`promisedland-prod-private-rt`**:
   - **Routes tab**: should show `10.0.0.0/16 → local` and `0.0.0.0/0 → nat-xxxx`.
   - **Subnet associations tab**: should list both private subnets. If missing → associate them.
4. Click the **main (default)** route table (shows "Main: Yes"):
   - Should have **no** `0.0.0.0/0` route — only `10.0.0.0/16 → local`. This is correct.

### 22.4 Create Security Groups

**Public SG:**
1. Go to **VPC → Security Groups → Create security group**.
2. Name: `promisedland-prod-public-sg`, VPC: your VPC.
3. Inbound rules:
   - SSH (TCP 22) → Source: **My IP**
   - HTTP (TCP 80) → Source: **Anywhere-IPv4 (0.0.0.0/0)**
   - HTTPS (TCP 443) → Source: **Anywhere-IPv4 (0.0.0.0/0)**
4. Outbound: leave default (All traffic).
5. Click **Create security group**.

**Private SG:**
1. Create another security group: `promisedland-prod-private-sg`.
2. Inbound rules:
   - All traffic → Source: select **`promisedland-prod-public-sg`** from the dropdown (not a CIDR).
3. Outbound: leave default.
4. Click **Create security group**.

### 22.5 Create S3 Bucket

1. Search for **S3** → open S3 console.
2. Click **Create bucket**.
3. Settings:
   - Bucket name: `promisedland-aswanidev-data` (must be globally unique)
   - Region: your region (eu-north-1)
   - Block Public Access: **Block all public access** ✅ (keep all checkboxes checked)
   - Versioning: **Enable**
   - Default encryption: **Server-side encryption with Amazon S3 managed keys (SSE-S3)**
4. Click **Create bucket**.

5. **Lifecycle rules** (for backup retention and media tiering):
   - Open your bucket → **Management** tab → **Create lifecycle rule**.
   - Rule 1 — Backup retention:
     - Rule name: `promisedland-backup-retention`
     - Prefix: `promisedland/`
     - Noncurrent version expiration: **30 days**
     - Abort incomplete multipart uploads: **7 days**
     - Click **Create rule**.
   - Rule 2 — Media tiering (optional, saves cost on old uploads):
     - Rule name: `promisedland-media-tiering`
     - Prefix: `media/`
     - Transitions: **30 days → Standard-IA**, **90 days → Glacier**
     - Click **Create rule**.

6. **CORS configuration** (if the frontend ever uploads directly to S3):
   - Open your bucket → **Permissions** tab → **CORS configuration** → **Edit**.
   - Paste:
     ```json
     [
       {
         "AllowedHeaders": ["*"],
         "AllowedMethods": ["GET", "PUT", "POST"],
         "AllowedOrigins": ["https://promisedland.aswanidev.xyz"],
         "ExposeHeaders": ["ETag"],
         "MaxAgeSeconds": 3600
       }
     ]
     ```
   - Click **Save changes**.

### 22.5b Verify S3 Access (After IAM Setup)

Once the instance profile is attached (Section 22.6), SSH into the instance and verify:

```bash
aws sts get-caller-identity
# Should return your account ID and the instance profile role ARN
# (NOT "root" — if it shows root, remove ~/.aws/credentials)

aws s3 ls s3://promisedland-aswanidev-data
# Should return empty (no error)

# Test upload/download
echo "test" > /tmp/s3test.txt
aws s3 cp /tmp/s3test.txt s3://promisedland-aswanidev-data/test.txt
aws s3 cp s3://promisedland-aswanidev-data/test.txt /tmp/s3test-download.txt
aws s3 rm s3://promisedland-aswanidev-data/test.txt
cat /tmp/s3test-download.txt   # should print "test"
```

If `get-caller-identity` fails, the instance profile is not attached or hasn't propagated yet (wait 60 seconds and retry).

### 22.5c Run the Backup (Console Setup)

After S3 access is verified:

```bash
# Install sqlite3 (AL2023)
sudo dnf install -y sqlite

# Install cron (AL2023 — not included by default)
sudo dnf install -y cronie
sudo systemctl enable --now crond

# Run backup manually
sudo env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data \
  /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups

# Schedule daily at 03:17 UTC
(crontab -l 2>/dev/null | grep -v 'promisedland/deploy/backup'; \
 echo "17 3 * * * sudo /usr/bin/env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups >> /var/log/promisedland-backup.log 2>&1") | crontab -
crontab -l
```

Verify the backup landed in S3:

```bash
aws s3 ls s3://promisedland-aswanidev-data/promisedland/
# Should show: promisedland-<timestamp>.db.gz
```

### 22.5d S3 Cost & Lifecycle Summary

| Item | Approx. Cost |
|---|---|
| S3 storage (daily DB backup, ~small gzipped file) | ~$0.01/month |
| S3 PUT/GET requests | ~$0.01/month |
| Lifecycle transition to Standard-IA (after 30 days) | ~50% cheaper storage |
| Lifecycle transition to Glacier (after 90 days) | ~80% cheaper storage |
| NAT Gateway (if using private subnets) | ~$32/month + data |

The lifecycle rules automatically move old media to cheaper storage tiers and expire old backup versions. No manual cleanup needed.

### 22.6 Create IAM Role + Instance Profile

1. Search for **IAM** → open IAM console.
2. **Create policy:**
   - Left sidebar → **Policies** → **Create policy**.
   - Click **JSON** tab.
   - Paste:
     ```json
     {
       "Version": "2012-10-17",
       "Statement": [
         {
           "Effect": "Allow",
           "Action": ["s3:PutObject","s3:GetObject","s3:ListBucket"],
           "Resource": ["arn:aws:s3:::promisedland-aswanidev-data","arn:aws:s3:::promisedland-aswanidev-data/*"]
         },
         {
           "Effect": "Allow",
           "Action": ["logs:CreateLogStream","logs:PutLogEvents"],
           "Resource": "arn:aws:logs:*:*:log-group:/promisedland/prod:*"
         }
       ]
     }
     ```
   - Click **Next** → Name: `promisedland-prod-s3-policy` → **Create policy**.
3. **Create role:**
   - Left sidebar → **Roles** → **Create role**.
   - Trusted entity type: **AWS service** → Use case: **EC2**.
   - Click **Next**.
   - Search and select `promisedland-prod-s3-policy`.
   - Click **Next** → Role name: `promisedland-prod-instance-role` → **Create role**.
4. **Create instance profile:**
   - Left sidebar → **Instance profiles** → **Create instance profile**.
   - Name: `promisedland-prod-instance-profile`.
   - Click **Create instance profile**.
   - Click on the profile → **Add role** → select `promisedland-prod-instance-role` → **Add role**.

### 22.7 Launch EC2 Instance

1. Search for **EC2** → open EC2 console.
2. Click **Launch instance**.
3. Settings:

| Field | Value |
|---|---|
| Name | `promisedland-prod-app` |
| AMI | Amazon Linux 2023 (or Ubuntu 24.04 LTS) |
| Instance type | `t3.small` (or `t3.micro`) |
| Key pair | Create new → name: `promisedland-prod-key` → download `.pem` |
| Network settings | Edit → VPC: `promisedland-prod-vpc` → Subnet: `promisedland-prod-public-1a` → Auto-assign public IP: **Enable** → Security group: `promisedland-prod-public-sg` |
| Storage | Add new volume: 20 GiB, gp3, **uncheck "Delete on termination"** (for data volume) |
| Advanced details | IAM instance profile: `promisedland-prod-instance-profile` |

4. Click **Launch instance**.
5. Wait for **Instance state: Running** and **Status checks: 2/2 passed**.

### 22.8 Allocate Elastic IP (Optional)

1. EC2 → **Elastic IPs** → **Allocate Elastic IP address**.
2. Tag: `Name = promisedland-prod-eip`.
3. Click **Allocate**.
4. Select the EIP → **Actions → Associate Elastic IP address** → select your instance → **Associate**.

### 22.9 Route 53 (Optional, if you have a domain)

1. Search for **Route 53** → open console.
2. **Hosted zones** → **Create hosted zone** → Domain name: `aswanidev.xyz` → **Create hosted zone**.
3. Note the 4 nameservers (NS records) — update these at your registrar.
4. **Create record** → Type: A → Name: `promisedland` → Value: `51.21.248.103` (your instance IP or Elastic IP) → TTL: 300 → **Create records**.

### 22.10 Connect via SSH

```bash
# From your laptop
chmod 400 ~/Downloads/promisedland-prod-key.pem
ssh -i ~/Downloads/promisedland-prod-key.pem ec2-user@51.21.248.103
```

For Ubuntu AMI, use `ubuntu` instead of `ec2-user`.

### 22.11 SSM Session Manager (Alternative to SSH — no port 22 needed)

If you don't want to open port 22:
1. Attach the `AmazonSSMManagedInstanceCore` policy to the instance role (`promisedland-prod-instance-role`).
2. Go to **Systems Manager → Session Manager → Start session** → select your instance.
3. Remove the SSH inbound rule from the security group.

### 22.12 Console Verification Checklist

After completing the above, verify:

- [ ] VPC exists with 2 public + 2 private subnets
- [ ] Route tables: public RT → IGW, private RT → NAT, main RT → no internet route
- [ ] Security groups: public SG has 22/80/443, private SG references public SG
- [ ] EC2 instance running in public subnet with public IP
- [ ] EBS data volume attached (separate from root)
- [ ] S3 bucket created with public access blocked and versioning enabled
- [ ] IAM instance profile attached to EC2 instance
- [ ] SSH/SSM access works
- [ ] DNS resolves: `dig +short promisedland.aswanidev.xyz` → instance IP

---

## Day-to-Day Commands (Quick Reference)

```bash
# Status
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml ps

# Logs
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml logs -f api

# Update deploy
cd /opt/promisedland && git pull && docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml up -d --build

# Restart
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml restart

# Health check
curl -fsS https://promisedland.aswanidev.xyz/healthz
curl -fsS https://promisedland.aswanidev.xyz/readyz

# Backup (manual)
sudo env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data \
  /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups

# Stop instance (save cost)
aws ec2 stop-instances --instance-ids i-08ff3548150fb5714
```

---

*This guide consolidates all steps and troubleshooting from a real deployment session on Amazon Linux 2023 (eu-north-1). It is designed to be self-contained: a deleted instance can be rebuilt from scratch using this document alone.*

```bash
# Docker deployment (standard)
# Status
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml ps

# Logs
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml logs -f api

# Update deploy
cd /opt/promisedland && git pull && docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml up -d --build

# Restart
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml restart

# Health check
curl -fsS https://promisedland.aswanidev.xyz/healthz
curl -fsS https://promisedland.aswanidev.xyz/readyz

# Backup (manual)
sudo env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data \
  /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups

# Stop instance (save cost)
aws ec2 stop-instances --instance-ids i-08ff3548150fb5714

# Manual deployment equivalents
sudo systemctl status promisedland-api
sudo systemctl restart promisedland-api
sudo journalctl -u promisedland-api -f
```

---

*This guide consolidates all steps and troubleshooting from a real deployment session on Amazon Linux 2023 (eu-north-1). It is designed to be self-contained: a deleted instance can be rebuilt from scratch using this document alone.*

```bash
# Status
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml ps

# Logs
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml logs -f api

# Update deploy
cd /opt/promisedland && git pull && docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml up -d --build

# Restart
docker compose -f docker-compose.yml -f deploy/compose/prod.yml -f deploy/compose/tls.yml restart

# Health check
curl -fsS https://promisedland.aswanidev.xyz/healthz
curl -fsS https://promisedland.aswanidev.xyz/readyz

# Backup (manual)
sudo env DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=promisedland-aswanidev-data \
  /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups

# Stop instance (save cost)
aws ec2 stop-instances --instance-ids i-08ff3548150fb5714
```

---

*This guide consolidates all steps and troubleshooting from a real deployment session on Amazon Linux 2023 (eu-north-1). It is designed to be self-contained: a deleted instance can be rebuilt from scratch using this document alone.*
