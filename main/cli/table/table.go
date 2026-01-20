package table

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintTable prints items as a table with headers from struct tags.
func PrintTable[T any](items []T) error {
	itemType := reflect.TypeOf(items)
	if itemType == nil || itemType.Kind() != reflect.Slice {
		return fmt.Errorf("table: items must be a slice")
	}

	elemType := itemType.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("table: items must be a slice of structs")
	}

	fields := getTableFields(elemType)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	headers := make([]string, len(fields))
	for i, f := range fields {
		headers[i] = f.header
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	for _, item := range items {
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return fmt.Errorf("table: nil item")
			}
			val = val.Elem()
		}
		if val.Kind() != reflect.Struct {
			return fmt.Errorf("table: item must be a struct")
		}

		values := make([]string, len(fields))
		for i, f := range fields {
			values[i] = formatValue(val.Field(f.index))
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	return w.Flush()
}

// PrintDetail prints a single item as key-value pairs.
func PrintDetail[T any](item T) error {
	val := reflect.ValueOf(item)
	if !val.IsValid() {
		return fmt.Errorf("table: invalid item")
	}
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return fmt.Errorf("table: nil item")
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("table: item must be a struct")
	}

	fields := getJSONFields(val.Type())
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, f := range fields {
		fieldVal := val.Field(f.index)
		if f.omitempty && isZero(fieldVal) {
			continue
		}
		fmt.Fprintf(w, "%s:\t%s\n", f.label, formatValue(fieldVal))
	}
	return w.Flush()
}

type tableField struct {
	index  int
	header string
}

type jsonField struct {
	index     int
	label     string
	omitempty bool
}

func getTableFields(typ reflect.Type) []tableField {
	fields := make([]tableField, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("table")
		if tag == "" || tag == "-" {
			continue
		}
		fields = append(fields, tableField{index: i, header: tag})
	}
	return fields
}

func getJSONFields(typ reflect.Type) []jsonField {
	fields := make([]jsonField, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		f := jsonField{index: i, label: toLabel(parts[0])}
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				f.omitempty = true
			}
		}
		fields = append(fields, f)
	}
	return fields
}

func toLabel(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	for i, part := range parts {
		low := strings.ToLower(part)
		if v, ok := initialisms[low]; ok {
			parts[i] = v
			continue
		}
		if low == "" {
			continue
		}
		parts[i] = strings.ToUpper(low[:1]) + low[1:]
	}
	return strings.Join(parts, " ")
}

var initialisms = map[string]string{
	"id":  "ID",
	"uid": "UID",
	"url": "URL",
}

func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		return formatValue(v.Elem())
	}
	if v.CanInterface() {
		if t, ok := v.Interface().(time.Time); ok {
			if t.IsZero() {
				return ""
			}
			return t.Format(time.RFC3339)
		}
		if stringer, ok := v.Interface().(fmt.Stringer); ok {
			return stringer.String()
		}
	}
	return fmt.Sprintf("%v", v.Interface())
}

func isZero(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}
		return isZero(v.Elem())
	}
	return v.IsZero()
}
