# Sally Scaling Cost Report

Date: 2026-05-12  
Revised: 2026-05-11 (Wyatt + Claude review)

This report estimates Sally's monthly vendor costs from one early user through the first user milestone of 100 architect users. The goal is not a procurement quote; it is a working cost model that shows the main cost drivers, the break-even subscription floor, and where the estimate should be replaced with measured production data.

## Executive Summary

Sally's first 100 architect users can be hosted for well under $100/month in direct infrastructure costs. LLM spend is the dominant variable cost and the one most worth measuring early.

| Scenario | Monthly cost | Notes |
|---|---:|---|
| Lean (Fly.io + Neon + R2) | $20–$60 | Scale-to-zero app, serverless Postgres, free object storage egress |
| Standard (DO 2 GiB + managed Postgres) | $55–$110 | Simple, predictable, proven at this scale |
| Conservative (DO 4 GiB + managed Postgres + monitoring) | $120–$250 | Better headroom, includes buffer for LLM retries and bad extraction days |

At 100 users, direct vendor costs are not the problem. The constraints are extraction quality, support load, and proving that architects will pay $10–$20/month (or firms $49–$99/month). Price accordingly — not to cover servers, but to cover your time.

## Product And Workload Assumptions

Current Sally architecture:

- Chrome MV3 extension captures product-page content.
- Go server handles Mothership dashboard, auth, share links, uploads, admin views, and extraction API.
- Postgres stores users, projects, schedules, items, project members, share links, API tokens, and extraction logs.
- LLM providers are configurable: `stub`, `openai`, `ollama`, `chatcompletion`, `anthropic`.
- Extraction logs already capture `prompt_tokens` and `completion_tokens` — these become the source of truth for cost modeling after real usage.

Planning definition of "100 users":

- 100 architect users with accounts.
- 30–60 monthly active users.
- 1–5 contractor/share viewers per active architect.
- 10–50 SPEC extractions per active architect per month.
- **Base case**: 100 architects × 25 extractions = 2,500 LLM calls/month.
- **Heavy case**: 100 architects × 100 extractions = 10,000 LLM calls/month.

LLM token assumptions per extraction:

| Request type | Input tokens | Output tokens | Why |
|---|---:|---:|---|
| Typical product page | 8,000 | 800 | Visible text, structured data, schedule context, JSON response |
| Heavy page / PDF | 20,000 | 1,500 | Long product pages, extracted cut-sheet text, larger project context |

Replace these with measured values from `extraction_logs` once 20–50 real users are active.

## Cloud Platform Comparison

Codex's original draft defaulted to DigitalOcean without evaluating alternatives. Here is an honest comparison for a Go + Postgres SaaS run by one person.

### DigitalOcean

**Verdict: solid default, not a no-brainer.**

Pros: simple UI, good managed Postgres, predictable pricing, 2 TB included transfer, strong documentation, one-provider simplicity.  
Cons: more expensive per GiB than Hetzner, less developer-ergonomic deployment than Fly.io, managed Postgres minimum ($15/mo) is meaningful at Stage 0–1.

Good fit if: you want everything in one dashboard and you already know it.

### Fly.io

**Verdict: recommended for a Go app at this stage.**

Fly deploys Go from a Dockerfile with a single command (`fly deploy`). No VPC configuration, no IAM roles, no security group rules. You get global anycast routing, scale-to-zero shared VMs, and a managed Postgres option. The free tier covers a small persistent VM and a basic Postgres cluster.

Pricing: shared-cpu-1x VM is ~$1.94/month for always-on (or free in the free allowance). Managed Postgres starts at ~$5/month for a single node on shared hardware.

The developer experience gap between Fly and DigitalOcean is significant for a one-person team. `fly deploy` beats configuring droplets, nginx, systemd, and deploy scripts by hand.

Cons: Fly is newer and less battle-tested than DO. Their managed Postgres has had occasional instability. Keep a `pg_dump` offsite regardless.

### Hetzner

**Verdict: cheapest compute, worth it if EU data residency is acceptable.**

A Hetzner CX22 (2 vCPU, 4 GiB RAM) costs €4.49/month — roughly a third of the equivalent DO droplet. For an indie project where cost matters more than US data residency, Hetzner is hard to beat on compute. Their managed Postgres equivalent is via Hetzner Managed Server or a self-hosted Postgres on a small VPS.

Cons: EU-only servers (Frankfurt, Helsinki, Nuremberg, Ashburn, Singapore). No native managed Postgres product as polished as DO's. Smaller ecosystem of tutorials and integrations.

Worth considering at Stage 1+ if cost is tight.

### AWS

**You said you like it — honest take: wait.**

AWS is excellent but carries real operational overhead for a solo developer at 100 users. ECS, ECR, RDS, VPC, IAM roles, security groups, ALB — the setup that's correct for AWS at this scale takes days to get right and weeks to understand fully. The cost is comparable to DO or Fly at small scale, but the learning curve and ongoing ops burden are not.

**Exception**: if you already have AWS infra (existing account, familiarity with CDK or Terraform, existing RDS), it is not crazy. But don't start here.

Better AWS option if you go this route: **App Runner** (managed containers, no VPC required, autoscale to zero) + **RDS Aurora Serverless v2** (pay-per-use Postgres). Roughly $15–$40/month at low traffic. Simpler than ECS but still more complex than Fly.

### GCP Cloud Run

**Underrated for a containerized Go app.**

Cloud Run deploys containers, scales to zero, and charges per request-second. For a low-traffic SaaS this can be very cheap ($0–$10/month for the compute portion). Pair with Cloud SQL for Postgres (~$10/month for a shared-core instance).

The ops model is simpler than EC2/ECS: no VMs to manage, no SSH access needed. But GCP's console and IAM are confusing and the managed experience is less polished than Fly.

### Azure

Skip it unless you have an existing commitment. Not competitive for a new solo project.

### Render / Railway

Both are simpler than DO with good free tiers. Render is more production-stable; Railway is more developer-friendly. Either is fine for Stage 0–1. Both get more expensive than DO at higher usage. Not recommended as primary for 100 paid users — the pricing model penalizes growth.

### Recommended Stack

**Primary recommendation: Fly.io + Neon + Cloudflare R2**

| Component | Choice | Estimated monthly cost |
|---|---|---:|
| App server | Fly.io shared-cpu-1x | $0–$5 |
| Database | Neon serverless Postgres | $0–$19 |
| Object storage | Cloudflare R2 | $0–$5 |
| CDN / DNS | Cloudflare free | $0 |
| **Total infra (100 users)** | | **$5–$29** |

[Neon](https://neon.tech) is a serverless Postgres provider with a generous free tier (10 GB, autoscaling, branching). It scales to zero between requests. For 100 users with intermittent DB load, Neon free or Pro ($19/month) is likely cheaper than DO's $15/month minimum for a managed Postgres node that idles most of the time. Branch-based dev/staging environments are genuinely useful.

**Fallback recommendation: DigitalOcean**

If you want everything in one provider with a proven track record and you don't want to think about it:

| Component | Choice | Monthly cost |
|---|---|---:|
| App server | 2 GiB Droplet | $12 |
| Database | Managed Postgres single node | $15 |
| Object storage | Spaces (or Cloudflare R2) | $0–$5 |
| Backups | Droplet weekly backup | $2–$4 |
| **Total infra (100 users)** | | **$29–$36** |

## Current Public Pricing Inputs

Prices verified 2026-05-12. Re-check before any customer-facing forecast.

| Service | Pricing used | Source |
|---|---|---|
| DigitalOcean Droplet | 1 GiB: $6/mo, 2 GiB: $12/mo, 4 GiB: $24/mo | digitalocean.com/pricing |
| DO Managed Postgres | Single node: $15/mo; HA: $60+/mo | DO docs |
| Fly.io shared VM | Free allowance + ~$1.94/mo per shared-cpu-1x | fly.io/docs/about/pricing |
| Neon Postgres | Free tier (10 GB); Pro $19/mo (autoscaling) | neon.tech/pricing |
| Cloudflare R2 | 10 GB/mo free; $0.015/GB after; egress free | cloudflare.com/developer-platform/r2 |
| Cloudflare DNS/CDN | Free | cloudflare.com |
| OpenAI GPT-4o mini | ~$0.15/1M input; ~$0.60/1M output | openai.com/api/pricing |
| OpenAI GPT-4o | ~$2.50/1M input; ~$10.00/1M output | openai.com/api/pricing |
| Anthropic Claude Haiku 4.5 | ~$0.80/1M input; ~$4.00/1M output | anthropic.com/pricing |
| Anthropic Claude Sonnet 4.6 | ~$3.00/1M input; ~$15.00/1M output | anthropic.com/pricing |
| LemonSqueezy | 5% + $0.50 per transaction (MoR, handles tax globally) | lemonsqueezy.com/pricing |
| Stripe | 2.9% + $0.30 per transaction (you handle tax) | stripe.com/pricing |

**Note on LLM model names**: the original draft referenced "GPT-5.4 mini" — that model does not exist as of this writing. Prices above use current known models. By the time Sally has 100 paying users, newer/cheaper models will likely exist; the per-token cost trend has been consistently downward.

## LLM Cost Model

LLM spend is the main variable cost. Compute and database are mostly fixed at this milestone.

Approximate cost per extraction using **current** models:

| Model | Typical request | Heavy request |
|---|---:|---:|
| GPT-4o mini | $0.0016 | $0.0033 |
| Claude Haiku 4.5 | $0.0070 | $0.0145 |
| GPT-4o | $0.028 | $0.058 |
| Claude Sonnet 4.6 | $0.036 | $0.083 |

Monthly LLM spend at 100 users:

| Usage level | Calls/mo | GPT-4o mini | Claude Haiku | GPT-4o | Claude Sonnet |
|---|---:|---:|---:|---:|---:|
| Light | 1,000 | $2–$3 | $7–$15 | $28–$58 | $36–$83 |
| Base | 2,500 | $4–$8 | $18–$36 | $70–$145 | $90–$206 |
| Heavy | 10,000 | $16–$33 | $70–$145 | $280–$580 | $360–$825 |

GPT-4o mini is the obvious default. The quality gap between mini and full models is real for complex pages and PDFs, but for straightforward product pages it may be acceptable. The right answer is to run both on the same real extractions and measure field completeness.

Sally already has the provider abstraction and extraction logging to support model-per-extraction routing. A sensible strategy: default to the cheap model, escalate to a stronger model on retry when required fields are missing.

**Self-hosted Ollama**: Sally supports Ollama and you have a machine running it. At 100 users, routing non-critical or low-stakes extractions through a local Ollama instance (when it's running well) could meaningfully reduce LLM spend. Not reliable enough to plan around, but worth keeping as a cost lever.

## Infrastructure Cost By Stage

### Stage 0: One Internal User

| Cost item | Monthly estimate |
|---|---:|
| App server | $0–$5 |
| Postgres | $0–$19 |
| Object storage | $0 |
| LLM usage | $1–$10 |
| Domain / DNS | $0–$2 |
| **Total** | **$1–$36** |

Use `stub` or local Ollama for UI work to avoid burning paid LLM calls during development.

### Stage 1: 10 Real Users

| Cost item | Monthly estimate |
|---|---:|
| App server | $2–$12 |
| Managed Postgres (Neon or DO) | $0–$19 |
| Object storage | $0–$5 |
| Backups | $1–$4 |
| LLM usage | $5–$30 |
| Uptime monitoring | $0–$5 |
| Email / admin tools | $0–$10 |
| **Total** | **$8–$85** |

Start recording actual extractions per user, tokens per extraction, error rate, and support time. These numbers matter more than any estimate.

### Stage 2: 100 User Milestone

**Lean (Fly.io + Neon):**

| Cost item | Monthly estimate |
|---|---:|
| App server (Fly.io) | $0–$5 |
| Postgres (Neon Pro) | $0–$19 |
| Object storage (Cloudflare R2) | $0–$5 |
| Uptime / error monitoring | $0–$10 |
| LLM usage, GPT-4o mini default | $4–$33 |
| Email / admin tools | $0–$10 |
| **Total** | **$4–$82** |

**Standard (DigitalOcean):**

| Cost item | Monthly estimate |
|---|---:|
| App VM: 2 GiB Droplet | $12 |
| Managed Postgres single node | $15 |
| Droplet backup (weekly) | $2–$4 |
| Object storage | $0–$5 |
| Uptime / error monitoring | $0–$10 |
| LLM usage, GPT-4o mini default | $4–$33 |
| Email / admin tools | $0–$10 |
| **Total** | **$33–$89** |

**Conservative (more headroom):**

| Cost item | Monthly estimate |
|---|---:|
| App VM: 4 GiB Droplet | $24 |
| Managed Postgres | $15–$30 |
| Backups | $5–$15 |
| Object storage | $0–$5 |
| Monitoring / error tracking | $5–$30 |
| LLM usage (mini + stronger retries) | $20–$80 |
| Email / admin tools | $10–$25 |
| Contingency | $20–$50 |
| **Total** | **$99–$259** |

Use $100–$200/month as the planning envelope for the first 100 users when talking about pricing. The lean stack can do it for much less, but buffer matters.

## Network Traffic And Storage

Network traffic is not a meaningful cost at 100 users:

- DigitalOcean's 2 GiB Droplet includes 2 TB/month transfer. Sally's traffic (HTML, small JSON, thumbnails) won't touch it.
- Cloudflare R2 has free egress. DigitalOcean Spaces is $5/month if you prefer one-provider simplicity.
- Fly.io includes 100 GB egress free per month.

Watch-outs:

- Proxying or caching full-size product images will grow storage and bandwidth noticeably.
- Server-side PDF storage: measure this separately as Sally's PDF extraction evolves.
- Public share links becoming contractor-facing catalogs could create unexpected anonymous read traffic at scale.

## BC/DR And Availability

| Level | Monthly cost | Recovery posture | Fit |
|---|---:|---|---|
| Basic backups | $5–$15 | Weekly VM backup, managed DB backups, nightly `pg_dump` to R2/Spaces, manual restore | Best first production default |
| Tested restore | $15–$50 | Basic backups + monthly restore rehearsal to temporary VM/DB | Recommended before paid pilots |
| HA architecture | $80–$150+ | Two app nodes, load balancer, HA Postgres, documented failover | Premature until a paying customer requires an uptime SLA |

Do not buy HA before 100 users. Write and test a restore runbook instead:

1. Restore latest `pg_dump` from object storage to a fresh Postgres instance.
2. Deploy the current Go binary or container image.
3. Restore uploads from object storage if needed.
4. Verify login, project list, extraction endpoint, and share links.
5. Record actual recovery time.

Target for the first paid milestone: RPO < 24 hours, RTO < 4 hours. This is achievable for $5–$15/month and is probably fine for early customers.

## Payment Processing

**Recommendation: LemonSqueezy (or Paddle) over raw Stripe.**

Stripe is excellent but it makes *you* the merchant of record. That means you are responsible for collecting and remitting sales tax in every jurisdiction where you have "economic nexus" — which in the US means you can owe sales tax in states where you've never set foot, triggered by revenue thresholds. Architects are distributed across all 50 states. This is a real compliance burden for a one-person business.

LemonSqueezy and Paddle are Merchants of Record: they collect the money, handle all tax compliance globally, and remit to you net of their fee. The fee (LemonSqueezy: 5% + $0.50/transaction) is higher than Stripe per transaction, but the tax compliance alone is worth it at this stage. You can switch to Stripe + a tax automation layer (Stripe Tax, TaxJar) later when the revenue justifies it.

At $15/user/month with 100 users ($1,500 MRR):
- **LemonSqueezy fee**: ~$125/month
- **Stripe fee**: ~$74/month + your time managing sales tax compliance

The $51/month difference buys meaningful peace of mind early. Revisit at 500+ users.

## Other Overhead

| Category | Cost behavior | Notes |
|---|---|---|
| Support | Step function | Even 50 active users can generate hours of support. This will likely exceed cloud cost at 100 users. |
| Email | Mostly fixed | Transactional email (magic links, notifications) stays cheap via Resend or Postmark free tiers. Google Workspace is not necessary early — Gmail + a custom domain alias works fine. |
| Observability | $0–$30/mo | Netdata is already self-hosted. Add hosted error tracking (Sentry free tier) only when it saves debugging time faster than logs. |
| Security/compliance | Mostly labor | OAuth, backups, least-privilege API keys, audit logs, and data deletion are more important than paid tooling at this stage. |
| Chrome Web Store | One-time | $5 developer registration. Already paid. |
| Taxes / accounting / legal | Fixed/step | Not in vendor-cost break-even; include in business pricing. An accountant for a small SaaS runs $500–$2,000/year. |

## Break-Even User Pricing

LemonSqueezy net revenue formula:

```
net per user = (price × 0.95) − 0.50
break-even price = (monthly vendor cost / user count + 0.50) / 0.95
```

At 100 architect users:

| Monthly vendor cost | Vendor cost/user | LemonSqueezy break-even |
|---:|---:|---:|
| $50 | $0.50 | $1.05/user/mo |
| $100 | $1.00 | $1.58/user/mo |
| $200 | $2.00 | $2.63/user/mo |

Infrastructure break-even is trivially low. The real question is what number makes the business worth running.

Example plan economics at 100 users:

| Price | Gross MRR | LemonSqueezy fees (~5%) | Net before vendor cost | Net after $100 infra | Net after $200 infra |
|---:|---:|---:|---:|---:|---:|
| $10/user/mo | $1,000 | $75 | $925 | $825 | $725 |
| $15/user/mo | $1,500 | $100 | $1,400 | $1,300 | $1,200 |
| $25/user/mo | $2,500 | $150 | $2,350 | $2,250 | $2,150 |
| $49/firm/mo | varies | — | — | — | — |

**Pricing guardrails:**

- Do not charge less than $10/architect/month if pricing per seat. That is the floor where vendor costs become meaningless noise.
- A per-firm plan ($49–$99/month for an office of 1–5 architects) may convert better than per-seat for small studios. Architects work in small teams; individual seat pricing feels like enterprise friction.
- Include an explicit SPEC extraction allowance and be transparent about it. Users who understand usage limits self-regulate; those who don't will find any throttle feels arbitrary.
- Treat contractor/share viewers as free permanently (or until viewer traffic meaningfully affects costs). Charging for viewers kills the "send a link to your contractor" value proposition.
- Consider a one-time project fee as a future option: pay $X to create a project, no monthly commitment. Architects often work project-to-project.

## Key Risks

- **LLM quality**: the cheap model may require frequent retries or produce too many missing fields, pushing actual LLM spend 2–5× above the base estimate.
- **PDF extraction token explosion**: a 50-page cut-sheet can use 10× the tokens of a typical product page.
- **Retry loops**: failed extractions may be retried several times, paying LLM cost without producing value. Cap retries; surface failure clearly.
- **Share-link traffic**: public catalogs could create unexpected anonymous read volume.
- **Single-node database**: planned downtime for Postgres upgrades, or unplanned node failure, creates outage. Acceptable at this stage with a tested restore runbook.
- **Support load**: manual support from even 30 active users can dwarf all infrastructure costs.
- **Vendor pricing changes**: LLM API pricing has been dropping, but capacity and availability can change. Re-check before launch.

## Metrics To Instrument

Sally's admin dashboard already tracks extraction stats. Add or verify:

- Extractions per user per day/week/month.
- Prompt and completion tokens per extraction (already in `extraction_logs`).
- Derived LLM cost per extraction (provider + model + token counts).
- Retry rate and failure rate by provider/model.
- Missing-fields rate by provider/model (most important quality signal).
- Storage used by uploads.
- Share-link page views by project (watch for viral/high-traffic shares).
- Active architect users vs. read-only contractor viewers.

## What I'd Actually Do

**Right now (Stage 0–1):**

1. Deploy on Fly.io. `fly launch` from the repo root, add a `fly.toml`, done. Takes an afternoon.
2. Use Neon free tier for Postgres. Set up nightly `pg_dump` to Cloudflare R2 (free tier).
3. Use GPT-4o mini as the default extraction model. It's 10–20× cheaper than Sonnet for comparable product-page quality.
4. Keep Cloudflare for DNS. It already handles the domain.
5. Set up Sentry free tier for error tracking.
6. Budget: ~$5–$20/month for infrastructure. Everything else is LLM spend driven by actual usage.

**Before first paid user:**

1. Test the restore runbook (Postgres restore + redeploy) and record the time. Fix whatever is slow.
2. Set up LemonSqueezy with a $15/architect/month or $49/firm/month plan.
3. Instrument LLM cost per extraction in the admin dashboard.
4. Decide on extraction soft-cap (e.g., warn at 200 extractions/month, hard-cap at 500 for base plan).

**At 50–100 paying users:**

1. Evaluate whether Fly.io shared VMs are hitting limits. If yes, upgrade to dedicated; still cheaper than a DO 4 GiB droplet.
2. Evaluate Neon Pro ($19/month) if DB is growing.
3. Consider adding a retry escalation path (GPT-4o mini → GPT-4o on field-missing failures).
4. Revisit payment processor. If MRR is $5,000+, Stripe + Stripe Tax starts to make economic sense vs. LemonSqueezy's percentage fee.

The entire stack for the first 100 users should cost less than a dinner out. The business risk is not the server bill.

---

## Founder Compensation and Funding

*This section is not legal or financial advice. Before formalizing any equity arrangement or taking outside money, consult a business attorney. The cost is worth it.*

### The Situation

Two founders:

- **The architect**: came up with the idea, owns the domain expertise, brought business knowledge, user requirements, and professional constraints. Likely has a warm network of potential customers — other architects and interior designers who face the same spec-entry problem.
- **The developer (Wyatt)**: built the product, 50+ hours in, $100+ in personal LLM spend, and would continue building features, fixing bugs, and providing support.

Neither of these contributions is optional. An idea without execution is nothing. Execution without domain knowledge builds the wrong thing. This is a genuine co-founding relationship and should be treated as one.

### Formalize It Before There Is Money

The easiest time to agree on a split is before revenue exists. Once money is on the table, every conversation gets harder. The single most important step is to write down — even informally to start — what each person contributes and what they expect in return.

**Minimum viable agreement (before an LLC):**

A written email or shared doc stating:
- Each party's equity percentage or revenue share.
- What each person is contributing (time, IP, expenses).
- What happens if one person stops contributing.
- That the understanding is subject to a proper operating agreement once an entity is formed.

This is not legally binding on its own, but it establishes intent and creates a paper trail. Disputes almost always come down to "I thought we agreed..." — this prevents that.

### Entity Structure

**Recommended: LLC (Limited Liability Company)**

For a two-person bootstrapped SaaS targeting small architecture firms:

- An LLC protects both founders' personal assets from business liabilities.
- Pass-through taxation (profits are taxed once, on your personal returns) — no double taxation like a C-corp.
- Straightforward to form ($50–$200 in most states via a registered agent service).
- An operating agreement defines the equity split, voting rights, profit distribution, and what happens if a founder exits.

**When to consider a C-corp instead**: if you plan to raise venture capital, a Delaware C-corp is standard. VCs typically won't invest in LLCs. For a bootstrapped architecture-niche SaaS targeting 100–500 users, a C-corp adds complexity with no benefit. Convert later if needed.

Formation cost: ~$500–$1,500 if using a lawyer; ~$100–$300 DIY via ZenBusiness or Stripe Atlas. Get a lawyer to review the operating agreement — this is where mistakes get made.

### Equity Split: Starting Point for Discussion

There is no universally correct answer, but here is a framework:

**The Slicing Pie / Grunt Fund approach** (recommended for the current stage):

Rather than negotiating a fixed split upfront, track contributions dynamically at a fair market rate and convert to a fixed split once the business has enough evidence to set terms fairly. This is described in Mike Moyer's *Slicing Pie* and is well-suited to a pre-revenue partnership where both founders are contributing non-cash value.

Each hour contributed is valued at a market rate:
- Developer time: $100–$150/hour (conservative for a senior full-stack dev who owns infrastructure, architecture, and support)
- Domain expert / advisor time: $150–$250/hour (experienced licensed architect's consulting rate)
- Cash invested (out-of-pocket expenses like LLM tokens): valued at 2× cash (to account for the risk premium)

The running total ratio becomes the equity split when you formalize.

**Rough estimate based on current contributions:**

| Contribution | Est. value |
|---|---:|
| Developer: ~50 hours @ $125/hr | $6,250 |
| Developer: $100+ LLM/expenses (2× multiplier) | $200+ |
| Architect: ideation, requirements, domain consulting (est. 20–30 hrs @ $200/hr) | $4,000–$6,000 |

This puts the current split somewhere around **50/50 to 55/45 developer-heavy** based on hours in — but the architect's network and future business development contribution can easily balance or reverse this. Agree on the formula, not just the current number.

**If you prefer a simpler fixed split**: 50/50 with equal vesting is the most common and often the most durable. It removes the perception of scorekeeping and signals equal partnership. The risk is that unequal future effort feels unfair; address this with vesting and a clear agreement on what happens if someone steps back.

### Vesting

Even between two people who trust each other completely, vesting protects both parties.

**Recommended: 24-month vesting, 6-month cliff**

- Nothing vests in the first 6 months.
- At the 6-month mark, 25% vests immediately.
- The remaining 75% vests monthly over the following 18 months.

This means if either founder exits early (life happens — the architect gets too busy, the developer finds another job), the equity they take reflects actual contribution rather than a paper promise.

### Profit Distribution

For an LLC, profits are distributed to members according to the operating agreement. A simple approach:

1. Cover operating expenses first (infrastructure, payment processor fees, etc.).
2. Maintain a small cash reserve (1–2 months of operating costs).
3. Distribute the remainder to members in proportion to equity, on a schedule (monthly, quarterly, etc.) you agree on.

At 100 users paying $15/month:
- Gross MRR: $1,500
- LemonSqueezy fees: ~$125
- Infrastructure: ~$100–$200
- **Net available to distribute: ~$1,175–$1,275/month**

At a 50/50 split, that is roughly $600/month each. Not a salary, but it covers the server bill and compensates for ongoing time. It becomes meaningful faster than most people expect if usage grows.

### Compensating Past Work

The 50+ hours and $100+ in out-of-pocket expenses invested so far are real. Options for acknowledging them:

1. **Count them in the equity formula** (recommended for now): treat them as founder contributions that set the initial ratio, rather than a debt the business owes.
2. **Reimburse expenses from first revenue**: the $100+ in LLM tokens is a legitimate business expense. Reimburse it as soon as the business has revenue before taking distributions.
3. **Back-pay as a note**: the business "owes" Wyatt for pre-formation development work, payable from future revenue with a reasonable interest rate. This is real but adds complexity; generally not worth formalizing unless the imbalance is large.

### Where the Money Comes From

In roughly increasing order of complexity and dilution:

**1. Bootstrap from personal funds (recommended)**

Infrastructure costs are $20–$200/month. Both founders can cover this personally while the product is pre-revenue. Agree upfront on who covers what and track it — these are either equity contributions (if valued as founder cash) or operating expenses to be reimbursed from first revenue.

**2. Early revenue from the architect's network**

The architect's professional network is the most underrated asset in this whole picture. If 5–10 architecture colleagues will pay $15–$50/month to try a tool that a trusted peer built and uses themselves, that is the cleanest possible funding: real revenue from the target market, no dilution, and direct feedback from professional users.

This should be attempted before anything else. A single email from the architect to their network along the lines of "I've been building this tool with a developer friend and it's saved me hours — I'd like to invite you to try it for $X/month" will outperform any other early marketing.

**3. Pre-sales / founding member pricing**

Offer a "founding member" rate ($99–$199 one-time, or $9/month locked forever) to the first 20–50 users. This:
- Funds a few months of infrastructure and development time.
- Creates an invested early cohort who will give feedback and refer others.
- Validates willingness to pay before significant additional investment.

**4. Small friends-and-family investment**

If there are personal contacts willing to invest $5,000–$25,000 for a stake in the LLC, this covers several months of development time and infrastructure without the overhead of a formal funding round. This is usually structured as a SAFE (Simple Agreement for Future Equity) or a small convertible note. Keep it simple and keep the cap table small.

**5. Small business grants and programs**

Not the most likely path, but worth knowing:
- **SBIR/STTR grants**: Federal grants for small businesses doing R&D. Architecture and construction tech can qualify. Typical Phase I awards are $50,000–$300,000. High effort to apply, but non-dilutive.
- **State-level small business grants**: Many states have grants for software startups. Low amounts ($5,000–$25,000) but free money.
- **SCORE mentorship**: Free mentorship from retired executives. Useful for the business side; the architect's professional network may be more valuable.

**6. Angel investment**

Once there are 20–50 paying users and a clear growth path, a single angel investor (often an architect, contractor, or construction-tech professional) might invest $25,000–$100,000 for a minority stake. This funds a year or more of part-time development. The architect's professional network is the place to find this investor.

**7. Accelerators (Y Combinator, Techstars, etc.)**

Premature until there is meaningful revenue and growth. YC in particular looks for strong traction before accepting construction/AEC niche SaaS tools. Not off the table eventually, but not the right focus now.

**8. Venture capital**

Not a fit for this product at this stage, and possibly not ever. VC requires a path to $100M+ ARR and a large exit. Architecture-niche SaaS targeting small firms is a lifestyle business unless the market turns out to be much larger than the 100-user milestone suggests. That is fine — a profitable, small, cash-flowing business that pays its founders well is a good outcome.

### The Honest Short Version

The two questions to answer this week:

1. **What is the equity split and how is it calculated?** Decide this now in writing, even if informally. 50/50 with vesting is a reasonable default if ongoing contributions are expected to be roughly equal.

2. **Who pays for the server?** The infrastructure costs are low enough that the question of "funding" is really just "who covers $20–$200/month while we find paying users." Whoever has cash flow covers it; track it as a reimbursable expense or equity contribution.

The real funding event is the first 10 paying users. That is when you have revenue, a model, and something worth structuring properly. Everything before that is founder time and cheap server bills.
