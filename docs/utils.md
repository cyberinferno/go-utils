# Utils Documentation

The `utils` package provides common helper functions for arrays, bytes, strings, JSON validation, time conversion, Discord notifications, and boolean formatting. These utilities are designed to be small, focused, and easy to use across Go projects.

## Features

- **Array**: Random element selection from slices (generic)
- **Bool**: Human-readable "Yes"/"No" conversion
- **Bytes**: Fixed-length string buffers and byte slice concatenation
- **Pointer**: Convert any value to a pointer (generic)
- **Discord**: Send messages to Discord channels via webhooks
- **JSON**: Validation of JSON object strings
- **String**: Null-terminated string reading and random alphanumeric generation
- **Time**: GMT/UTC to IST (Indian Standard Time) conversion

## Installation

```go
import "github.com/cyberinferno/go-utils/utils"
```

## Array Utilities

### GetRandomElement

Returns a randomly chosen element from a slice. Works with any type via generics. The slice must be non-empty; otherwise the function panics.

```go
import "github.com/cyberinferno/go-utils/utils"

// With a slice of strings
options := []string{"rock", "paper", "scissors"}
choice := utils.GetRandomElement(options) // one of "rock", "paper", "scissors"

// With a slice of integers
ids := []int{101, 102, 103}
id := utils.GetRandomElement(ids)

// With a single-element slice (always returns that element)
only := utils.GetRandomElement([]int{42}) // 42
```

**Parameters:**

- **arr**: The slice to pick from (must have at least one element)

**Returns:**

- A random element of type T from the slice

**Note:** For deterministic tests, seed `math/rand` before calling (e.g. `rand.Seed(seed)` or use a custom source).

---

## Bool Utilities

### BoolToYesNo

Converts a boolean to a human-readable "Yes" or "No" string. Useful for display in UIs, logs, or API responses.

```go
import "github.com/cyberinferno/go-utils/utils"

active := true
fmt.Println(utils.BoolToYesNo(active)) // "Yes"

enabled := false
fmt.Println(utils.BoolToYesNo(enabled)) // "No"
```

**Parameters:**

- **value**: The boolean to convert

**Returns:**

- `"Yes"` if value is true, `"No"` if value is false

---

## Pointer Utilities

### Pointer

Returns a pointer to the given value. Generic over any type `T`; useful when you need a `*T` (e.g. optional struct fields, API payloads, or functions that accept pointers).

```go
import "github.com/cyberinferno/go-utils/utils"

// Primitives
n := utils.Pointer(42)   // *int
s := utils.Pointer("hi") // *string
b := utils.Pointer(true) // *bool

// Structs and custom types
type Config struct{ Host string }
cfg := utils.Pointer(Config{Host: "localhost"}) // *Config

// Optional field in JSON/API payloads
payload := map[string]*int{"count": utils.Pointer(10)}
```

**Parameters:**

- **value**: The value to convert to a pointer (any type T)

**Returns:**

- A pointer to the given value (`*T`)

---

## Bytes Utilities

### MakeFixedLengthStringBytes

Creates a byte slice of a fixed length containing the string's bytes. If the string is shorter than the length, the remainder is zero-padded; if longer, the string is truncated. Useful for fixed-width binary formats or buffers (e.g. C interop, fixed-size message fields).

```go
import "github.com/cyberinferno/go-utils/utils"

// Short string: zero-padded
buf := utils.MakeFixedLengthStringBytes("hi", 5) // []byte{'h','i',0,0,0}

// Exact length: unchanged
buf = utils.MakeFixedLengthStringBytes("hello", 5) // []byte("hello")

// Long string: truncated
buf = utils.MakeFixedLengthStringBytes("hello world", 5) // []byte("hello")
```

**Parameters:**

- **str**: The string to convert to bytes
- **length**: The fixed length of the resulting byte slice

**Returns:**

- A byte slice of exactly `length` bytes with the string content (padded or truncated as needed)

### JoinBytes

Concatenates one or more byte slices into a single byte slice. Similar to `strings.Join` but for `[]byte`.

```go
import "github.com/cyberinferno/go-utils/utils"

a := []byte("foo")
b := []byte("bar")
c := []byte("baz")
result := utils.JoinBytes(a, b, c) // []byte("foobarbaz")

// Single slice
result = utils.JoinBytes([]byte("only")) // []byte("only")

// No arguments returns empty slice
result = utils.JoinBytes() // []byte{}
```

**Parameters:**

- **s**: One or more byte slices to concatenate (variadic)

**Returns:**

- A new byte slice containing all input slices in order

---

## Discord Utilities

### SendDiscordNotification

Sends a message to a Discord channel via its webhook URL. The function performs a synchronous HTTP POST; errors are ignored (no return value). Use this for fire-and-forget notifications (e.g. alerts, logs). For error handling or retries, implement your own HTTP call.

```go
import "github.com/cyberinferno/go-utils/utils"

webhookURL := "https://discord.com/api/webhooks/123456/abcdef..."
content := "Deployment completed successfully"
utils.SendDiscordNotification(webhookURL, content)
```

**Parameters:**

- **webhook**: The Discord webhook URL to POST to (from Discord channel settings → Integrations → Webhooks)
- **content**: The message content (used as the `content` field in the JSON body; avoid embedding unescaped quotes in content for valid JSON)

**Note:** The request body is `{"content": "<content>"}`. Content is not JSON-escaped; if it contains `"` or other special characters, the payload may be invalid. For rich content, consider building the JSON body yourself and using `net/http` directly.

---

## JSON Utilities

### IsJsonString

Reports whether a string is valid JSON in object form. It attempts to unmarshal the string into a `map[string]interface{}` and returns true only if unmarshaling succeeds. JSON arrays or primitives (string, number, boolean) return false.

```go
import "github.com/cyberinferno/go-utils/utils"

utils.IsJsonString(`{}`)                // true
utils.IsJsonString(`{"a":1}`)          // true
utils.IsJsonString(`{"key": "value"}`) // true
utils.IsJsonString(``)                 // false
utils.IsJsonString(`not json`)         // false
utils.IsJsonString(`[1,2,3]`)          // false (array, not object)
utils.IsJsonString(`"hello"`)          // false (primitive)
```

**Parameters:**

- **s**: The string to validate

**Returns:**

- `true` if s is valid JSON representing an object; `false` otherwise

---

## String Utilities

### ReadStringFromBytes

Interprets a byte slice as a null-terminated string. Returns the string up to the first null byte (`0x00`), or the entire buffer if no null byte is present. Useful when reading fixed-size buffers from C code, binary protocols, or file formats.

```go
import "github.com/cyberinferno/go-utils/utils"

buf := []byte("hello\x00world")
s := utils.ReadStringFromBytes(buf) // "hello"

buf = []byte("no-null-here")
s = utils.ReadStringFromBytes(buf) // "no-null-here"

buf = []byte("\x00empty")
s = utils.ReadStringFromBytes(buf) // ""

s = utils.ReadStringFromBytes(nil)  // ""
```

**Parameters:**

- **buffer**: The byte slice to read from

**Returns:**

- The string content before the first null byte, or the whole buffer as a string if no null is present; empty string for nil or empty buffer

### GenerateRandomString

Creates a string of the given length consisting of random alphanumeric characters (`a-z`, `A-Z`, `0-9`). Useful for tokens, temporary IDs, or random identifiers.

```go
import "github.com/cyberinferno/go-utils/utils"

token := utils.GenerateRandomString(32)  // e.g. "aB3xY9kL2..."
short := utils.GenerateRandomString(8)  // e.g. "k9Fm2xQa"
empty := utils.GenerateRandomString(0)  // ""
```

**Parameters:**

- **length**: The desired length of the output string (0 for empty string)

**Returns:**

- A random alphanumeric string of exactly `length` characters

**Note:** For cryptographic use, use `crypto/rand` with encoding (e.g. base64) instead. This function uses `math/rand`.

---

## Time Utilities

### ConvertGMTtoIST

Parses a datetime string in GMT and returns the same instant formatted in IST (UTC+5:30). The input must use the layout `2006-01-02 15:04:05` in the GMT timezone.

```go
import (
	"fmt"
	"github.com/cyberinferno/go-utils/utils"
)

ist, err := utils.ConvertGMTtoIST("2024-01-15 12:00:00")
if err != nil {
	// invalid layout or timezone
	return err
}
fmt.Println(ist) // "2024-01-15 17:30:00"
```

**Parameters:**

- **gmtDatetime**: Datetime string in `2006-01-02 15:04:05` format, interpreted as GMT

**Returns:**

- The same instant formatted as `2006-01-02 15:04:05` in IST
- An error if parsing fails (e.g. invalid layout or timezone)

### ConvertUTCtoIST

Parses a datetime string in UTC (RFC 3339–style with `Z` suffix) and returns the same instant formatted in IST (UTC+5:30). The input must use the layout `2006-01-02T15:04:05Z`.

```go
import (
	"fmt"
	"github.com/cyberinferno/go-utils/utils"
)

ist, err := utils.ConvertUTCtoIST("2024-01-15T12:00:00Z")
if err != nil {
	return err
}
fmt.Println(ist) // "2024-01-15 17:30:00"
```

**Parameters:**

- **utcDatetime**: Datetime string in `2006-01-02T15:04:05Z` format (UTC)

**Returns:**

- The same instant formatted as `2006-01-02 15:04:05` in IST
- An error if parsing fails (e.g. invalid layout)

---

## Type Reference

### Array

| Function            | Signature                    | Description                          |
|---------------------|-----------------------------|--------------------------------------|
| GetRandomElement    | `func GetRandomElement[T any](arr []T) T` | Random element from slice; panics if empty. |

### Bool

| Function    | Signature                        | Description                |
|------------|-----------------------------------|----------------------------|
| BoolToYesNo| `func BoolToYesNo(value bool) string` | "Yes" or "No" from bool.   |

### Pointer

| Function | Signature                | Description                    |
|----------|---------------------------|--------------------------------|
| Pointer  | `func Pointer[T any](value T) *T` | Returns a pointer to the value. |

### Bytes

| Function                   | Signature                                      | Description                    |
|---------------------------|------------------------------------------------|--------------------------------|
| MakeFixedLengthStringBytes| `func MakeFixedLengthStringBytes(str string, length int) []byte` | Fixed-length bytes, padded/truncated. |
| JoinBytes                 | `func JoinBytes(s ...[]byte) []byte`           | Concatenate byte slices.       |

### Discord

| Function                 | Signature                                           | Description                    |
|-------------------------|-----------------------------------------------------|--------------------------------|
| SendDiscordNotification | `func SendDiscordNotification(webhook string, content string)` | POST message to Discord webhook. |

### JSON

| Function      | Signature                    | Description                    |
|--------------|------------------------------|--------------------------------|
| IsJsonString | `func IsJsonString(s string) bool` | True if s is valid JSON object. |

### String

| Function             | Signature                              | Description                        |
|---------------------|----------------------------------------|------------------------------------|
| ReadStringFromBytes | `func ReadStringFromBytes(buffer []byte) string` | String up to first null byte.      |
| GenerateRandomString| `func GenerateRandomString(length int) string`   | Random alphanumeric string.         |

### Time

| Function         | Signature                                | Description                    |
|-----------------|-------------------------------------------|--------------------------------|
| ConvertGMTtoIST | `func ConvertGMTtoIST(gmtDatetime string) (string, error)` | GMT → IST, layout `2006-01-02 15:04:05`. |
| ConvertUTCtoIST | `func ConvertUTCtoIST(utcDatetime string) (string, error)` | UTC → IST, layout `2006-01-02T15:04:05Z`. |

---

## Complete Examples

### Example 1: Random choice and display

```go
package main

import (
	"fmt"
	"github.com/cyberinferno/go-utils/utils"
)

func main() {
	options := []string{"Yes", "No", "Maybe"}
	answer := utils.GetRandomElement(options)
	fmt.Printf("Answer: %s\n", answer)

	display := utils.BoolToYesNo(answer == "Yes")
	fmt.Printf("Display: %s\n", display)
}
```

### Example 2: Fixed-size buffer and null-terminated read

```go
package main

import (
	"fmt"
	"github.com/cyberinferno/go-utils/utils"
)

func main() {
	// Write a null-terminated string into a 64-byte buffer
	buf := utils.MakeFixedLengthStringBytes("hello", 64)
	// Read back the string (stops at null)
	s := utils.ReadStringFromBytes(buf)
	fmt.Println(s) // "hello"
}
```

### Example 3: Validate JSON and send Discord alert

```go
package main

import (
	"log"
	"github.com/cyberinferno/go-utils/utils"
)

func main() {
	payload := `{"event": "deploy", "status": "ok"}`
	if !utils.IsJsonString(payload) {
		log.Fatal("invalid JSON payload")
	}
	// Process payload...
	utils.SendDiscordNotification(webhookURL, "Deploy completed: "+payload)
}
```

### Example 4: UTC timestamps to IST

```go
package main

import (
	"fmt"
	"log"
	"github.com/cyberinferno/go-utils/utils"
)

func main() {
	// From API that returns UTC with Z
	utc := "2024-06-01T09:00:00Z"
	ist, err := utils.ConvertUTCtoIST(utc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("IST: %s\n", ist) // "2024-06-01 14:30:00"
}
```

---

## Best Practices

1. **Pointer**: Use when building structs or maps that require `*T` (e.g. optional fields, JSON with `omitempty`, or APIs that distinguish null from zero value). Avoid storing the result in long-lived globals if the pointed-to value could be stack-allocated.
2. **GetRandomElement**: Ensure the slice is non-empty (e.g. check `len(arr) > 0`) to avoid panics, or use a dedicated "empty" value for your type.
3. **SendDiscordNotification**: Keep webhook URLs in configuration (environment variables or secrets), not in source code. For important alerts, consider adding retries or a wrapper that logs errors.
4. **IsJsonString**: Use when you specifically need a JSON *object*. For arrays or raw values, unmarshal into `json.RawMessage` or a concrete type and check errors instead.
5. **GenerateRandomString**: Use for non-security randomness (e.g. IDs, test data). For secrets or tokens, use `crypto/rand` with a safe encoding.
6. **Time conversions**: Both GMT and UTC conversion functions assume the input is in the stated format; invalid layout or timezone returns an error. Handle errors in production.

---

## Limitations

- **GetRandomElement**: Uses `math/rand`; not cryptographically secure. Empty slice causes panic.
- **SendDiscordNotification**: No error return; content is not JSON-escaped (unsafe if content contains `"` or control characters).
- **IsJsonString**: Only accepts JSON objects; arrays and primitives return false.
- **ConvertGMTtoIST / ConvertUTCtoIST**: Fixed input layouts only; no parsing of other formats (e.g. RFC3339 with offset).
