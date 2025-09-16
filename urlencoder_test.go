package urlencoder

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"testing"
)

// equalUnordered compares two slices irrespective of order.
func equalUnordered(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[any]int)
	bm := make(map[any]int)
	for _, v := range a {
		am[v]++
	}
	for _, v := range b {
		bm[v]++
	}
	return reflect.DeepEqual(am, bm)
}

// TestEncode_SimpleKey verifies that a simple key/value pair is encoded.
func TestEncode_SimpleKey(t *testing.T) {
	encoder := NewURLEncoder()
	input := map[string]any{
		"foo": "bar",
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := values.Get("foo"); got != "bar" {
		t.Errorf("expected foo=bar, got %q", got)
	}
}

// TestEncode_NestedMap verifies that nested maps produce dot-notation keys.
func TestEncode_NestedMap(t *testing.T) {
	encoder := NewURLEncoder()
	input := map[string]any{
		"data": map[string]any{
			"key": "value",
		},
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := values.Get("data.key"); got != "value" {
		t.Errorf("expected data.key=value, got %q", got)
	}
}

// TestEncode_NestedStruct verifies that a struct with proper json tags is
// encoded into dot-notation keys.
func TestEncode_NestedStruct(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	encoder := NewURLEncoder()
	input := map[string]any{
		"person": Person{
			Name: "John",
			Age:  30,
		},
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := values.Get("person.name"); got != "John" {
		t.Errorf("expected person.name=John, got %q", got)
	}
	if got := values.Get("person.age"); got != "30" {
		t.Errorf("expected person.age=30, got %q", got)
	}
}

// TestEncode_Slice verifies that a slice is encoded into indexed keys.
func TestEncode_Slice(t *testing.T) {
	encoder := NewURLEncoder()
	input := map[string]any{
		"list": []string{"a", "b", "c"},
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, expected := range []string{"a", "b", "c"} {
		key := "list[" + strconv.Itoa(i) + "]"
		if got := values.Get(key); got != expected {
			t.Errorf("expected %s=%q, got %q", key, expected, got)
		}
	}
}

// TestEncode_Map verifies that a map is encoded with keys joined by dots.
func TestEncode_Map(t *testing.T) {
	encoder := NewURLEncoder()
	input := map[string]any{
		"settings": map[string]any{
			"theme": "dark",
			"lang":  "en",
		},
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := values.Get("settings.theme"); got != "dark" {
		t.Errorf("expected settings.theme=dark, got %q", got)
	}
	if got := values.Get("settings.lang"); got != "en" {
		t.Errorf("expected settings.lang=en, got %q", got)
	}
}

// TestEncode_MissingJSONTag verifies that a struct with missing json tags
// returns an error.
func TestEncode_MissingJSONTag(t *testing.T) {
	type NoTag struct {
		Field string
	}
	encoder := NewURLEncoder()
	input := map[string]any{
		"notag": NoTag{Field: "value"},
	}
	_, err := encoder.Encode(input)
	if err == nil {
		t.Fatal("expected error for struct with missing json tag, got nil")
	}
}

// TestEncode_MapNonStringKey verifies that a map with non-string keys is
// rejected.
func TestEncode_MapNonStringKey(t *testing.T) {
	encoder := NewURLEncoder()
	input := map[string]any{
		"badMap": map[int]string{
			1: "one",
		},
	}
	_, err := encoder.Encode(input)
	if err == nil {
		t.Fatal("expected error for map with non-string keys, got nil")
	}
}

// TestEncode_AnonymousField verifies that anonymous fields are encoded correctly.
func TestEncode_AnonymousField(t *testing.T) {
	type Embedded struct {
		Field string `json:"field"`
	}
	type WithEmbedded struct {
		Embedded
		Other string `json:"other"`
	}
	encoder := NewURLEncoder()
	input := map[string]any{
		"struct": WithEmbedded{
			Embedded: Embedded{Field: "embedded"},
			Other:    "other",
		},
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect keys "struct.field" and "struct.other".
	if got := values.Get("struct.field"); got != "embedded" {
		t.Errorf("expected struct.field=embedded, got %q", got)
	}
	if got := values.Get("struct.other"); got != "other" {
		t.Errorf("expected struct.other=other, got %q", got)
	}
}

// TestEncode_NilPointer verifies that a nil pointer is handled gracefully.
func TestEncode_NilPointer(t *testing.T) {
	encoder := NewURLEncoder()
	var ptr *string = nil
	input := map[string]any{
		"nilptr": ptr,
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Nil pointer should not produce any key.
	if _, ok := values["nilptr"]; ok {
		t.Errorf("expected no key for nil pointer, got %v", values)
	}
}

// TestEncode_EmptyInput verifies that encoding an empty map returns empty url.Values.
func TestEncode_EmptyInput(t *testing.T) {
	encoder := NewURLEncoder()
	input := map[string]any{}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected empty url.Values, got %v", values)
	}
}

// TestEncode_Complex verifies a complex structure with nested maps,
// structs, slices, and time values.
func TestEncode_Complex(t *testing.T) {
	type Inner struct {
		Field string `json:"field"`
	}
	type Outer struct {
		Inner  Inner `json:"inner"`
		List   []int `json:"list"`
		Active bool  `json:"active"`
	}
	encoder := NewURLEncoder()
	input := map[string]any{
		"outer": Outer{
			Inner:  Inner{Field: "value"},
			List:   []int{1, 2, 3},
			Active: true,
		},
	}
	values, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check nested struct fields.
	if got := values.Get("outer.inner.field"); got != "value" {
		t.Errorf("expected outer.inner.field=value, got %q", got)
	}
	// Check slice encoding.
	for i, expected := range []int{1, 2, 3} {
		key := "outer.list[" + strconv.Itoa(i) + "]"
		if got := values.Get(key); got != strconv.Itoa(expected) {
			t.Errorf("expected %s=%q, got %q", key,
				strconv.Itoa(expected), got)
		}
	}
	// Check bool encoding.
	if got := values.Get("outer.active"); got != "true" {
		t.Errorf("expected outer.active=true, got %q", got)
	}
}

// TestEncode_NestedStructures encodes a nested struct with a slice.
func TestEncode_NestedStructures(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Inner Inner    `json:"inner"`
		List  []string `json:"list"`
	}
	encoder := NewURLEncoder()
	data := map[string]any{
		"obj": Outer{
			Inner: Inner{Value: "test"},
			List:  []string{"a", "b"},
		},
	}
	values, err := encoder.Encode(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v := values.Get("obj.inner.value"); v != "test" {
		t.Errorf("expected obj.inner.value to be 'test', got %v", v)
	}
	if v := values.Get("obj.list[0]"); v != "a" {
		t.Errorf("expected obj.list[0] to be 'a', got %v", v)
	}
	if v := values.Get("obj.list[1]"); v != "b" {
		t.Errorf("expected obj.list[1] to be 'b', got %v", v)
	}
}

// TestEncodeDecode_Cycle encodes a complex structure then decodes it back,
// verifying that the original structure is preserved.
func TestEncodeDecode_Cycle(t *testing.T) {
	type Inner struct {
		Field string `json:"field"`
	}
	type Outer struct {
		Inner  Inner    `json:"inner"`
		Values []string `json:"values"`
	}

	original := map[string]any{
		"outer": Outer{
			Inner:  Inner{Field: "value"},
			Values: []string{"a", "b", "c"},
		},
	}
	expected := map[string]any{
		"outer": map[string]any{
			"inner": map[string]any{
				"field": "value",
			},
			"values": []any{"a", "b", "c"},
		},
	}

	encoder := NewURLEncoder()
	values, err := encoder.Encode(original)
	if err != nil {
		t.Fatalf("unexpected error during encode: %v", err)
	}

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error during decode: %v", err)
	}

	// Normalize values by sorting slices before comparison
	normalize := func(data map[string]any) {
		if outer, ok := data["outer"].(map[string]any); ok {
			if values, ok := outer["values"].([]any); ok {
				sort.Slice(values, func(i, j int) bool {
					return fmt.Sprintf("%v", values[i]) < fmt.Sprintf("%v", values[j])
				})
			}
		}
	}

	normalize(expected)
	normalize(decoded)

	if !reflect.DeepEqual(expected, decoded) {
		t.Errorf(
			"expected decoded structure to match original.\nExpected: %#v\nDecoded: %#v",
			expected,
			decoded,
		)
	}
}

// TestDecode_SimpleKey tests decoding a simple key-value pair.
func TestDecode_SimpleKey(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("foo", "bar")

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val, ok := decoded["foo"]; !ok || val != "bar" {
		t.Errorf("expected foo=bar, got %v", decoded)
	}
}

// TestDecode_NestedKeys tests decoding nested keys using dot notation.
func TestDecode_NestedKeys(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("person.name", "John")
	values.Set("person.age", "30")

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	person, ok := decoded["person"].(map[string]any)
	if !ok {
		t.Fatalf("expected person to be map, got %T", decoded["person"])
	}
	if person["name"] != "John" {
		t.Errorf("expected name=John, got %v", person["name"])
	}
	if person["age"] != "30" {
		t.Errorf("expected age=30, got %v", person["age"])
	}
}

// TestDecode_Slice tests decoding slice values.
func TestDecode_Slice(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("list[0]", "a")
	values.Set("list[1]", "b")

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, ok := decoded["list"].([]any)
	if !ok {
		t.Fatalf("expected list to be slice, got %T", decoded["list"])
	}
	if len(list) != 2 {
		t.Errorf("expected slice length 2, got %d", len(list))
	}
	// Since order is not guaranteed, check for presence.
	foundA, foundB := false, false
	for _, v := range list {
		if v == "a" {
			foundA = true
		}
		if v == "b" {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Errorf("expected slice to contain 'a' and 'b', got %v", list)
	}
}

// TestDecode_Map tests decoding of map keys.
func TestDecode_Map(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("map.key1", "value1")
	values.Set("map.key2", "value2")

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := decoded["map"].(map[string]any)
	if !ok {
		t.Fatalf("expected map to be map, got %T", decoded["map"])
	}
	if m["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %v", m["key1"])
	}
	if m["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %v", m["key2"])
	}
}

// TestDecode_Complex tests a combination of nested keys, slices, and maps.
func TestDecode_Complex(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("user.name", "Alice")
	values.Set("user.emails[0]", "alice@example.com")
	values.Set("user.emails[1]", "alice@work.com")
	values.Set("user.address.street", "123 Main St")

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	user, ok := decoded["user"].(map[string]any)
	if !ok {
		t.Fatalf("expected user to be map, got %T", decoded["user"])
	}
	if user["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", user["name"])
	}
	emails, ok := user["emails"].([]any)
	if !ok {
		t.Fatalf("expected emails to be slice, got %T", user["emails"])
	}
	if len(emails) != 2 {
		t.Errorf("expected 2 emails, got %d", len(emails))
	}
	found1, found2 := false, false
	for _, v := range emails {
		if v == "alice@example.com" {
			found1 = true
		}
		if v == "alice@work.com" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("emails missing expected values: %v", emails)
	}
	address, ok := user["address"].(map[string]any)
	if !ok {
		t.Fatalf("expected address to be map, got %T", user["address"])
	}
	if address["street"] != "123 Main St" {
		t.Errorf("expected street=123 Main St, got %v", address["street"])
	}
}

// TestDecode_InvalidSliceIndex verifies that an invalid slice index returns an
// error.
func TestDecode_InvalidSliceIndex(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("list[abc]", "value")

	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error for invalid slice index, got nil")
	}
}

// TestDecode_ConflictingKeys tests that a conflict between a simple key and a
// nested key (e.g. "person" as a string and "person.name") causes an error.
func TestDecode_ConflictingKeys(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("person", "value")
	values.Set("person.name", "John")

	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error for conflicting keys, got nil")
	}
}

// TestDecode_Structure tests that the decoded output has the expected nested
// structure.
func TestDecode_Structure(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("a", "1")
	values.Set("b.c", "2")
	values.Set("b.d", "3")
	values.Set("e[0]", "4")
	values.Set("e[1]", "5")

	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify top-level keys.
	if decoded["a"] != "1" {
		t.Errorf("expected a=1, got %v", decoded["a"])
	}
	b, ok := decoded["b"].(map[string]any)
	if !ok {
		t.Fatalf("expected b to be map, got %T", decoded["b"])
	}
	if b["c"] != "2" {
		t.Errorf("expected b.c=2, got %v", b["c"])
	}
	if b["d"] != "3" {
		t.Errorf("expected b.d=3, got %v", b["d"])
	}
	e, ok := decoded["e"].([]any)
	if !ok {
		t.Fatalf("expected e to be slice, got %T", decoded["e"])
	}
	// Since order is not guaranteed, sort the slice for comparison.
	expected := []any{"4", "5"}
	if !equalUnordered(e, expected) {
		t.Errorf("expected e=%v, got %v", expected, e)
	}
}

// TestDecode_ExceedMaxRecursion constructs a key with more than the maximum
// allowed nesting (maxRecursionDepth=10) to force an error.
func TestDecode_ExceedMaxRecursion(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	// Build key "a.a.a.a.a.a.a.a.a.a.a" (11 dots, 12 parts)
	key := "a"
	for i := 0; i < 11; i++ {
		key = key + ".a"
	}
	values.Set(key, "value")
	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error due to exceeding max recursion depth, got nil")
	}
}

// TestDecode_MalformedSlice feeds a slice key with a non-integer index.
func TestDecode_MalformedSlice(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("list[abc]", "value")
	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error for malformed slice index, got nil")
	}
}

// TestDecode_NegativeSliceIndex tests a slice index with a negative number.
func TestDecode_NegativeSliceIndex(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("list[-1]", "value")
	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error for negative slice index, got nil")
	}
}

// TestDecode_ConflictingKeyTypes creates a conflict by using a key first as a
// scalar and then as a nested object.
func TestDecode_ConflictingKeyTypes(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("item", "scalar")
	values.Set("item.sub", "nested")
	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error due to conflicting key types, got nil")
	}
}

// TestDecode_MultipleValues verifies that when multiple values are present
// for the same key, only the first is used.
func TestDecode_MultipleValues(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Add("key", "first")
	values.Add("key", "second")
	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val, ok := decoded["key"]; !ok || val != "first" {
		t.Errorf("expected key to be 'first', got %v", decoded["key"])
	}
}

// TestDecode_EmptyKey tests the behavior when an empty key is provided.
func TestDecode_EmptyKey(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("", "empty")
	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error with empty key: %v", err)
	}
	if val, ok := decoded[""]; !ok || val != "empty" {
		t.Errorf("expected empty key to be 'empty', got %v", decoded[""])
	}
}

// TestDecode_AttackKeys uses keys containing characters that might be used
// in injection or XSS attacks to see if any unexpected parsing occurs.
func TestDecode_AttackKeys(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("attacker<script>", "xss")
	values.Set("normal.key", "value")
	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded["attacker<script>"] != "xss" {
		t.Errorf("expected attacker<script> to be 'xss', got %v",
			decoded["attacker<script>"])
	}
	nested, ok := decoded["normal"].(map[string]any)
	if !ok {
		t.Fatalf("expected normal to be a map, got %T", decoded["normal"])
	}
	if nested["key"] != "value" {
		t.Errorf("expected normal.key to be 'value', got %v", nested["key"])
	}
}

// TestDecode_SparseSliceIndices checks that sparse indices do not break
// conversion to a regular slice.
func TestDecode_SparseSliceIndices(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	values.Set("sparse[0]", "start")
	values.Set("sparse[1000000]", "end")
	decoded, err := encoder.Decode(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice, ok := decoded["sparse"].([]any)
	if !ok {
		t.Fatalf("expected sparse to be slice, got %T", decoded["sparse"])
	}
	foundStart, foundEnd := false, false
	for _, v := range slice {
		if v == "start" {
			foundStart = true
		}
		if v == "end" {
			foundEnd = true
		}
	}
	if !foundStart || !foundEnd {
		t.Errorf("expected slice to contain 'start' and 'end', got %v", slice)
	}
}

// TestDecode_ExceedMaxSliceSize adds one more element than allowed to trigger
// the maximum slice size error.
func TestDecode_ExceedMaxSliceSize(t *testing.T) {
	encoder := NewURLEncoder()
	values := url.Values{}
	sliceName := "bigSlice"
	for i := 0; i <= maxSliceSize; i++ {
		key := sliceName + "[" + strconv.Itoa(i) + "]"
		values.Set(key, "val")
	}
	_, err := encoder.Decode(values)
	if err == nil {
		t.Fatal("expected error due to exceeding max slice size, got nil")
	}
}
