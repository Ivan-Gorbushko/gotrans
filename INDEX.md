# Documentation Index

Choose what you need:

| Purpose | File | Time |
|---------|------|------|
| **Get started quickly** | [README.md](README.md) | 5 min |
| **Understand the design** | [ARCHITECTURE.md](ARCHITECTURE.md) | 10 min |
| **Quick reference** | [QUICK_START.md](QUICK_START.md) | 3 min |
| **Find answers** | [FAQ.md](FAQ.md) | 10 min |

## README.md

Your starting point. Contains:
- Feature overview
- Quick start (5 steps)
- How it works
- Database schema
- Entity name resolution
- Supported locales
- Multi-locale operations
- Use cases
- Best practices

**Read this first.**

## ARCHITECTURE.md

Deep dive into design decisions:
- Core principles
- Optimization strategy
- Save/load operations
- Database design
- Reflection justification
- Type safety details
- Performance analysis
- Limitations

**Read this to understand why.**

## QUICK_START.md

Minimal code example:
- Entity definition
- Setup
- Save/load/delete operations
- Multi-locale example

**Read this for a quick reminder.**

## FAQ.md

40+ questions organized by category:
- General questions
- Technical questions
- Field mapping
- Database
- Performance
- Testing
- Troubleshooting
- Feature requests

**Read this to find answers.**

## Quick Links

**By Task:**
- Setting up: README.md → QUICK_START.md
- Integrating: QUICK_START.md → ARCHITECTURE.md
- Troubleshooting: FAQ.md
- Learning: ARCHITECTURE.md
- Coding: example/main.go

**By Role:**
- **API Users**: README.md → QUICK_START.md → FAQ.md
- **Developers**: QUICK_START.md → ARCHITECTURE.md → gotrans.go
- **Maintainers**: ARCHITECTURE.md → gotrans.go → *_test.go

**By Question:**
- "How do I...?" → QUICK_START.md
- "Why does...?" → ARCHITECTURE.md
- "What if...?" → FAQ.md
- "Show me code" → example/main.go

## Testing

```bash
# Run all tests
go test -v ./...

# Run example
go run ./example/main.go
```

## Key Concepts

- **Embedded Locale**: Entity carries locale via `TranslationEntityLocale()`
- **Field Mapping**: Struct fields map to DB fields via `TranslatableFields()`
- **Automatic Grouping**: Translations grouped by locale for optimization
- **Type Safe**: Go generics ensure compile-time checking
- **Multi-locale**: Multiple locales optimized automatically

## Performance

| Scenario | Calls | Improvement |
|---|---|---|
| 100 entities, 1 locale | 1 | 100x |
| 100 entities, 2 locales | 2 | 50x |
| 100 entities, 5 locales | 5 | 20x |

## Supported Databases

- MySQL 5.7+
- MySQL 8.0+
- SQLite 3.x
- PostgreSQL
- Any sqlx-supported database

## Supported Languages

41 languages: English, French, German, Spanish, Italian, Russian, Chinese, Japanese, Korean, Arabic, and 31 more.

## At a Glance

```go
// Define entity
type Product struct {
    ID     int
    Locale gotrans.Locale
    Title  string
}

// Implement interface
func (p Product) TranslationEntityLocale() gotrans.Locale { return p.Locale }
func (p Product) TranslationEntityID() int { return p.ID }
func (p Product) TranslatableFields() map[string]string {
    return map[string]string{"Title": "title"}
}

// Use it
translator := gotrans.NewTranslator[Product](repo)
translator.SaveTranslations(ctx, entities)
translator.LoadTranslations(ctx, entities)
```

**Start with [README.md](README.md)** →

