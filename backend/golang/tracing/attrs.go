package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"path"
	"reflect"

	"github.com/iancoleman/strcase"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Attributed defines an interface for objects that provide trace attributes.
type Attributed interface {
	Attributes() []attribute.KeyValue
}

// TraceValue adds a single attribute to the current span.
func TraceValue(ctx context.Context, name string, val interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	av, ok := attributeValue(reflect.ValueOf(val))
	if ok {
		span.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key(name),
			Value: av,
		})
	}
}

// TraceAny recursively adds all exported struct fields to the span attributes.
func TraceAny(ctx context.Context, prefix string, obj interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}

	if attributed, ok := obj.(Attributed); ok {
		span.SetAttributes(attributed.Attributes()...)
	} else {
		span.SetAttributes(attributesFrom(prefix, obj)...)
	}
}

// Error records an error and updates the span status.
func Error(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AttributesFrom converts an object to a set of attributes.
// It recursively processes exported struct fields and adds them as attributes.
// The prefix parameter is used to create the attribute names.
func AttributesFrom(prefix string, obj interface{}) []attribute.KeyValue {
	// If the prefix is not empty, add an underscore to separate it from the field names.
	if prefix != "" {
		prefix += "_"
	}

	return attributesFrom(prefix, obj)
}

// --- Helpers ---

const (
	tagName = "trace"
	dot     = '.'
)

func attributesFrom(prefix string, obj interface{}) []attribute.KeyValue {
	if obj == nil {
		return nil
	}

	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}

	if isUnsupported(rv) {
		return nil
	}

	rt := rv.Type()
	attrs := make([]attribute.KeyValue, 0, rt.NumField())
	prefixBytes, buf := prefixAndBuffer(rt)

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		tagVal := field.Tag.Get(tagName)
		if tagVal == "-" {
			continue
		}

		fieldName := field.Name
		if tagVal != "" {
			fieldName = tagVal
		}

		fieldVal := rv.Field(i)
		if av, ok := attributeValue(fieldVal); ok {
			buf.Reset()
			buf.Write(prefixBytes)
			buf.WriteString(strcase.ToSnake(fieldName))

			key := attribute.Key(prefix + buf.String())
			attrs = append(attrs, attribute.KeyValue{Key: key, Value: av})
		}
	}

	return attrs
}

func attributeValue(v reflect.Value) (av attribute.Value, ok bool) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return attribute.Value{}, false
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return attribute.StringValue(v.String()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return attribute.Int64Value(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return attribute.Int64Value(int64(v.Uint())), true
	case reflect.Float32, reflect.Float64:
		return attribute.Float64Value(v.Float()), true
	case reflect.Bool:
		return attribute.BoolValue(v.Bool()), true
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		data, _ := json.Marshal(v.Interface())
		return attribute.StringValue(string(data)), true
	default:
		return attribute.Value{}, false
	}
}

func prefixAndBuffer(rt reflect.Type) ([]byte, bytes.Buffer) {
	var buf bytes.Buffer
	if sf, ok := rt.FieldByName("_"); ok {
		if tagVal := sf.Tag.Get(tagName); tagVal != "" {
			buf.WriteString(tagVal)
			buf.WriteByte(dot)
		}
	}

	if buf.Len() == 0 {
		pkgPath := path.Base(rt.PkgPath())
		buf.WriteString(pkgPath)
		buf.WriteByte(dot)
		buf.WriteString(strcase.ToSnake(rt.Name()))
		buf.WriteByte(dot)
	}

	return buf.Bytes(), buf
}

func isUnsupported(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
		return false
	default:
		return true
	}
}
