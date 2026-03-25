# QUICK START

## Define Your Entity

```go
type Product struct {
    ID          int
    Locale      gotrans.Locale  // Required
    Title       string          // Translatable
    Description string          // Translatable
}

func (p Product) TranslationLocale() gotrans.Locale {
    return p.Locale
}

func (p Product) TranslationEntityID() int {
    return p.ID
}

func (p Product) TranslatableFields() map[string]string {
    return map[string]string{
        "Title":       "title",
        "Description": "description",
    }
}
```

## Setup

```go
repo := mysql.NewTranslationRepository(db)
translator := gotrans.NewTranslator[Product](repo)
```

## Save

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple", Description: "Fresh"},
}
err := translator.SaveTranslations(ctx, products)
```

## Load

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN},
}
products, err := translator.LoadTranslations(ctx, products)
fmt.Println(products[0].Title) // "Apple"
```

## Delete

```go
// Specific fields
translator.DeleteTranslations(ctx, gotrans.LocaleEN, "product", []int{1}, []string{"title"})

// All translations
translator.DeleteTranslationsByEntity(ctx, "product", []int{1})
```

## Multi-Locale (Auto-Optimized)

```go
products := []Product{
    {ID: 1, Locale: gotrans.LocaleEN, Title: "Apple"},
    {ID: 1, Locale: gotrans.LocaleFR, Title: "Pomme"},
    {ID: 2, Locale: gotrans.LocaleEN, Title: "Banana"},
    {ID: 2, Locale: gotrans.LocaleFR, Title: "Banane"},
}
// Makes 2 DB calls (grouped by locale), not 4
translator.SaveTranslations(ctx, products)
```

## Run Example

```bash
go run ./example/main.go
```

## Run Tests

```bash
go test -v ./...
```

