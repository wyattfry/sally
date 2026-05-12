# Sally Scaling Cost Report

Date: 2026-05-12

This report estimates Sally's monthly vendor costs from one early user through the first user milestone of 100 architect users. The goal is not a procurement quote; it is a working cost model that shows the main cost drivers, the break-even subscription floor, and the places where the estimate should be replaced with measured production data.

## Executive Summary

Sally's first 100 architect users should be inexpensive to host if the product remains a Go web app, Postgres database, Chrome extension, and hosted LLM extraction API.

The expected 100-user monthly vendor cost is:

| Scenario | Monthly cost | Notes |
|---|---:|---|
| Lean production | $55-$95 | Single app VM, single-node managed Postgres, free/low-cost object storage, GPT-5.4 mini-class extraction |
| Conservative production | $115-$245 | Larger app VM, managed Postgres, object storage, monitoring, heavier LLM usage, extra buffer |
| Higher-availability setup | $180-$350 | Two app nodes, load balancer, high-availability Postgres, stronger backup/restore posture |

At 100 users, Sally can cover direct vendor costs at roughly $2-$4 per architect user per month depending on LLM volume. That is only the infrastructure break-even point. A real paid plan should start closer to $10-$20 per architect user per month, or a higher per-firm plan, because support, product development, refunds, taxes, sales time, and usage spikes will dominate the vendor bill.

Contractor/share-link viewers should remain free unless they become a meaningful traffic source. Their read-only traffic is much cheaper than architect extraction work.

## Product And Workload Assumptions

Current Sally architecture:

- Chrome MV3 extension captures product-page content.
- Go server handles Mothership dashboard, auth, share links, uploads, admin views, and extraction API.
- Postgres stores users, projects, schedules, items, project members, share links, API tokens, and extraction logs.
- LLM providers are configurable. Current code supports `stub`, `openai`, `ollama`, `chatcompletion`, and `anthropic`.
- Extraction logs already include `prompt_tokens` and `completion_tokens`, which should become the source of truth after real usage exists.

Planning definition of "100 users":

- 100 architect users with accounts.
- 30-60 monthly active users.
- 1-5 contractor/share viewers per active architect.
- 10-50 SPEC extractions per active architect per month.
- Base case: 100 architects x 25 SPEC extractions each = 2,500 LLM calls per month.
- Heavy case: 100 architects x 100 SPEC extractions each = 10,000 LLM calls per month.

LLM token assumptions per SPEC extraction:

| Request type | Input tokens | Output tokens | Why |
|---|---:|---:|---|
| Typical product page | 8,000 | 800 | Visible text, structured data, schedule context, JSON response |
| Heavy product page/PDF | 20,000 | 1,500 | Long product pages, extracted cut-sheet text, larger project context |

These should be replaced with measured values from `extraction_logs.prompt_tokens` and `extraction_logs.completion_tokens` once 20-50 real users are active.

## Current Public Pricing Inputs

Prices checked on 2026-05-12. Re-check before a launch or investor/customer-facing forecast.

| Service | Pricing input used | Source |
|---|---:|---|
| DigitalOcean Basic Droplet | 1 GiB: $6/mo, 2 GiB: $12/mo, 4 GiB: $24/mo | <https://www.digitalocean.com/pricing/droplets> |
| DigitalOcean Droplet backup | Weekly backup: 20% of Droplet cost; daily backup: 30% | <https://www.digitalocean.com/pricing/droplets> |
| DigitalOcean Managed Postgres | Single node starts at $15/mo; HA starts at $30 primary plus at least one $30 standby; extra storage $0.21/GiB-mo | <https://docs.digitalocean.com/products/databases/postgresql/details/pricing/> |
| Cloudflare Tunnel | Free | <https://www.cloudflare.com/products/tunnel/> |
| Cloudflare R2 | 10 GB-month free; then standard storage $0.015/GB-month; egress free | <https://developers.cloudflare.com/r2/pricing/> |
| DigitalOcean Spaces alternative | $5/mo includes 250 GiB storage and 1 TiB outbound transfer | <https://docs.digitalocean.com/products/spaces/details/pricing/> |
| DigitalOcean Uptime | 1 uptime check free; $1/check/mo after | <https://www.digitalocean.com/pricing/uptime-monitoring> |
| OpenAI GPT-5.4 mini | $0.75 / 1M input tokens; $4.50 / 1M output tokens | <https://openai.com/api/pricing/> |
| OpenAI GPT-5.4 | $2.50 / 1M input tokens; $15.00 / 1M output tokens | <https://openai.com/api/pricing/> |
| Anthropic Claude Sonnet class | Public Sonnet 4.5/4.6 reporting: about $3 / 1M input tokens; $15 / 1M output tokens | <https://www.anthropic.com/claude/sonnet> |
| Stripe card payments | 2.9% + $0.30 per successful domestic card transaction | <https://stripe.com/us/pricing> |
| Google Workspace Business Starter | $7/user/mo on annual plan | <https://workspace.google.com/intl/en/pricing/> |

## LLM Cost Model

LLM spend is the main variable cost. Compute and database costs are mostly fixed at this milestone.

Approximate cost per extraction:

| Model class | Typical request | Heavy request |
|---|---:|---:|
| GPT-5.4 mini | $0.010 | $0.022 |
| GPT-5.4 | $0.032 | $0.073 |
| Claude Sonnet class | $0.036 | $0.083 |

Monthly LLM spend at 100 users:

| Usage level | Calls/mo | GPT-5.4 mini | GPT-5.4 | Claude Sonnet class |
|---|---:|---:|---:|---:|
| Light | 1,000 | $10-$22 | $32-$73 | $36-$83 |
| Base | 2,500 | $25-$55 | $80-$181 | $90-$206 |
| Heavy | 10,000 | $100-$220 | $320-$725 | $360-$825 |

Recommendation: default to the cheapest model that produces acceptable extraction quality, track per-provider success rate and missing fields, and reserve expensive models for retry/escalation paths. Sally already has the provider abstraction and extraction logging needed to support this.

## Infrastructure Cost By Stage

### Stage 0: One Internal User

This stage can stay on the current dev host or a single small VM.

| Cost item | Monthly estimate |
|---|---:|
| App + web server | $0-$6 |
| Postgres | $0-$15 |
| Object/file storage | $0 |
| Backups/snapshots | $0-$2 |
| LLM usage | $1-$10 |
| Domain/DNS/tunnel | $0-$2 |
| Total | $1-$35 |

Notes:

- A local or existing VM is fine here.
- Use `stub` or local `ollama` for UI work to avoid burning paid LLM calls.
- Do not draw pricing conclusions from this stage; the LLM quality loop matters more than infrastructure cost.

### Stage 1: 10 Real Users

This stage should be a real production-like environment, even if it is not highly available.

| Cost item | Monthly estimate |
|---|---:|
| App VM: 1-2 GiB Droplet | $6-$12 |
| Managed Postgres single node | $15 |
| Droplet backup | $1-$4 |
| Object storage | $0-$5 |
| Uptime monitoring | $0-$1 |
| LLM usage | $5-$30 |
| Domain/email/admin tools | $2-$15 |
| Total | $29-$82 |

Notes:

- A single-node database is acceptable if expectations are clear and restore procedures are tested.
- Start recording actual extractions per user, tokens per extraction, error rate, and support issues.
- Keep free contractor/share viewing; it should not materially affect cost at this stage.

### Stage 2: 100 User Milestone

Recommended lean setup:

| Cost item | Monthly estimate |
|---|---:|
| App VM: 2 GiB Droplet | $12 |
| Managed Postgres single node | $15 |
| Droplet backup | $2-$4 |
| Object storage | $0-$5 |
| Uptime monitoring | $0-$5 |
| LLM usage, GPT-5.4 mini-class | $25-$55 |
| Logs/monitoring/tools buffer | $5-$15 |
| Domain/email/admin tools | $2-$15 |
| Total | $61-$126 |

Recommended conservative setup:

| Cost item | Monthly estimate |
|---|---:|
| App VM: 4 GiB Droplet | $24 |
| Managed Postgres single node | $15-$30 |
| Backups/snapshots/object backups | $5-$15 |
| Object storage | $0-$5 |
| Uptime/error monitoring | $5-$30 |
| LLM usage, mixed mini + stronger retries | $50-$140 |
| Domain/email/admin tools | $10-$25 |
| Contingency | $20-$50 |
| Total | $129-$319 |

The lean setup is probably enough for 100 users if usage is bursty and normal architectural-product pages are handled well. The conservative setup is a better planning number when discussing pricing, because it includes vendor drift and bad extraction days.

## Network Traffic And Storage

Network traffic should not be a meaningful cost at 100 users:

- DigitalOcean's 2 GiB Droplet includes 2,000 GiB/month transfer.
- Sally's dashboard and share pages are mostly HTML/CSS, thumbnails, and small JSON/API requests.
- Product images are usually referenced by URL, not rehosted, unless users upload thumbnails.
- Cloudflare R2 has free egress and a 10 GB-month free tier; DigitalOcean Spaces is $5/mo if the project prefers one-provider simplicity.

Watch-outs:

- If Sally starts proxying or caching full-size product images, bandwidth and storage can grow.
- If server-side PDF extraction stores original PDFs, storage growth should be measured separately.
- If public share links become customer-facing catalogs with heavy contractor traffic, CDN/object storage architecture will matter more.

## BC/DR And Availability

For 1-100 users, there are three realistic levels:

| Level | Monthly cost | Recovery posture | Fit |
|---|---:|---|---|
| Basic backups | $5-$15 incremental | Weekly VM backup, managed DB backups, nightly `pg_dump` to object storage, manual restore | Best first production default |
| Tested restore | $15-$50 incremental | Basic backups plus monthly restore rehearsal to a temporary VM/database | Recommended before paid pilots |
| HA architecture | $80-$150+ incremental | Two app nodes, load balancer, HA Postgres, documented failover | Premature unless paid users require uptime SLA |

Recommendation: do not buy full HA before the first 100 users unless a paying pilot requires it. Instead, write and test a restore runbook:

1. Restore Postgres backup or latest `pg_dump`.
2. Deploy the current Go server binary/container.
3. Restore uploads/object storage if needed.
4. Verify login, project list, extraction endpoint, and share links.
5. Record actual recovery time.

The target for the first paid milestone can be RPO under 24 hours and RTO under 4 hours. That is much cheaper than true high availability and probably matches customer expectations for an early tool.

## Other Overhead And Variable Costs

These are not always cloud spend, but they affect what users must be charged.

| Category | Cost behavior | Notes |
|---|---|---|
| Payment processing | Variable | Stripe's domestic card rate means low monthly prices lose a noticeable percentage to the fixed $0.30 fee. |
| Support | Step function | Even 100 users can create hours of support work. This is likely bigger than cloud cost. |
| Email | Mostly fixed | Google Workspace at one admin seat is around $7/mo. Transactional email may stay free/cheap at this scale. |
| Observability | Free to $30+/mo | Netdata is self-hosted now. Add hosted error tracking only when it saves debugging time. |
| Security/compliance | Mostly labor | OAuth, backups, least-privilege API keys, audit logs, and data deletion are more important than paid tooling at this stage. |
| Chrome Web Store | One-time | Chrome Web Store developer registration is a one-time fee, not meaningful monthly spend. |
| Taxes/accounting/legal | Fixed/step | Not included in vendor-cost break-even; include in business pricing. |

## Break-Even User Pricing

Stripe net revenue formula for a monthly per-user subscription:

```text
net per user = (price * 0.971) - 0.30
break-even price = (monthly vendor cost / user count + 0.30) / 0.971
```

At 100 architect users:

| Monthly vendor cost | Vendor cost/user | Stripe-adjusted break-even price |
|---:|---:|---:|
| $75 | $0.75 | $1.08/user/mo |
| $150 | $1.50 | $1.85/user/mo |
| $250 | $2.50 | $2.88/user/mo |
| $350 | $3.50 | $3.91/user/mo |

That table only covers direct vendor costs. It does not fund support, product development, sales, admin time, tax overhead, refunds, or profit.

Recommended early pricing guardrails:

- Do not charge less than $10/architect/month if pricing per seat.
- Prefer a firm/workspace plan over tiny per-seat pricing, for example $49-$99/month for a small architecture office.
- Include a generous but explicit SPEC extraction allowance.
- Treat contractor/share viewers as free until their traffic materially changes costs.
- Consider usage guardrails: soft warnings at high extraction volume, then either throttle, require upgrade, or route to cheaper models.

Example plan economics at 100 users:

| Price | Gross MRR | Approx Stripe fees | Net before vendor cost | Net after $150 vendor cost | Net after $300 vendor cost |
|---:|---:|---:|---:|---:|---:|
| $5/user/mo | $500 | $45 | $455 | $305 | $155 |
| $10/user/mo | $1,000 | $59 | $941 | $791 | $641 |
| $15/user/mo | $1,500 | $74 | $1,426 | $1,276 | $1,126 |
| $25/user/mo | $2,500 | $103 | $2,397 | $2,247 | $2,097 |

Recommendation: for early pilots, use pricing that is high enough to validate willingness to pay, not merely enough to cover infrastructure. A $10-$20/architect/month plan or $49-$99/firm/month plan easily covers 100-user vendor costs while leaving room for support.

## Key Risks

- LLM quality may require stronger, more expensive models than the base estimate.
- Long PDF/cut-sheet extraction can multiply token usage.
- Users may retry failed extractions several times, raising LLM cost without creating value.
- Public share links could create unexpected anonymous traffic.
- Manual support may dominate all vendor costs.
- Single-node database architecture creates downtime risk even if it keeps costs low.
- Vendor pricing can change; re-check pricing monthly during pilot planning.

## Metrics To Add To The Cost Dashboard

Sally's admin dashboard already tracks extraction stats. For pricing decisions, add or verify these metrics:

- Extractions per user per day/week/month.
- Prompt tokens and completion tokens per extraction.
- LLM cost per extraction, derived from provider/model and token counts.
- Retry rate and failure rate by provider/model.
- Missing-fields count by provider/model.
- Storage used by uploads and extracted documents.
- Share-link page views by project.
- Active architect users versus read-only contractor viewers.
- Gross margin per account or project.

## Practical Recommendation

Run the first paid 100-user milestone on a lean but production-like stack:

- 2 GiB app VM.
- Single-node managed Postgres.
- Nightly database export to object storage.
- Cloudflare DNS/Tunnel or normal HTTPS ingress.
- Free/cheap uptime monitoring plus existing Netdata.
- GPT-5.4 mini-class model as default extraction path.
- Stronger model only for retry/escalation.

Use $150-$300/month as the planning envelope for the first 100 users. If 100 users are paying, vendor cost is not the constraint. The constraint is extraction quality, support load, and proving that architects get enough value to pay at least $10-$20 per month or that firms will pay $49-$99 per month.
