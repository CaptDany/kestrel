<picture>
  <source media="(prefers-color-scheme: dark)" srcset="ui/static/logo/kestrel.svg">
  <img alt="Kestrel" src="ui/static/logo/kestrel.svg" height="32">
</picture>

A self-hosted purchase planner that calendarizes e-commerce purchases based on your paydays and budget.

Paste a link from any supported store (Amazon, MercadoLibre, Aliexpress, IKEA, Walmart) and kestrel automatically extracts the price and schedules the purchase on an upcoming payday.

## Features

- **Price Extraction**: Paste any product URL — kestrel auto-detects title, price, and currency
- **Payday Scheduling**: Configure your payday schedule (monthly, biweekly, weekly)
- **Budget Modes**: Per-payday, flexible (base + bonuses), or total pool
- **Smart Sorting**: Sort by price, priority, date added, or desired date
- **Purchase Modes**: One item per cycle or as many as budget allows
- **Saving**: Large items automatically enter saving mode across multiple cycles
- **Playwright Support**: Optional browser-based scraper for JavaScript-heavy sites
- **Self-Hosted**: Single binary, SQLite-backed, runs anywhere

## Quick Start

```bash
docker compose up -d
```

Open http://localhost:8000

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `KESTREL_PORT` | `8000` | HTTP port |
| `KESTREL_DB_PATH` | `data/kestrel.db` | SQLite database path |

## Supported Stores

- Amazon (all country TLDs)
- MercadoLibre / MercadoLivre
- Aliexpress
- IKEA
- Walmart

## License

MIT
