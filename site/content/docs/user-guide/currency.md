---
title: "Currency & Exchange Rates"
weight: 2
---

CalcMark supports currency values and conversion between currencies using exchange rates defined in YAML frontmatter.

### Currency Conversion {#currency-conversion}

Convert between currencies using `in` with exchange rates defined in YAML frontmatter:

```calcmark
---
exchange:
  USD_EUR: 0.92
  EUR_GBP: 0.86
---

price_usd = $100
price_eur = price_usd in EUR

salary = 50000 EUR
salary_gbp = salary in GBP
```

Exchange rates use the format `FROM_TO: rate` where 1 unit of FROM equals `rate` units of TO.

---

**Related guide:** [Business Planning](/guides/business-planning/) — currency arithmetic, ratios, and financial modeling
