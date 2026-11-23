# AWS deployment notes (target topology)

> These are planning notes grounded in PRD §11. No AWS account, resource IDs, credentials, account numbers, domains or IPs are included — fill those in at deployment time from your own account. This topology is **not yet exercised**.

## Shape

One EC2 instance runs Docker Compose with `api`, `web` and the optional edge `proxy` (PRD §10.2, §11.1). Frontend and backend remain separate images/containers. SQLite is a single writer on an EBS-backed path; user media goes to S3.

| Concern | Initial choice | Notes |
|---|---|---|
| Compute | EC2, smallest suitable instance | Check current pricing, free-tier eligibility and regional availability; set a billing budget/alerts; stop test resources when idle |
| Storage | EBS volume | Mount at `/srv/promisedland/data` and point the prod Compose override at it. Never keep SQLite in the disposable container filesystem |
| Networking | VPC + Security Groups | Open only what is needed: HTTP/HTTPS to the host, SSH restricted to trusted addresses (or use managed access). The API port is not publicly exposed in the prod override |
| TLS | nginx/Caddy with a certificate workflow, or LB termination | PRD §11.1 lists nginx/Caddy; HTTPS is required for all public traffic |
| Registry | ECR | Store versioned `backend` and `frontend` images; keep the previous known-good tag for rollback |
| Media | S3 | Block public access by default; store object keys in SQLite; validate uploads server-side; lifecycle rules + budget alerts for cost |
| Identity | IAM | Least-privilege instance profile (S3 media, CloudWatch logs) and a restricted deployment identity. Prefer GitHub Actions OIDC over long-lived access keys (PRD §11.3) |
| Monitoring | CloudWatch | Container/application logs, EC2 status, disk, memory where the agent supports it, and an app-health alarm |
| DNS | Route 53 (optional) | Only if a custom domain is used |

## Deployment outline (to be automated)

1. Provision VPC/SG, EC2, EBS, ECR repositories, S3 bucket and CloudWatch log group in your own account.
2. Attach an instance profile with only the permissions the running containers need.
3. On the host: install Docker + Compose v2, mount EBS at `/srv/promisedland/data`, place the production `.env` outside source control (use a secrets manager where available), and clone/checkout the repo.
4. Pull or build images, then:
   ```bash
   docker compose -f docker-compose.yml -f deploy/compose/prod.yml up -d --build
   ```
   The prod override activates the edge proxy, publishes only port 80, sets `APP_ENV=production`, and requires `CORS_ORIGIN` and `SESSION_SECRET` to be present.
5. Verify `healthz`/`readyz`, the edge route, and logs. Keep the prior image tag for rollback; see `docs/operations.md` for backup/restore and rollback caveats.

## CI/CD status

The current workflow (`.github/workflows/ci.yml`) runs checks only. Image build/scan/publish to ECR, scripted EC2 deployment, post-deploy health checks and rollback automation are **deferred** per PRD §11.3 and should be added before relying on this topology for a public launch.
