package urlcodec

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	maxRecursionDepth = 10   // Maximum allowed depth for nested structures
	maxSliceSize      = 1000 // Maximum allowed size for slices

	// Matches a string with a word followed by "[" and a number in decimal
	// (base 10) and "]" e.g. "mySlice[0]" matches as "mySlice" and "0"
	sliceRegexp = `(\w+)\[(\d+)\]`
)

// URLEncoder encodes and decodes URL values.
type URLEncoder struct{}

// NewURLEncoder returns a new URLEncoder.
//
// Returns:
//   - *URLEncoder: The new URLEncoder.
func NewURLEncoder() *URLEncoder {
	return &URLEncoder{}
}

// Encode encodes URL data and supports the following recursive URL syntax:
// someKey=value
// someStruct.field=value
// someSlice[0]=value
// someMap.key=value
//
// It will return an error if a "json" tag is not found for a struct field.
//
// Parameters:
//   - data: Data to encode
//
// Returns:
//   - url.Values: URL values
//   - error: Error
func (e URLEncoder) Encode(data map[string]any) (url.Values, error) {
	values := url.Values{}
	for key, value := range data {
		err := encodeURL(&values, key, reflect.ValueOf(value))
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}

// Decode decodes URL values and supports the following recursive URL syntax:
// someKey=value
// someStruct.field=value
// someSlice[0]=value
// someMap.key=value
//
// Parameters:
//   - values: URL values
//
// Returns:
//   - map[string]any: Decoded data
//   - error: Error
func (e URLEncoder) Decode(values url.Values) (map[string]any, error) {
	return decodeURL(values)
}

// decodeURL decodes an URL.
func decodeURL(values url.Values) (map[string]any, error) {
	urlData := make(map[string]any)
	depth := 0
	for key, value := range values {
		var err error
		depth, err = setNestedMapValue(urlData, key, value[0], depth)
		if err != nil {
			return nil, err
		}
	}
	convertMinSlicesToRegularSlices(urlData)
	return urlData, nil
}

// convertMinSlicesToRegularSlices converts all MinSlice instances in the map to
// regular slices recursively.
func convertMinSlicesToRegularSlices(data map[string]any) {
	for key, value := range data {
		switch v := value.(type) {
		case *minSlice:
			data[key] = v.toSlice()
		case map[string]any:
			convertMinSlicesToRegularSlices(v)
		}
	}
}

// encodeURL encodes an URL.
func encodeURL(values *url.Values, fieldTag string, v reflect.Value) error {
	return encodeValue(values, fieldTag, v)
}

// encodeValue encodes a value.
func encodeValue(values *url.Values, fieldTag string, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return encodePointer(values, fieldTag, v)
	case reflect.String:
		return encodeString(values, fieldTag, v)
	case reflect.Int, reflect.Int32, reflect.Int64:
		return encodeInt(values, fieldTag, v)
	case reflect.Float32, reflect.Float64:
		return encodeFloat(values, fieldTag, v)
	case reflect.Bool:
		return encodeBool(values, fieldTag, v)
	case reflect.Slice:
		return encodeSlice(values, fieldTag, v)
	case reflect.Map:
		return encodeMap(values, fieldTag, v)
	case reflect.Struct:
		return encodeStruct(values, fieldTag, v)
	default:
		return fmt.Errorf(
			"value type not supported by URL encoding: %s",
			v.Kind(),
		)
	}
}

// encodePointer encodes a pointer.
func encodePointer(values *url.Values, fieldTag string, v reflect.Value) error {
	if !v.IsNil() {
		return encodeValue(values, fieldTag, v.Elem())
	}
	return nil
}

// encodeString encodes a string.
func encodeString(values *url.Values, fieldTag string, v reflect.Value) error {
	values.Set(fieldTag, v.String())
	return nil
}

// encodeInt encodes an int.
func encodeInt(values *url.Values, fieldTag string, v reflect.Value) error {
	values.Set(fieldTag, fmt.Sprintf("%d", v.Int()))
	return nil
}

// encodeFloat encodes a float.
func encodeFloat(values *url.Values, fieldTag string, v reflect.Value) error {
	values.Set(fieldTag, fmt.Sprintf("%f", v.Float()))
	return nil
}

// encodeBool encodes a bool.
func encodeBool(values *url.Values, fieldTag string, v reflect.Value) error {
	values.Set(fieldTag, strconv.FormatBool(v.Bool()))
	return nil
}

// encodeSlice encodes a slice by encoding each element.
func encodeSlice(values *url.Values, fieldTag string, v reflect.Value) error {
	for j := 0; j < v.Len(); j++ {
		sliceElem := v.Index(j)
		newFieldTag := fmt.Sprintf("%s[%d]", fieldTag, j)
		if err := encodeValue(values, newFieldTag, sliceElem); err != nil {
			return err
		}
	}
	return nil
}

// encodeMap encodes a map.
func encodeMap(values *url.Values, fieldTag string, v reflect.Value) error {
	// Only support maps with string keys.
	if v.Type().Key().Kind() != reflect.String {
		return fmt.Errorf(
			"map keys must be strings, got %s", v.Type().Key().Kind(),
		)
	}
	for _, key := range v.MapKeys() {
		keyStr := key.String()
		newFieldTag := keyStr
		if fieldTag != "" {
			newFieldTag = fieldTag + "." + keyStr
		}
		if err := encodeValue(
			values, newFieldTag, v.MapIndex(key),
		); err != nil {
			return err
		}
	}
	return nil
}

// encodeStruct encodes a struct.
func encodeStruct(values *url.Values, fieldTag string, v reflect.Value) error {
	for i := 0; i < v.NumField(); i++ {
		if err := encodeStructField(values, fieldTag, v, i); err != nil {
			return err
		}
	}
	return nil
}

// encodeStructField encodes a struct field.
func encodeStructField(
	values *url.Values, fieldTag string, v reflect.Value, i int,
) error {
	field := v.Field(i)
	fieldType := v.Type().Field(i)

	if fieldType.Anonymous {
		if err := encodeValue(values, fieldTag, field); err != nil {
			return err
		}
		return nil
	}

	newFieldTag := fieldType.Tag.Get("json")
	if newFieldTag == "-" || newFieldTag == "" {
		return fmt.Errorf(
			"cannot encode field %q because it has no json tag", fieldType.Name,
		)
	}

	if fieldTag != "" {
		newFieldTag = fieldTag + "." + newFieldTag
	}
	if err := encodeValue(values, newFieldTag, field); err != nil {
		return err
	}

	return nil
}

// setNestedMapValue sets the value of a nested map.
func setNestedMapValue(
	current map[string]any, key string, value any, depth int,
) (int, error) {
	// Handle empty key explicitly.
	if key == "" {
		if _, exists := current[""]; exists {
			return depth, fmt.Errorf("conflicting key: empty key already set")
		}
		current[""] = value
		return depth, nil
	}

	parts := strings.Split(key, ".")
	if len(parts) > maxRecursionDepth {
		return depth, fmt.Errorf(
			"exceeded maximum recursion depth of %d", maxRecursionDepth,
		)
	}

	for i, part := range parts {
		// Increase depth per level.
		depth++
		if i == len(parts)-1 {
			return depth, setFinalValue(current, part, value)
		}
		var err error
		current, err = getIntermediateValue(current, part)
		if err != nil {
			return depth, err
		}
	}
	return depth, nil
}

// setFinalValue sets the value of the final key.
func setFinalValue(current map[string]any, part string, value any) error {
	reg := regexp.MustCompile(sliceRegexp)
	// If part appears to be a slice but doesn't match valid format, error.
	if strings.Contains(part, "[") && strings.Contains(part, "]") {
		if sliceIndex := reg.FindStringSubmatch(part); sliceIndex == nil {
			return fmt.Errorf("invalid slice index: %q", part)
		}
	}
	if sliceIndex := reg.FindStringSubmatch(part); sliceIndex != nil {
		return setSliceValue(current, sliceIndex, value)
	}
	if _, exists := current[part]; exists {
		return fmt.Errorf("conflicting key: %q already set", part)
	}
	current[part] = value
	return nil
}

// setSliceValue sets the value of a slice element.
func setSliceValue(
	current map[string]any, sliceIndex []string, value any,
) error {
	sliceName, idx, err := parseSliceIndex(sliceIndex)
	if err != nil {
		return err
	}
	slice, err := getOrCreateSlice(current, sliceName)
	if err != nil {
		return err
	}
	slice.set(idx, value)
	current[sliceName] = slice // Use MinSlice to handle slice elements safely
	return nil
}

// getIntermediateValue gets the intermediate value of a nested key. It uses
// regexp to check if the key is a slice index.
func getIntermediateValue(
	current map[string]any, part string,
) (map[string]any, error) {
	reg := regexp.MustCompile(sliceRegexp)
	if sliceIndex := reg.FindStringSubmatch(part); sliceIndex != nil {
		return createMapIntoSlice(sliceIndex, current)
	}
	// Create a map with the part name if it doesn't exist
	if _, ok := current[part]; !ok {
		current[part] = make(map[string]any)
	}
	return getMap(current, part)
}

// getMap returns a map from the current map.
func getMap(current map[string]any, part string) (map[string]any, error) {
	retMap, ok := current[part]
	if !ok {
		return nil, fmt.Errorf("expected map[string]any, got %T", current[part])
	}
	cast, ok := retMap.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any, got %T", retMap)
	}
	return cast, nil
}

// createMapIntoSlice creates a map inside a slice and returns it.
func createMapIntoSlice(
	sliceIndex []string, current map[string]any,
) (map[string]any, error) {
	sliceName, idx, err := parseSliceIndex(sliceIndex)
	if err != nil {
		return nil, err
	}
	slice, err := getOrCreateSlice(current, sliceName)
	if err != nil {
		return nil, err
	}
	// Ensure the element at idx is a map and initialize if necessary
	elem, exists := slice.get(idx)
	if !exists {
		elem = make(map[string]any)
		slice.set(idx, elem)
	}
	// Ensure elem is a map
	castedElem, ok := elem.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected map[string]any, got %T", elem)
	}
	current[sliceName] = slice
	return castedElem, nil
}

// parseSliceIndex returns the slice name and index from a slice index string.
func parseSliceIndex(sliceIndex []string) (string, int, error) {
	if len(sliceIndex) != 3 {
		return "", 0, fmt.Errorf("invalid slice index: %v", sliceIndex)
	}
	// For example, "mySlice[0]" gives sliceName "mySlice" and index "0".
	sliceName, index := sliceIndex[1], sliceIndex[2]
	idx, err := strconv.Atoi(index)
	if err != nil {
		return "", 0, fmt.Errorf("invalid index: %s", index)
	}
	if idx < 0 {
		return "", 0, fmt.Errorf("invalid negative index: %d", idx)
	}
	return sliceName, idx, nil
}

// getOrCreateSlice returns a slice or creates a new one if it doesn't exist.
func getOrCreateSlice(
	current map[string]any,
	sliceName string,
) (*minSlice, error) {
	if _, ok := current[sliceName]; !ok {
		current[sliceName] = newMinSlice()
	}
	minSlice, ok := current[sliceName].(*minSlice)
	if !ok {
		return nil, fmt.Errorf("expected *minSlice, got %T", current[sliceName])
	}
	if len(minSlice.elements) >= maxSliceSize {
		return nil, fmt.Errorf(
			"exceeded maximum slice size of %d",
			maxSliceSize,
		)
	}
	return minSlice, nil
}

// minSlice keeps track of slice elements with minimal length
type minSlice struct {
	elements map[int]any
}

// newMinSlice returns a new MinSlice
func newMinSlice() *minSlice {
	return &minSlice{elements: make(map[int]any)}
}

// set sets the value at the given index
func (s *minSlice) set(index int, value any) {
	s.elements[index] = value
}

// get returns the value at the given index
func (s *minSlice) get(index int) (any, bool) {
	value, exists := s.elements[index]
	return value, exists
}

// toSlice converts the MinSlice to a regular slice
func (s *minSlice) toSlice() []any {
	slice := make([]any, 0, len(s.elements))
	for _, value := range s.elements {
		slice = append(slice, value)
	}
	return slice
}
