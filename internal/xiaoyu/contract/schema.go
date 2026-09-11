package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
)

var ErrToolArgumentsInvalid = errors.New("xiaoyu tool arguments are invalid")

// ValidateArguments enforces the small, deterministic JSON-Schema subset used
// by AGMP Tool contracts before a Domain Handler runs. It intentionally avoids
// embedding a second general-purpose schema engine in the security boundary.
// Unknown schema keywords are ignored; supported constraints are type,
// properties, required, additionalProperties, items and enum.
func ValidateArguments(schema map[string]any, args map[string]any) error {
	if len(schema) == 0 {
		return nil
	}
	if args == nil {
		args = map[string]any{}
	}
	if err := validateSchemaValue(schema, args, "$", true); err != nil {
		return fmt.Errorf("%w: %v", ErrToolArgumentsInvalid, err)
	}
	return nil
}

func validateSchemaValue(schema map[string]any, value any, path string, root bool) error {
	if allowed, ok := schema["enum"].([]any); ok && len(allowed) > 0 {
		matched := false
		for _, item := range allowed {
			if reflect.DeepEqual(item, value) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%s is not an allowed value", path)
		}
	}

	typeName, _ := schema["type"].(string)
	if root && typeName == "" {
		typeName = "object"
	}
	if typeName != "" && !matchesJSONType(typeName, value) {
		return fmt.Errorf("%s must be %s", path, typeName)
	}

	switch typeName {
	case "object", "":
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		properties := schemaMap(schema["properties"])
		for _, name := range stringSlice(schema["required"]) {
			if _, exists := object[name]; !exists {
				return fmt.Errorf("%s.%s is required", path, name)
			}
		}
		additionalAllowed := true
		if raw, exists := schema["additionalProperties"]; exists {
			if flag, ok := raw.(bool); ok {
				additionalAllowed = flag
			}
		}
		for name, child := range object {
			childSchema, known := properties[name]
			if !known {
				if !additionalAllowed {
					return fmt.Errorf("%s.%s is not allowed", path, name)
				}
				continue
			}
			if err := validateSchemaValue(childSchema, child, path+"."+name, false); err != nil {
				return err
			}
		}
	case "array":
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return nil
		}
		array, ok := value.([]any)
		if !ok {
			return nil
		}
		for index, child := range array {
			if err := validateSchemaValue(items, child, fmt.Sprintf("%s[%d]", path, index), false); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchesJSONType(typeName string, value any) bool {
	switch strings.TrimSpace(typeName) {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		return isJSONNumber(value)
	case "integer":
		return isJSONInteger(value)
	case "null":
		return value == nil
	default:
		return true
	}
}

func isJSONNumber(value any) bool {
	switch value.(type) {
	case json.Number, float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isJSONInteger(value any) bool {
	switch number := value.(type) {
	case json.Number:
		_, err := number.Int64()
		return err == nil
	case float32:
		return float32(math.Trunc(float64(number))) == number
	case float64:
		return math.Trunc(number) == number
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func schemaMap(value any) map[string]map[string]any {
	result := map[string]map[string]any{}
	switch object := value.(type) {
	case map[string]any:
		for name, raw := range object {
			if schema, ok := raw.(map[string]any); ok {
				result[name] = schema
			}
		}
	case map[string]map[string]any:
		return object
	}
	return result
}

func stringSlice(value any) []string {
	switch values := value.(type) {
	case []string:
		return values
	case []any:
		result := make([]string, 0, len(values))
		for _, raw := range values {
			if item, ok := raw.(string); ok && strings.TrimSpace(item) != "" {
				result = append(result, item)
			}
		}
		return result
	default:
		return nil
	}
}
