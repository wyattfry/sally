# Updating Project Hours

Use this document when Wyatt asks to update project hours, work logs, or `docs/git-work-log.csv`.

## Output

Update:

```text
docs/git-work-log.csv
```

The CSV must keep this header:

```csv
date,hours worked,tasks
```

## Source Of Truth

Use git history for this repo. Count Wyatt-authored commits and exclude automation:

- Exclude commits authored by `GitHub Actions`
- Exclude `Version update [skip actions]` commits
- Ignore merge commits in the task summary text, but allow their timestamps to contribute to the activity window when they are Wyatt-authored

Use local commit dates. The prior CSV was generated with:

```bash
git log --date=iso-strict-local --pretty=format:'%H%x09%an%x09%ae%x09%ad%x09%s' --all --reverse
```

## Hour Estimate

Approximate active work time from commit timestamps:

1. Group Wyatt-authored commits by local date.
2. Within each date, split sessions when the gap between adjacent commits is more than 2 hours.
3. For each session, estimate `session duration + 0.5 hours`.
4. Use a minimum of `0.5 hours` per session.
5. Sum sessions per day and round to 1 decimal place.

This is intentionally an approximation of active project work from git activity, not a timesheet.

## Task Summary

For each day, summarize the non-automation, non-merge commit subjects into a concise sentence. Prefer product-oriented wording over listing every commit when a day has many commits.

## Helpful Script

This prints the raw daily inputs and estimated hours:

```bash
python - <<'PY'
import subprocess
from datetime import datetime, timedelta
from collections import defaultdict

raw = subprocess.check_output([
    "git", "log",
    "--date=iso-strict-local",
    "--pretty=format:%H%x09%an%x09%ae%x09%ad%x09%s",
    "--all",
    "--reverse",
], text=True)

by_day = defaultdict(list)
for line in raw.splitlines():
    commit, author, email, date_text, subject = line.split("\t", 4)
    if author == "GitHub Actions" or subject.startswith("Version update"):
        continue
    t = datetime.fromisoformat(date_text)
    by_day[t.date().isoformat()].append((t, subject))

for day, items in sorted(by_day.items()):
    times = [t for t, _ in items]
    sessions = []
    start = last = times[0]
    for t in times[1:]:
        if t - last <= timedelta(hours=2):
            last = t
        else:
            sessions.append((start, last))
            start = last = t
    sessions.append((start, last))

    hours = sum(max(0.5, (end - start).total_seconds() / 3600 + 0.5) for start, end in sessions)
    print(day, round(hours, 1), "hours")
    for _, subject in items:
        if not subject.startswith("Merge branch"):
            print("  -", subject)
PY
```

After updating the CSV, check row count and total:

```bash
wc -l docs/git-work-log.csv
awk -F, 'NR>1 {sum += $2} END {printf "%.1f\n", sum}' docs/git-work-log.csv
```
