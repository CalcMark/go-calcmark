---
title: "Project Workback"
summary: "Work backwards from a launch date with date arithmetic, sprint planning, and cost estimates."
weight: 40
calcmark_build: progressive
---

You have a fixed launch date and need to work backwards. This walkthrough builds a complete project workback schedule using CalcMark's date literals, duration arithmetic, the `from` keyword, and rate-based cost estimates.

The complete CalcMark file is available at {{< repo-file path="testdata/examples/project-workback.cm" >}}.

---

## Key Dates

Start by pinning the two anchors of your schedule: the launch date and today. Subtracting one date from another gives you a duration.

```cm
launch_date = Mar 15 2025
kickoff = today

Days until launch:

days_to_launch = launch_date - kickoff
```

`Mar 15 2025` is a date literal -- CalcMark parses it directly. The `today` keyword resolves to the current date at runtime. Subtracting two dates produces a duration in days.

**CalcMark features:** Date literals (`Mar 15 2025`); `today` keyword; date subtraction producing a duration.

---

## Phase Durations

Break the project into phases. CalcMark understands `weeks` and `days` as duration units, and you can add them together freely.

```cm
development_time = 4 weeks
qa_time = 2 weeks
staging_time = 1 week
launch_prep = 5 days

Total planned time:

total_planned = development_time + qa_time + staging_time + launch_prep
```

Durations with different units combine naturally: `4 weeks + 2 weeks + 1 week + 5 days` resolves to a single duration. CalcMark normalizes the result for you.

**CalcMark features:** Duration literals (`weeks`, `days`); duration addition across units.

---

## Buffer Calculation

Subtract the total planned time from the available time to see how much slack you have.

```cm
buffer = days_to_launch - total_planned
```

A positive buffer means you have room. A negative one means you are already behind.

**CalcMark features:** Duration subtraction.

---

## Working Backwards from Launch

This is the core of a workback schedule. Subtract each phase duration from a date to get milestone start dates.

```cm
launch_prep_start = launch_date - launch_prep
staging_start = launch_prep_start - staging_time
qa_start = staging_start - qa_time
dev_start = qa_start - development_time
```

Subtracting a duration from a date produces a new date. You chain these to walk backwards through every phase, from launch day all the way to development kickoff.

**CalcMark features:** Date arithmetic (date minus duration produces a date); chained date calculations.

---

## Sprint Planning

With the development window defined, you can plan sprints and estimate total capacity.

```cm
num_sprints = 2
sprint_duration = 2 weeks

Points per sprint (team velocity):

team_velocity = 40
total_points = num_sprints * team_velocity
```

Plain arithmetic works alongside dates and durations. `2 * 40 = 80` story points across two sprints.

**CalcMark features:** Plain arithmetic; markdown prose between calculations.

---

## Key Milestones

The `from` keyword adds a duration to a date. Use it with `today` for quick relative dates, or with any calculated date for schedule milestones.

```cm
one_week_from_now = 1 week from today
two_weeks_from_now = 2 weeks from today
next_month = 1 month from today

Milestone dates calculated from schedule:

design_review = dev_start + 1 week
first_sprint_end = dev_start + 2 weeks
feature_freeze = dev_start + 3 weeks
code_complete = qa_start
```

`1 week from today` reads like natural language. It is equivalent to `today + 1 week`. You can also add durations to dates with the `+` operator directly.

**CalcMark features:** `from` keyword (`1 week from today`); date plus duration; `month` duration unit.

---

## Resource Allocation

Count your team, then multiply by phase duration to get person-weeks.

```cm
developers = 4
qa_engineers = 2
devops = 1

Person-weeks calculation (4 weeks dev, 2 weeks QA, ~2 weeks devops):

dev_person_weeks = developers * 4
qa_person_weeks = qa_engineers * 2
devops_person_weeks = devops * 2

total_person_weeks = dev_person_weeks + qa_person_weeks + devops_person_weeks
```

Simple multiplication keeps the estimate transparent. 22 person-weeks across three roles.

**CalcMark features:** Plain arithmetic; variable reuse.

---

## Daily Rate Cost Estimate

Rate literals like `$800/day` represent a cost per unit of time. The `over` keyword accumulates a rate across a duration.

```cm
dev_daily_rate = $800/day
qa_daily_rate = $700/day
devops_daily_rate = $900/day

Team cost over development phase:

dev_cost = dev_daily_rate * developers over development_time
qa_cost = qa_daily_rate * qa_engineers over qa_time
devops_cost = devops_daily_rate * devops over (staging_time + launch_prep)
```

`$800/day * 4 over 4 weeks` multiplies the daily rate by the headcount, then accumulates over the duration. The result is a currency value representing total spend for that phase.

**CalcMark features:** Rate literals (`$800/day`); `over` keyword for rate accumulation over a duration; currency arithmetic.

---

## Risk Assessment

Add a contingency buffer and shift the start date earlier.

```cm
risk_buffer = 8 days
adjusted_dev_start = dev_start - risk_buffer
```

Subtracting `8 days` from a date moves it earlier. Your adjusted kickoff now accounts for unknowns.

**CalcMark features:** Duration literals (`days`); date minus duration.

---

## Calendar Summary

Assign meaningful names to each milestone for a clean calendar view.

```cm
project_kickoff = adjusted_dev_start
dev_complete = qa_start
qa_complete = staging_start
staging_complete = launch_prep_start
go_live = launch_date
```

Variable aliasing makes the final output readable. Each name maps to a concrete date calculated earlier.

**CalcMark features:** Variable aliasing for readability.

---

## Schedule Feasibility

Compare available time against needed time to confirm the schedule is realistic.

```cm
available_time = days_to_launch
needed_time = total_planned + risk_buffer
```

If `needed_time` exceeds `available_time`, you need to cut scope or extend the deadline.

**CalcMark features:** Duration comparison; duration addition.

---

## Features Demonstrated

This example showcases the following CalcMark features:

- **Date literals** -- `Mar 15 2025`
- **`today` keyword** -- resolves to the current date at runtime
- **Date arithmetic** -- subtracting dates, adding/subtracting durations from dates
- **Duration literals** -- `weeks`, `days`, `month`
- **Duration arithmetic** -- adding and subtracting mixed-unit durations
- **`from` keyword** -- `1 week from today` for natural-language date offsets
- **Rate literals** -- `$800/day`, `$700/day`
- **`over` keyword** -- `rate * count over duration` for cost accumulation
- **Plain arithmetic** -- sprint velocity, person-weeks, headcount
- **Markdown prose** -- headings, paragraphs, and inline comments between calculations

## Try It

{{< repo-file path="testdata/examples/project-workback.cm" >}}

```bash
cm testdata/examples/project-workback.cm
```
