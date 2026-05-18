package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
)

// JSONClient wraps an RPC client and provides JSON-based invocation of Telegram TL functions.
// It translates between JSON payloads and native TL objects, handling interface field
// deserialization and optional snake_case key conversion. Use this to build proxy servers
// or HTTP gateways that accept JSON requests and forward them to the Telegram API.
//
// Example:
//
//	rpc := tg.NewRPCClient(conn)
//	jc := telegram.NewJSONClient(rpc)
//	result, err := jc.InvokeJSON(ctx, "messages.sendMessage", payload, false)
type JSONClient struct {
	rpc *tg.RPCClient
}

// NewJSONClient creates a new JSONClient backed by the given RPC client.
// The RPC client must be connected and authenticated before invoking TL functions.
//
// Example:
//
//	rpc := tg.NewRPCClient(conn)
//	jc := telegram.NewJSONClient(rpc)
func NewJSONClient(rpc *tg.RPCClient) *JSONClient {
	return &JSONClient{rpc: rpc}
}

var (
	idToName    map[uint32]string
	idToNameOnce sync.Once
)

func getIDToName() map[uint32]string {
	idToNameOnce.Do(func() {
		m := make(map[uint32]string, len(tg.NamesMap))
		for name, id := range tg.NamesMap {
			m[id] = name
		}
		idToName = m
	})
	return idToName
}

func findRequestStruct(id uint32) (tg.TLObject, error) {
	if factory, ok := tg.FunctionsMap[id]; ok {
		return factory(), nil
	}
	if factory, ok := tg.ConstructorMap[id]; ok {
		return factory(), nil
	}
	return nil, fmt.Errorf("telegram: no factory for constructor ID 0x%08x", id)
}

func findFactory(typeName string) (func() tg.TLObject, bool) {
	id, found := tg.NamesMap[typeName]
	if !found {
		return nil, false
	}
	if f, ok := tg.FunctionsMap[id]; ok {
		return f, true
	}
	if f, ok := tg.ConstructorMap[id]; ok {
		return f, true
	}
	return nil, false
}

func extractInterfaceJSON(raw map[string]interface{}) map[string]json.RawMessage {
	result := map[string]json.RawMessage{}
	for key, val := range raw {
		if key == "_" {
			continue
		}
		nested, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		typeName, hasType := nested["_"]
		if !hasType {
			continue
		}
		typeNameStr, ok := typeName.(string)
		if !ok {
			continue
		}
		if _, found := findFactory(typeNameStr); found {
			b, err := json.Marshal(val)
			if err != nil {
				continue
			}
			result[key] = b
			delete(raw, key)
		}
	}
	return result
}

func setInterfaceFields(req tg.TLObject, ifaceJSON map[string]json.RawMessage) {
	v := reflect.ValueOf(req).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() != reflect.Interface {
			continue
		}

		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		tagName := strings.Split(jsonTag, ",")[0]

		rawMsg, ok := ifaceJSON[tagName]
		if !ok {
			continue
		}

		var tmp map[string]interface{}
		if err := json.Unmarshal(rawMsg, &tmp); err != nil {
			continue
		}
		typeName, hasType := tmp["_"]
		if !hasType {
			continue
		}
		typeNameStr, ok := typeName.(string)
		if !ok {
			continue
		}
		factory, found := findFactory(typeNameStr)
		if !found {
			continue
		}
		obj := factory()
		if err := json.Unmarshal(rawMsg, obj); err != nil {
			continue
		}
		field.Set(reflect.ValueOf(obj))
	}
}

func camelOrPascalToSnake(s string) string {
	// TL field names are ASCII-only; iterating over bytes is valid and
	// avoids allocating a []rune slice.
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i > 0 && c >= 'A' && c <= 'Z' {
			prev := s[i-1]
			nextLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
			prevLower := prev >= 'a' && prev <= 'z'
			if prevLower || nextLower {
				buf.WriteByte('_')
			}
		}
		if c >= 'A' && c <= 'Z' {
			buf.WriteByte(c + 32)
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func convertKeysToSnakeCase(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, v := range val {
			result[camelOrPascalToSnake(k)] = convertKeysToSnakeCase(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = convertKeysToSnakeCase(item)
		}
		return result
	default:
		return v
	}
}

func encodeResultToJSON(obj tg.TLObject, useSnakeCase bool) ([]byte, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("telegram: marshal result: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("telegram: unmarshal result: %w", err)
	}

	tlName, hasName := getIDToName()[obj.ConstructorID()]
	if hasName {
		if useSnakeCase {
			raw["_"] = camelOrPascalToSnake(tlName)
		} else {
			raw["_"] = tlName
		}
	}

	if useSnakeCase {
		converted := convertKeysToSnakeCase(raw)
		return json.Marshal(converted)
	}

	return json.Marshal(raw)
}

// InvokeJSON invokes a Telegram TL function by name with a JSON payload and returns the result as JSON.
// The functionName must match a registered TL function name (e.g. "messages.sendMessage").
// The payload is a JSON byte slice that maps to the function's request struct fields.
// When useSnakeCase is true, both input and output keys are converted from PascalCase
// to snake_case for compatibility with JSON APIs.
//
// Example:
//
//	payload := []byte(`{"peer": {"_": "inputPeerUser", "user_id": 123, "access_hash": 456}, "message": "Hello", "random_id": 789}`)
//	result, err := jc.InvokeJSON(ctx, "messages.sendMessage", payload, false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(string(result))
func (c *JSONClient) InvokeJSON(ctx context.Context, functionName string, payload []byte, useSnakeCase bool) ([]byte, error) {
	id, ok := tg.NamesMap[functionName]
	if !ok {
		return nil, fmt.Errorf("telegram: unknown TL function %q", functionName)
	}

	req, err := findRequestStruct(id)
	if err != nil {
		return nil, err
	}

	if len(payload) > 0 {
		var raw map[string]interface{}
		if err := json.Unmarshal(payload, &raw); err != nil {
			return nil, fmt.Errorf("telegram: invalid JSON payload: %w", err)
		}

		ifaceJSON := extractInterfaceJSON(raw)

		reMarshalled, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("telegram: re-marshal payload: %w", err)
		}

		if err := json.Unmarshal(reMarshalled, req); err != nil {
			return nil, fmt.Errorf("telegram: unmarshal into request: %w", err)
		}

		setInterfaceFields(req, ifaceJSON)
	}

	result, err := c.rpc.Invoke(ctx, req, func(r *tg.Reader) (tg.TLObject, error) {
		return tg.ReadTLObject(r)
	})
	if err != nil {
		return nil, err
	}

	return encodeResultToJSON(result, useSnakeCase)
}
