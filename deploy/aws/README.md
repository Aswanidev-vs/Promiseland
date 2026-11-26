# AWS hosting guide — single EC2 instance

> PromisedLand is a small, single-writer SQLite app. This guide puts the
> whole stack (`api`, `web`, and the optional edge `proxy`) on **one** EC2
> instance behind HTTPS. It is deliberately simple: one box, one EBS volume,
> one compose file. Scale out later only if you need to.
>
> Nothing here is automated yet (no Terraform, no deploy job, no IaC). Every
> step is a command you run by hand the first time; automate afterwards. All
> values in `<angle brackets>` are yours to fill in.

## 0. What you are deploying

Three images, built from this repo:

| Image | Source | Role |
|---|---|---|
| `promisedland-api:ci` | `backend/Dockerfile` | Go API on `:8080`, runs as UID 100 (`app`) |
| `promisedland-web:ci` | `frontend/Dockerfile` | Static SPA + nginx on `:80` (UID 101) |
| `promisedland-edge:ci` | `deploy/nginx/Dockerfile` | **The one you actually run in production**: SPA + nginx that proxies `/api/`, `/ws/`, `/healthz`, `/readyz`, `/media/` to the API. Listens on `:80`, UID 101 |

The prod compose override (`deploy/compose/prod.yml`) activates the edge
`proxy` profile, publishes **only port 80** on the host, sets
`APP_ENV=production`, and refuses to start without `CORS_ORIGIN` and
`SESSION_SECRET`. The API is reachable on `127.0.0.1:8080` for diagnostics
only.

SQLite lives on an EBS-backed host path (`/srv/promisedland/data` →
container `/data`); user media lands in the same directory under `media/`.
The API runs as UID 100, so that directory must be owned by `100:101`.

## 1. Prerequisites

- An AWS account with a billing budget/alert set (set this first — a stray
  instance running for a month is the most common surprise).
- A domain name (optional but recommended; TLS is painful without one).
- Local tools: AWS CLI v2, `ssh`, `docker` (for building images locally if
  you don't want to build on the instance).
- This repository checked out locally.

## 2. Provision (AWS console or CLI)

Create these in your chosen region. Use the smallest instance that fits;
PromisedLand is a single-writer SQLite app and idles cheaply.

1. **VPC + Security Group.** Inbound: `TCP 22` (SSH) from **your IP only**
   (or replace SSH with SSM Session Manager — see §12), `TCP 80` and `TCP 443`
   from `0.0.0.0/0`. Outbound: default (all). Nothing else.
2. **EC2 instance.** Amazon Linux 2023 or Ubuntu 24.04 LTS, smallest
   suitable size, key pair for SSH, the SG above, in a public subnet with a
   public IP (or an Elastic IP you keep).
3. **EBS volume.** Separate from the root volume, sized for your data (SQLite
   + media). Attach it to the instance.
4. **S3 bucket** for media/backups, e.g. `promisedland-<yourname>-data`, with
   **Block Public Access on** and a lifecycle rule to expire old versions.
5. **ECR repositories** (optional but recommended): `promisedland/api`,
   `promisedland/web`, `promisedland/edge`. Otherwise you build on the
   instance.
6. **IAM instance profile** attached to the EC2 instance, least-privilege:
   - `s3:PutObject`, `s3:GetObject`, `s3:ListBucket` on your bucket (media +
     backups).
   - `logs:CreateLogStream`, `logs:PutLogEvents` on a CloudWatch log group.
   - No `AdministratorAccess`, no long-lived access keys on the instance.
7. **Route 53 record** (if you have a domain): `A`/`AAAA` to the instance
   (or to your load balancer).

## 3. Host setup (SSH in)

```bash
# Ubuntu 24.04
sudo apt update && sudo apt install -y docker.io docker-compose-v2 git
sudo usermod -aG docker $USER && newgrp docker

# Mount the EBS volume (example: /dev/xvdf). Format only on first use.
sudo mkfs.ext4 /dev/xvdf
sudo mkdir -p /srv/promisedland/data
sudo mount /dev/xvdf /srv/promisedland/data
echo '/dev/xvdf /srv/promisedland/data ext4 defaults,nofail 0 2' | sudo tee -a /etc/fstab

# The API runs as UID 100; the edge runs as UID 101. Make the data dir
# writable by the API and readable by the edge.
sudo chown 100:101 /srv/promisedland/data
sudo chmod 775 /srv/promisedland/data

# Clone
sudo mkdir -p /opt/promisedland && sudo chown $USER:$USER /opt/promisedland
git clone https://github.com/Aswanidev-vs/Promiseland.git /opt/promisedland
cd /opt/promisedland
```

Amazon Linux 2023 is the same shape with `dnf install docker` and `systemctl
enable --now docker`.

## 4. Configure (`.env`)

Create `/opt/promisedland/.env` (outside source control, mode `600`):

```bash
APP_ENV=production
PORT=8080
DATABASE_PATH=/data/promisedland.db
MEDIA_PATH=/data/media
CORS_ORIGIN=https://<your-domain>
SESSION_SECRET=<generate: openssl rand -hex 32>
FEATURE_ADULT_DATING=false
ADULT_AGE_ASSURANCE=none
PLATFORM_MODERATOR_EMAIL=<your email, optional, for the admin console>
```

`prod.yml` refuses to start without `CORS_ORIGIN` and `SESSION_SECRET`, so a
missing value fails loudly instead of silently running with dev defaults. The
API also refuses `production` with the dev `SESSION_SECRET`.

## 5. TLS

**The repo ships no TLS.** `deploy/nginx/default.conf` listens on `80` only.
Pick one and do it before you announce the URL:

- **Option A — Cloudflare in front (recommended, simplest).** Point your DNS
  at Cloudflare, proxy the record (orange cloud), set SSL/TLS to **Full
  (strict)**, and install an Origin CA certificate on the instance. Free,
  renews automatically, hides the origin IP. No code change needed.
- **Option B — ALB + ACM.** Put an Application Load Balancer in front of the
  instance, request a certificate in ACM, and terminate TLS on the ALB. The
  instance only needs to answer `:80`. No code change.
- **Option C — Caddy on the instance.** Add a Caddy container in front of the
  edge proxy; it obtains and renews Let's Encrypt certs automatically. Needs a
  small Compose change and a domain pointing at the instance.

Do not run the public site on plain HTTP.

## 6. First deploy

```bash
cd /opt/promisedland
docker compose -f docker-compose.yml -f deploy/compose/prod.yml up -d --build
```

This builds the API, web and edge images on the instance, starts the API on
`127.0.0.1:8080`, and brings up the edge proxy on `:80`. The first boot
applies all 22 embedded migrations automatically.

If you built images in CI and pushed to ECR, replace `--build` with
`docker compose ... pull` and set the image names in `.env` (the Compose
file uses `build:` by default; switch to `image:` when you publish).

## 7. Verify

```bash
curl -fsS http://127.0.0.1:8080/healthz   # {"status":"ok"}
curl -fsS http://127.0.0.1:8080/readyz     # {"status":"ready"}
curl -fsS http://localhost/healthz          # through the edge
curl -fsS http://localhost/ | head -c 80    # the SPA
docker compose -f docker-compose.yml -f deploy/compose/prod.yml ps
docker compose -f docker-compose.yml -f deploy/compose/prod.yml logs --tail=50 api
```

Then open `https://<your-domain>` in a browser, register an account, create a
community, upload an avatar/banner, and exercise `/admin` with your staff
account.

## 8. Backups

Install `sqlite3` on the host (`sudo apt install -y sqlite3`) and run:

```bash
DATA_DIR=/srv/promisedland/data KEEP_DAYS=30 S3_BUCKET=<your-bucket> \\
  /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups
```

Schedule it (daily, 03:17 UTC):

```cron
17 3 * * * /opt/promisedland/deploy/backup/backup.sh /srv/promisedland/backups >> /var/log/promisedland-backup.log 2>&1
```

The script snapshots the live database with SQLite's online backup API (safe
in WAL mode), runs `PRAGMA integrity_check`, gzips it, tars the media
directory, optionally copies to S3, and applies retention. **Run one restore
into a scratch volume before you rely on it** — the procedure is in
`docs/operations.md` under "Backup and restore", and it is still marked
"To test before launch" because it has not been exercised on a real volume
yet.

## 9. Updates and rollback

Until CI publishing exists (PRD §11.3 items 6–9), the deploy is manual. The
instance tracks **`main`**, the release branch — development happens on `dev`
and reaches `main` through a pull request, so the box only ever runs promoted
commits:

```bash
cd /opt/promisedland
git switch main          # once; clone checks out the default branch
git pull
docker compose -f docker-compose.yml -f deploy/compose/prod.yml up -d --build
```

Rollback: keep the previous image tag (or the previous git commit). If a bad
deploy applied a migration, restore the matching backup before swapping
images — migrations are forward-only. See `docs/operations.md` for the
rollback and `SESSION_SECRET` rotation procedures.

## 10. Monitoring

- `docker compose ... logs -f api` (structured JSON to stdout; Docker's
  `json-file` driver rotates by default — set `--log-opt max-size` if you
  want explicit caps).
- CloudWatch: install the CloudWatch agent or use the Docker log driver
  `awslogs` to ship container logs; alarms on EC2 status check, disk
  utilisation and the app health endpoint.
- Application health: `GET /healthz` (liveness) and `GET /readyz`
  (readiness, returns 503 `database_unavailable` if the DB ping fails).

## 11. Cost and housekeeping

- Stop the instance when you are not actively testing (`aws ec2 stop-instances`).
- Set a billing budget with an email alert at a low threshold.
- S3 lifecycle rules on the backup prefix (expire after N days) and on media
  (move to cheaper storage after N days).
- The database is a single writer: do not run two API containers against the
  same volume.

## 12. Security checklist before you announce it

- [ ] `SESSION_SECRET` is a fresh random value, not `change-me`.
- [ ] `CORS_ORIGIN` is the exact public HTTPS origin, not `*`.
- [ ] `FEATURE_ADULT_DATING=false` and `ADULT_AGE_ASSURANCE=none` (R5 is a
      disabled scaffold; do not enable without the legal/safety review in
      `docs/adult-safety.md`).
- [ ] Security Group: only 80/443 (and 22 from your IP, or SSM instead of SSH).
- [ ] No secret values committed (`.env` is gitignored).
- [ ] TLS live and redirecting HTTP → HTTPS.
- [ ] Backup cron installed and one restore rehearsed.
- [ ] Register + login + password reset exercised end to end (note: password
      reset requires an email provider — see §13).
- [ ] `PLATFORM_MODERATOR_EMAIL` set if you want the admin console.

## 13. Known gaps (not yet automated)

| Gap | Impact | Mitigation today |
|---|---|---|
| No deploy job / ECR publish | Deploy is `git pull` + `compose up --build` on the box | Manual; PRD §11.3 items 6–9 |
| No TLS automation | HTTPS must be set up by hand (§5) | Cloudflare/ALB/Caddy |
| No email provider | Password reset and email verification do not work in production | Keep `DEV_EXPOSE_TOKENS` off; document reset as manual until SMTP is added |
| No CloudWatch agent config | Logs stay in Docker's `json-file` driver | Install the agent; see §10 |
| No restore rehearsed | Backup script exists but restore is untested | Run it once before you rely on it |
| Media on local disk | No CDN, no quota, no AV scan, no re-encode | Documented as-is in PRD §10.6; S3 + lifecycle for now |
| Single instance | SQLite is single-writer; one box | Fine for beta; revisit if you need HA |

## CI/CD status

The workflow (`.github/workflows/ci.yml`) covers PRD §11.3 items 1–5: formatting,
vet, `go test -race`, `govulncheck`, the frontend build and tests, `bun audit`,
all three images built and Trivy-scanned, and a smoke test that starts the
stack and waits for `/healthz`. Publish to ECR, scripted EC2 deployment,
post-deploy health checks and rollback automation are **deferred** per PRD
§11.3 items 6–9 and should be added before relying on this topology for a
public launch.
