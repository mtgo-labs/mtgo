package telegram

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

const tdlibTypeField = "@type"

var (
	tdlibNameMap     map[string]string
	tdlibNameMapOnce sync.Once
)

func buildTDLibNameMap() map[string]string {
	tdlibNameMapOnce.Do(func() {
		m := make(map[string]string, len(tg.NamesMap)*2)
		for fullName := range tg.NamesMap {
			dot := strings.LastIndex(fullName, ".")
			if dot >= 0 {
				short := fullName[dot+1:]
				if existing, ok := m[short]; ok && existing != fullName {
					m[fullName] = fullName
					continue
				}
				m[short] = fullName
			}
			m[fullName] = fullName
		}
		tdlibNameMap = m
	})
	return tdlibNameMap
}

func resolveTDLibTypeName(typeName string) (string, uint32, bool) {
	m := buildTDLibNameMap()
	fullName, ok := m[typeName]
	if !ok {
		return "", 0, false
	}
	id, ok := tg.NamesMap[fullName]
	if !ok {
		return "", 0, false
	}
	return fullName, id, true
}

func findFactoryForID(id uint32) (func() tg.TLObject, bool) {
	if f, ok := tg.FunctionsMap[id]; ok {
		return f, true
	}
	if f, ok := tg.ConstructorMap[id]; ok {
		return f, true
	}
	return nil, false
}

func tdlibExtractInterfaces(raw map[string]interface{}) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, 0)
	for key, val := range raw {
		if key == tdlibTypeField || key == "_" {
			continue
		}
		nested, ok := val.(map[string]interface{})
		if ok {
			typeName, hasType := nested[tdlibTypeField]
			if !hasType {
				typeName, hasType = nested["_"]
			}
			if hasType {
				typeNameStr, ok2 := typeName.(string)
				if ok2 {
					if _, _, found := resolveTDLibTypeName(typeNameStr); found {
						b, err := json.Marshal(val)
						if err == nil {
							result[key] = b
							delete(raw, key)
						}
					}
				}
			}
			continue
		}
		arr, isArr := val.([]interface{})
		if !isArr || len(arr) == 0 {
			continue
		}
		firstObj, isObj := arr[0].(map[string]interface{})
		if !isObj {
			continue
		}
		if _, hasType := firstObj[tdlibTypeField]; !hasType {
			if _, hasType := firstObj["_"]; !hasType {
				continue
			}
		}
		b, err := json.Marshal(val)
		if err != nil {
			continue
		}
		result[key] = b
		delete(raw, key)
	}
	return result
}

func tdlibSetInterfaceFields(req tg.TLObject, ifaceJSON map[string]json.RawMessage) {
	v := reflect.ValueOf(req).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		tagName := strings.Split(jsonTag, ",")[0]
		rawMsg, ok := ifaceJSON[tagName]
		if !ok {
			continue
		}
		if field.Kind() == reflect.Slice {
			var rawSlice []json.RawMessage
			if err := json.Unmarshal(rawMsg, &rawSlice); err != nil {
				continue
			}
			sliceVal := reflect.MakeSlice(field.Type(), 0, len(rawSlice))
			for _, rawItem := range rawSlice {
				obj, err := tdlibDecodeInterfaceObject(rawItem)
				if err != nil {
					continue
				}
				sliceVal = reflect.Append(sliceVal, reflect.ValueOf(obj))
			}
			field.Set(sliceVal)
			continue
		}
		if field.Kind() != reflect.Interface {
			continue
		}
		obj, err := tdlibDecodeInterfaceObject(rawMsg)
		if err != nil {
			continue
		}
		field.Set(reflect.ValueOf(obj))
	}
}

func tdlibDecodeInterfaceObject(raw json.RawMessage) (tg.TLObject, error) {
	var tmp map[string]interface{}
	if err := json.Unmarshal(raw, &tmp); err != nil {
		return nil, err
	}
	typeName, _ := tmp[tdlibTypeField].(string)
	if typeName == "" {
		typeName, _ = tmp["_"].(string)
	}
	if typeName == "" {
		return nil, fmt.Errorf("missing @type or _ field")
	}
	_, id, ok := resolveTDLibTypeName(typeName)
	if !ok {
		return nil, fmt.Errorf("unknown type %q", typeName)
	}
	factory, ok := findFactoryForID(id)
	if !ok {
		return nil, fmt.Errorf("no factory for type %q", typeName)
	}
	obj := factory()
	cleaned := tdlibNormalizeInput(tmp)
	cleanedBytes, err := json.Marshal(cleaned)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cleanedBytes, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func tdlibNormalizeInput(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		if k == tdlibTypeField {
			k = "_"
		}
		switch val := v.(type) {
		case string:
			if n, err := strconv.ParseInt(val, 10, 64); err == nil && len(val) > 0 && val[0] != '-' && !(len(val) > 1 && val[0] == '0') {
				out[k] = n
				continue
			}
			if b, err := base64.RawStdEncoding.DecodeString(val); err == nil && len(b) > 0 && len(val) >= 2 {
				if isLikelyBinary(val) {
					out[k] = string(b)
					continue
				}
			}
			out[k] = val
		case map[string]interface{}:
			out[k] = tdlibNormalizeInput(val)
		case []interface{}:
			normalized := make([]interface{}, len(val))
			for i, item := range val {
				if sub, ok := item.(map[string]interface{}); ok {
					normalized[i] = tdlibNormalizeInput(sub)
				} else {
					normalized[i] = item
				}
			}
			out[k] = normalized
		default:
			out[k] = v
		}
	}
	return out
}

func isLikelyBinary(s string) bool {
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' || c == '-') {
			return false
		}
	}
	return len(s) > 4
}

func tdlibEncodeResponse(obj tg.TLObject) ([]byte, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	idToName := getIDToName()
	tlName, hasName := idToName[obj.ConstructorID()]
	if hasName {
		raw[tdlibTypeField] = tlName
	}
	delete(raw, "_")
	tdlibEncodeValues(raw)
	return json.Marshal(raw)
}

func tdlibEncodeValues(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case int64:
			m[k] = strconv.FormatInt(val, 10)
		case uint64:
			m[k] = strconv.FormatUint(val, 10)
		case float64:
			if jsonNum, ok := isJSONInt(val); ok {
				m[k] = strconv.FormatInt(jsonNum, 10)
			}
		case map[string]interface{}:
			tdlibEncodeValues(val)
		case []interface{}:
			for i, item := range val {
				if sub, ok := item.(map[string]interface{}); ok {
					tdlibEncodeValues(sub)
					val[i] = sub
				}
			}
		}
	}
}

func isJSONInt(f float64) (int64, bool) {
	if f == float64(int64(f)) && f > 1<<53 {
		return int64(f), true
	}
	return 0, false
}

func tdlibEncodeError(code int, message string) []byte {
	result, _ := json.Marshal(map[string]interface{}{
		tdlibTypeField: "error",
		"code":         code,
		"message":      message,
	})
	return result
}

// InvokeRawJSON invokes a Telegram MTProto method using TDLib JSON format.
// Input uses "@type" as the type discriminator and supports short method names
// (e.g., "sendMessage" resolves to "messages.sendMessage").
// int64 fields are accepted as quoted strings, and bytes fields as base64.
// Responses use "@type" and quoted int64 per TDLib convention.
// RPC errors are returned as {"@type":"error","code":...,"message":"..."}.
func (c *Client) InvokeRawJSON(ctx context.Context, jsonRequest []byte) ([]byte, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonRequest, &raw); err != nil {
		return tdlibEncodeError(400, fmt.Sprintf("invalid JSON: %v", err)), nil
	}

	typeVal, _ := raw[tdlibTypeField].(string)
	if typeVal == "" {
		typeVal, _ = raw["_"].(string)
	}
	if typeVal == "" {
		return tdlibEncodeError(400, "missing @type field"), nil
	}

	fullName, _, ok := resolveTDLibTypeName(typeVal)
	if !ok {
		return tdlibEncodeError(400, fmt.Sprintf("unknown method %q", typeVal)), nil
	}

	id, ok := tg.NamesMap[fullName]
	if !ok {
		return tdlibEncodeError(400, fmt.Sprintf("type not found: %q", fullName)), nil
	}

	factory, ok := findFactoryForID(id)
	if !ok {
		return tdlibEncodeError(400, fmt.Sprintf("no factory for %q", fullName)), nil
	}

	req := factory()

	delete(raw, tdlibTypeField)
	delete(raw, "_")

	ifaceJSON := tdlibExtractInterfaces(raw)

	cleaned := tdlibNormalizeInput(raw)
	cleanedBytes, err := json.Marshal(cleaned)
	if err != nil {
		return tdlibEncodeError(400, fmt.Sprintf("marshal input: %v", err)), nil
	}

	if err := json.Unmarshal(cleanedBytes, req); err != nil {
		return tdlibEncodeError(400, fmt.Sprintf("unmarshal into %s: %v", fullName, err)), nil
	}

	tdlibSetInterfaceFields(req, ifaceJSON)

	result, err := NewJSONClient(c.Raw()).invokeRaw(ctx, req)
	if err != nil {
		if rpcErr, ok := tgerr.As(err); ok {
			return tdlibEncodeError(rpcErr.Code, rpcErr.Message), nil
		}
		return nil, err
	}

	return tdlibEncodeResponse(result)
}

func (jc *JSONClient) invokeRaw(ctx context.Context, req tg.TLObject) (tg.TLObject, error) {
	return jc.rpc.Invoke(ctx, req, func(r *tg.Reader) (tg.TLObject, error) {
		return tg.ReadTLObject(r)
	})
}

// TDLibShortNames returns a sorted list of all supported short method names
// (e.g., "sendMessage") for discovery and documentation.
func (c *Client) TDLibShortNames() []string {
	m := buildTDLibNameMap()
	names := make(map[string]bool, len(m))
	for k, v := range m {
		if k == v {
			continue
		}
		if !strings.Contains(k, ".") {
			names[k] = true
		}
	}
	result := make([]string, 0, len(names))
	for n := range names {
		result = append(result, n)
	}
	sort.Strings(result)
	return result
}
