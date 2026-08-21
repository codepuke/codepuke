# codepuke infrastructure plan

How codepuke.com runs on the homelab k3s cluster. Every choice below follows a
pattern already proven in `../homelab`; the reference workload is potatoelf and
the reference database consumer is authentik. Where this plan requires a change
in `../homelab`, it is listed in "Required homelab changes" for Dan to make.
This repo never writes to `../homelab`.

## Environments and namespaces

Two environments, one cluster:

| Env  | Namespace      | Hostname          | Database       | Deployed from |
| ---- | -------------- | ----------------- | -------------- | ------------- |
| dev  | `codepuke-dev` | dev.codepuke.com  | `codepuke_dev` | `deploy/overlays/dev` at `main` |
| prod | `codepuke`     | codepuke.com      | `codepuke`     | `deploy/base` pinned by tag |

Namespace equals homelab workload directory name, per homelab convention.
Manifests live in this repo: `deploy/base/` plus `deploy/overlays/dev/`.
The homelab side is a thin Kustomize overlay (`workloads/codepuke/`) that
references `deploy/base` by git URL and tag, and two ArgoCD Applications
(`apps/workloads/codepuke.yaml`, `apps/workloads/codepuke-dev.yaml`). The dev
Application tracks this repo's `main` at `deploy/overlays/dev`, exactly like
potatoelf-dev.

## Tunnel ingress routing

The cloudflared tunnel (`homelab-k3s-http`, UUID
`9d70f230-1de9-41da-9b79-525ea898bdfa`) is a dumb pipe: one catch-all rule to
Traefik. Nothing in `infrastructure/cloudflared/` changes for codepuke.
All three DNS records (apex, www, dev) already exist as proxied CNAMEs to the
tunnel, so hostname routing is entirely plain `Ingress`:

- **prod** (`deploy/base/ingress.yaml`): `ingressClassName: traefik`, hosts
  `codepuke.com` and `www.codepuke.com`, both to the codepuke Service. No
  `tls:` block; TLS terminates at the Cloudflare edge.
- **www to apex redirect**: a Traefik `redirectRegex` Middleware in
  `deploy/base/middlewares.yaml`
  (`^https?://www\.codepuke\.com/(.*)` to `https://codepuke.com/${1}`,
  permanent), attached with the annotation
  `traefik.ingress.kubernetes.io/router.middlewares: codepuke-redirect-www-to-apex@kubernetescrd`.
  Same shape as potatoelf. The annotation applies to every router the Ingress
  generates, which is fine here because the apex router redirecting www hosts
  never matches its own rule.
- **dev** (`deploy/overlays/dev/`): Ingress patched to host `dev.codepuke.com`
  only, middleware annotation removed.

Traefik already trusts pod-CIDR forwarded headers, so
`X-Forwarded-Proto: https` reaches the app. The OIDC redirect URL derivation
in stage 6 depends on this and it is already in place.

## Pods

One Deployment per environment (stateless, Postgres holds all state; no PVC,
no nodeSelector). Pod conventions from homelab: `app.kubernetes.io/name:
codepuke` on the pod template (Loki label mapping), `environment: dev|prod`
labels, non-root uid/gid 1000, read-only root filesystem, drop ALL caps,
seccomp RuntimeDefault, emptyDir at `/tmp`, requests and limits set, probe
split of `/healthz` (liveness, no DB) and `/readyz` (readiness, `SELECT 1`).

### Mermaid sidecar

A second container in the same pod, exposing mermaid rendering as HTTP on
localhost. The Go server calls it only at publish and sync time, never on a
request path, and inlines the returned SVG (cached by source hash).

Image: `yuzutech/kroki-mermaid`, the Kroki mermaid companion. It is a
maintained HTTP wrapper around mermaid rendering (POST diagram source, receive
SVG), which means zero shim code to own. Decided. Stages 2 through 4 still
code against a `MermaidRenderer` interface with a no-op implementation; stage
5 wires in the HTTP client.

## PostgreSQL

The shared CNPG cluster (`postgres` in namespace `postgres`, 3 instances,
PG 18). No per-app Postgres, per homelab rule.

- Databases `codepuke` and `codepuke_dev`, owned by login roles of the same
  names. Underscored identifiers so nothing ever needs quoting in SQL.
- Connection target: `postgres-pooler.postgres.svc.cluster.local:5432`
  (PgBouncer, transaction mode). codepuke uses no LISTEN/NOTIFY, no advisory
  locks, and no session state, so the pooler is correct. goose and pgx work
  through it (`max_prepared_statements` is configured on the pooler).
- Declared in homelab as CNPG `managed.roles` entries plus `Database` CRs,
  the authentik pattern. CNPG reconciles each role's password from a
  `kubernetes.io/basic-auth` Secret in the `postgres` namespace, which ESO
  materializes from bao. The password therefore lives in exactly one place.

The dev role, database, and bao secret are created imperatively at stage 0 so
stage 1 can connect before any homelab change lands. This is safe against
GitOps: CNPG only reconciles roles it is told about and ArgoCD does not prune
resources it never tracked. When the homelab changes land, CNPG converges on
the same password (same bao secret) and adopts the existing database.

Local development connects with
`kubectl -n postgres port-forward svc/postgres-pooler 5432:5432` and the
credential from bao. Tests use testcontainers and never touch the cluster.

## Secrets

All secrets live in OpenBao under the homelab convention
`kv/<namespace>/<secret-name>`, one kv field per Secret key, and reach pods
only through External Secrets Operator (ClusterSecretStore `openbao`).

| bao path | fields | consumed by |
| --- | --- | --- |
| `kv/postgres/codepuke-db-credentials` | `username`, `password` | ESO Secret in `postgres` ns (CNPG role password) and in `codepuke` ns with a templated `uri` key |
| `kv/postgres/codepuke-dev-db-credentials` | `username`, `password` | same, for `codepuke-dev` |
| `kv/codepuke/codepuke-secrets` | `SESSION_KEY`, `OIDC_CLIENT_SECRET` | envFrom on the prod Deployment (stage 6) |
| `kv/codepuke-dev/codepuke-secrets` | same | dev Deployment |
| `kv/codepuke/ghcr-pull-secret` | `dockerconfigjson` (no leading dot) | dockerconfigjson-typed pull Secret (stage 5) |
| `kv/codepuke-dev/ghcr-pull-secret` | same | dev pull Secret |

The app-namespace DB Secret templates a composed
`uri: postgresql://<user>:<pass>@postgres-pooler.postgres.svc.cluster.local:5432/<db>`
key (authentik pattern), injected as `DATABASE_URL`. Secret values are seeded
with `bao kv put` from a generated value, never from a file on disk.

Database passwords must be URL-safe because the templated `uri` embeds them
without escaping: generate with `openssl rand -hex 24`, never base64 (its `/`
and `+` break URI parsing). The dev credential was rotated to hex for this
reason (bao secret version 2).

## Auth: Authentik application and group

Configured by hand in the Authentik UI per
`homelab/infrastructure/authentik/README.md`, at stage 6. Client IDs are
committed; only the client secret is secret.

- **Providers/Applications**: two, one per environment, each with its own
  client secret:
  - slug `codepuke`, redirect URI `https://codepuke.com/auth/callback`,
    issuer `https://authlayer.cloud/application/o/codepuke/`.
  - slug `codepuke-dev`, redirect URI
    `https://dev.codepuke.com/auth/callback`, issuer
    `https://authlayer.cloud/application/o/codepuke-dev/`.

  Both: OAuth2/OpenID, confidential, with PKCE (the app uses go-oidc +
  oauth2 with PKCE). Authorization flow
  `default-provider-authorization-implicit-consent`, signing key the authentik
  self-signed RS256 certificate, scopes `openid`, `profile`, and the custom
  verified-email mapping from the runbook. Client IDs `codepuke` and
  `codepuke-dev` are committed config; only the secrets are secret.
- **Group**: `codepuke-authors`, shared by both applications. Bound on each
  Application so only members can authenticate to codepuke at all (an unbound
  application is open to every authentik user). The `/admin` middleware
  additionally requires the group in the ID token's group claim, so
  authorization does not rest on the binding alone. Adding an author is a
  group membership change, no deploy.
- Client secrets seeded to `kv/codepuke/codepuke-secrets` and
  `kv/codepuke-dev/codepuke-secrets` as `OIDC_CLIENT_SECRET`, one distinct
  secret per environment.

## Images and delivery

- Image `ghcr.io/codepuke/codepuke`, private, built by GitHub Actions in this
  repo (copy potatoelf's `ci.yml` and `release.yml`): green `main` pushes
  `:dev` and `:sha-<full>` and bumps `deploy/overlays/dev/kustomization.yaml`,
  which rolls dev; a `v*` tag builds the release image for prod.
- ArgoCD already has org-wide read credentials for `codepuke/*` repos.
- Prod release is two commits: tag here, then bump `newTag` and `?ref=` in
  `homelab/workloads/codepuke/kustomization.yaml`.

## Required homelab changes

Items 1 through 3 landed with stage 5 (homelab commit "postgres, argocd,
homepage: add the codepuke dev preview"), along with the homepage entry and
the GitHub webhook. Item 4 landed with stage 6 (homelab commit "postgres,
argocd, homepage: add codepuke production"): role and database `codepuke`,
prod ExternalSecrets, `workloads/codepuke/` pinned at v0.1.0, and the
codepuke Application. Every homelab change in this list is done; the only
by-hand remainder is the Authentik application setup above.

1. `infrastructure/postgres/cluster.yaml`: add to `managed.roles`:

   ```yaml
   - name: codepuke_dev
     ensure: present
     login: true
     inherit: true
     connectionLimit: -1
     passwordSecret:
       name: codepuke-dev-db-credentials
   ```

2. `infrastructure/postgres/databases/codepuke-dev.yaml` (and register it in
   `infrastructure/postgres/kustomization.yaml`):

   ```yaml
   apiVersion: postgresql.cnpg.io/v1
   kind: Database
   metadata:
     name: codepuke-dev
     namespace: postgres
     annotations:
       argocd.argoproj.io/sync-options: SkipDryRunOnMissingResource=true
   spec:
     cluster:
       name: postgres
     name: codepuke_dev
     owner: codepuke_dev
     databaseReclaimPolicy: retain
   ```

3. `infrastructure/postgres/externalsecrets.yaml`: postgres-side
   ExternalSecret `codepuke-dev-db-credentials` from
   `postgres/codepuke-dev-db-credentials` (copy the authentik entry, type
   `kubernetes.io/basic-auth`). Plus `apps/workloads/codepuke-dev.yaml`
   sourcing `https://github.com/codepuke/codepuke` `main` at
   `deploy/overlays/dev`, registered in `apps/workloads/kustomization.yaml`.
   The dev overlay in this repo carries its own namespace and ExternalSecrets
   (potatoelf-dev pattern).

4. Prod equivalents of 1 through 3: role `codepuke`, database `codepuke`,
   secret `codepuke-db-credentials`, `workloads/codepuke/{kustomization,
   externalsecrets}.yaml`, `apps/workloads/codepuke.yaml`.

5. `infrastructure/homepage/configmap.yaml`: codepuke entry under `Sites`,
   and bump `homelab.danwolf.net/config-revision` in
   `infrastructure/homepage/deployment.yaml`.

6. GitHub webhook on `codepuke/codepuke` to
   `https://hooks.danwolf.net/argocd/api/webhook` so dev rolls out in seconds
   (commands in `homelab/workloads/CLAUDE.md`).

## Out of band, already done or not needed

- Cloudflare DNS: apex, www, and dev proxied CNAMEs to the tunnel exist.
- No cloudflared config change, no new tunnel routes, no cert-manager
  Certificate (edge TLS), no NetworkPolicy (none exist cluster-wide).

## Stage 0 state

Created imperatively against the live cluster (see "PostgreSQL" for why this
is GitOps-safe):

- bao: `kv/postgres/codepuke-dev-db-credentials` (`username`, `password`).
- Postgres: role `codepuke_dev` (login) and database `codepuke_dev` owned by
  it, on the shared CNPG cluster.

Prod resources intentionally deferred.
