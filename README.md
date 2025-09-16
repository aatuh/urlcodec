# urlencoder

Encode/decode nested data structures to/from `url.Values` using a
predictable dotted/recursive syntax.

## Install

```go
import "github.com/aatuh/urlencoder"
```

## Quick start

```go
e := urlencoder.NewURLEncoder()

vals, _ := e.Encode(map[string]any{
  "user": map[string]any{
    "id":   1,
    "name": "Ada",
  },
  "tags": []string{"a", "b"},
})
// user.id=1&user.name=Ada&tags[0]=a&tags[1]=b

m, _ := e.Decode(vals)
// m["user"].(map[string]any)["name"] == "Ada"
```

## Rules

- Keys: `a`, `a.b`, `a[0]`, `a.mapKey`.
- Structs require `json` tags for field names; embedded fields are
  inlined.
- Maps must have string keys.
- Pointers/interfaces are dereferenced when non‑nil.

## Notes

- Guardrails: max recursion depth and slice size, plus basic index
  validation.
- Decoding uses an internal sparse slice helper and returns regular
  slices.
