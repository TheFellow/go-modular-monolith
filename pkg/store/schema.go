package store

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
)

const timestampLayout = "2006-01-02T15:04:05.000000000Z"

type column struct {
	name, kind string
	boolean    bool
}

// shape maps a Go value to typed columns or an owned child table. No document
// encoding is used; structs flatten and every collection gets foreign keys.
type shape struct {
	typ    reflect.Type
	kind   string
	fields []shapeField
	elem   *shape
	table  *tableSchema
}
type shapeField struct {
	index int
	node  *shape
}
type tableSchema struct {
	name     string
	columns  []column
	root     *shape
	key      *shape
	parent   *tableSchema
	primary  string
	children []*tableSchema
}

func snake(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if unicode.IsUpper(c) && i > 0 && (unicode.IsLower(r[i-1]) || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
func joined(a, b string) string {
	if a == "" {
		return b
	}
	return a + "_" + b
}
func buildSchema(t reflect.Type) (*tableSchema, error) {
	if t.Kind() != reflect.Struct || t.NumField() == 0 {
		return nil, fmt.Errorf("store model must be a non-empty struct")
	}
	table := &tableSchema{name: modelName(t), primary: snake(t.Field(0).Name)}
	n, err := buildShape(t, "", table, true)
	if err != nil {
		return nil, err
	}
	table.root = n
	if len(table.columns) == 0 || table.columns[0].name != table.primary {
		return nil, fmt.Errorf("store primary key must be scalar")
	}
	return table, nil
}
func buildShape(t reflect.Type, name string, table *tableSchema, root bool) (*shape, error) {
	n := &shape{typ: t}
	add := func(name, kind string) {
		table.columns = append(table.columns, column{name: name, kind: kind, boolean: strings.HasSuffix(name, "__present") || t.Kind() == reflect.Bool})
	}
	if t == reflect.TypeFor[time.Time]() {
		n.kind = "time"
		add(name, "TEXT")
		return n, nil
	}
	if t.Implements(reflect.TypeFor[encoding.TextMarshaler]()) && reflect.PointerTo(t).Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) {
		n.kind = "text"
		add(name, "TEXT")
		return n, nil
	}
	if m, ok := t.MethodByName("Unwrap"); ok && m.Type.NumOut() == 2 && m.Type.Out(1).Kind() == reflect.Bool {
		if _, ok := reflect.PointerTo(t).MethodByName("Set"); !ok {
			return nil, fmt.Errorf("optional %s requires Set", t)
		}
		n.kind = "optional"
		add(name+"__present", "INTEGER")
		var err error
		n.elem, err = buildShape(m.Type.Out(0), wrappedName(name, m.Type.Out(0)), table, false)
		return n, err
	}
	//exhaustive:ignore Unsupported reflection kinds are rejected by the default case.
	switch t.Kind() {
	case reflect.Pointer:
		n.kind = "pointer"
		add(name+"__present", "INTEGER")
		var err error
		n.elem, err = buildShape(t.Elem(), wrappedName(name, t.Elem()), table, false)
		return n, err
	case reflect.Struct:
		n.kind = "struct"
		for i := range t.NumField() {
			f := t.Field(i)
			if f.Tag.Get("store") == "revision" && root {
				continue
			}
			if !f.IsExported() {
				return nil, fmt.Errorf("unsupported private field %s.%s; use a scalar text codec", t, f.Name)
			}
			child, err := buildShape(f.Type, joined(name, snake(f.Name)), table, false)
			if err != nil {
				return nil, err
			}
			n.fields = append(n.fields, shapeField{i, child})
		}
	case reflect.Slice, reflect.Map:
		n.kind = t.Kind().String()
		add(name+"__present", "INTEGER")
		child := &tableSchema{name: joined(table.name, name), parent: table, primary: "__row_id"}
		n.table = child
		table.children = append(table.children, child)
		var err error
		if t.Kind() == reflect.Map {
			child.key, err = buildShape(t.Key(), "key", child, false)
			if err != nil {
				return nil, err
			}
			if child.key.kind != "scalar" && child.key.kind != "text" && child.key.kind != "time" {
				return nil, fmt.Errorf("unsupported map key %s", t.Key())
			}
		}
		elemName := "value"
		_, optionalElement := t.Elem().MethodByName("Unwrap")
		if t.Elem().Kind() == reflect.Struct && t.Elem() != reflect.TypeFor[time.Time]() && !t.Elem().Implements(reflect.TypeFor[encoding.TextMarshaler]()) && !optionalElement {
			elemName = ""
		}
		child.root, err = buildShape(t.Elem(), elemName, child, false)
		if err != nil {
			return nil, err
		}
	case reflect.String:
		n.kind = "scalar"
		add(name, "TEXT")
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n.kind = "scalar"
		add(name, "INTEGER")
	case reflect.Float32, reflect.Float64:
		n.kind = "scalar"
		add(name, "REAL")
	default:
		return nil, fmt.Errorf("unsupported store field %s (%s)", name, t)
	}
	return n, nil
}
func (s *tableSchema) create(ctx context.Context, db sqlExecutor) error {
	defs := []string{}
	if s.parent == nil {
		defs = append(defs, `"__revision" INTEGER NOT NULL DEFAULT 1 CHECK("__revision" > 0)`)
	} else {
		defs = append(defs, `"__row_id" INTEGER PRIMARY KEY`, safeName("__parent_id")+" "+s.parentKeyKind()+" NOT NULL REFERENCES "+safeName(s.parent.name)+"("+safeName(s.parent.primary)+") ON DELETE CASCADE", `"__position" INTEGER NOT NULL CHECK("__position" >= 0)`)
	}
	for _, c := range s.columns {
		d := safeName(c.name) + " " + c.kind + " NOT NULL"
		if c.boolean {
			d += " CHECK(" + safeName(c.name) + " IN (0,1))"
		}
		if s.parent == nil && c.name == s.primary {
			d += " PRIMARY KEY"
		}
		defs = append(defs, d)
	}
	if s.parent != nil {
		defs = append(defs, `UNIQUE("__parent_id", "__position")`)
		if s.key != nil {
			defs = append(defs, `UNIQUE("__parent_id", "key")`)
		}
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS "+safeName(s.name)+" ("+strings.Join(defs, ",")+") STRICT"); err != nil {
		return err
	}
	for _, child := range s.children {
		if err := child.create(ctx, db); err != nil {
			return err
		}
	}
	return nil
}
func (s *tableSchema) selection() string {
	names := make([]string, 0, len(s.columns)+1)
	if s.parent == nil {
		names = append(names, safeName("__revision"))
	} else {
		names = append(names, safeName("__row_id"))
	}
	for _, c := range s.columns {
		names = append(names, safeName(c.name))
	}
	return strings.Join(names, ",")
}

// Collection writes and reads run after the parent row. Optional reconstruction
// is deferred until its owned children have loaded, so copying the value is safe.
type collectionValue struct {
	table  *tableSchema
	value  reflect.Value
	finish func()
}

func (n *shape) encode(v reflect.Value, values *[]any, children *[]collectionValue) error {
	switch n.kind {
	case "struct":
		for _, f := range n.fields {
			if err := f.node.encode(v.Field(f.index), values, children); err != nil {
				return err
			}
		}
	case "optional":
		out := v.MethodByName("Unwrap").Call(nil)
		*values = append(*values, out[1].Bool())
		return n.elem.encode(out[0], values, children)
	case "pointer":
		*values = append(*values, !v.IsNil())
		if v.IsNil() {
			v = reflect.Zero(v.Type().Elem())
		} else {
			v = v.Elem()
		}
		return n.elem.encode(v, values, children)
	case "slice", "map":
		*values = append(*values, !v.IsNil())
		*children = append(*children, collectionValue{table: n.table, value: v})
	case "time":
		*values = append(*values, v.Interface().(time.Time).UTC().Format(timestampLayout))
	case "text":
		text, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		if err != nil {
			return err
		}
		*values = append(*values, string(text))
	case "scalar":
		*values = append(*values, v.Interface())
	}
	return nil
}
func scanRaw(rows interface{ Scan(...any) error }, count int) ([]any, error) {
	raw := make([]any, count)
	dest := make([]any, count)
	for i := range raw {
		dest[i] = &raw[i]
	}
	err := rows.Scan(dest...)
	return raw, err
}
func (n *shape) decode(v reflect.Value, raw []any, pos *int, children *[]collectionValue) error {
	next := func() any {
		value := raw[*pos]
		(*pos)++
		return value
	}
	switch n.kind {
	case "struct":
		for _, f := range n.fields {
			if err := f.node.decode(v.Field(f.index), raw, pos, children); err != nil {
				return err
			}
		}
	case "pointer", "optional":
		present := next().(int64) != 0
		value := reflect.New(n.elem.typ).Elem()
		if err := n.elem.decode(value, raw, pos, children); err != nil {
			return err
		}
		if present {
			if n.kind == "pointer" {
				v.Set(value.Addr())
			} else {
				*children = append(*children, collectionValue{finish: func() { v.Addr().MethodByName("Set").Call([]reflect.Value{value}) }})
			}
		}
	case "slice", "map":
		present := next().(int64) != 0
		if present {
			if n.kind == "slice" {
				v.Set(reflect.MakeSlice(v.Type(), 0, 0))
			} else {
				v.Set(reflect.MakeMap(v.Type()))
			}
			*children = append(*children, collectionValue{table: n.table, value: v})
		}
	case "time":
		value, err := time.Parse(timestampLayout, next().(string))
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(value))
	case "text":
		return v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(next().(string)))
	case "scalar":
		value := next()
		//exhaustive:ignore Schema compilation restricts scalar kinds to these cases.
		switch v.Kind() {
		case reflect.String:
			v.SetString(value.(string))
		case reflect.Bool:
			v.SetBool(value.(int64) != 0)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v.SetInt(value.(int64))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v.SetUint(uint64(value.(int64)))
		case reflect.Float32, reflect.Float64:
			if f, ok := value.(float64); ok {
				v.SetFloat(f)
			} else {
				v.SetFloat(float64(value.(int64)))
			}
		}
	}
	return nil
}
func (t *Tx) saveChildren(parent any, children []collectionValue) error {
	for _, child := range children {
		table := child.table
		if _, err := t.tx.ExecContext(t.ctx, "DELETE FROM "+safeName(table.name)+" WHERE __parent_id=?", parent); err != nil {
			return err
		}
		keys := []reflect.Value(nil)
		if child.value.Kind() == reflect.Map {
			keys = child.value.MapKeys()
		}
		for i := 0; i < child.value.Len(); i++ {
			values := []any{parent, i}
			nested := []collectionValue{}
			var elem reflect.Value
			if child.value.Kind() == reflect.Map {
				if err := table.key.encode(keys[i], &values, &nested); err != nil {
					return err
				}
				elem = child.value.MapIndex(keys[i])
			} else {
				elem = child.value.Index(i)
			}
			if err := table.root.encode(elem, &values, &nested); err != nil {
				return err
			}
			names := []string{safeName("__parent_id"), safeName("__position")}
			for _, c := range table.columns {
				names = append(names, safeName(c.name))
			}
			result, err := t.tx.ExecContext(t.ctx, "INSERT INTO "+safeName(table.name)+" ("+strings.Join(names, ",")+") VALUES ("+placeholders(len(values))+")", values...)
			if err != nil {
				return err
			}
			id, err := result.LastInsertId()
			if err != nil {
				return err
			}
			if err := t.saveChildren(id, nested); err != nil {
				return err
			}
		}
	}
	return nil
}
func (t *Tx) loadChildren(parent any, children []collectionValue) error {
	for _, child := range children {
		if child.finish != nil {
			child.finish()
			continue
		}
		table := child.table
		rows, err := t.tx.QueryContext(t.ctx, "SELECT "+table.selection()+" FROM "+safeName(table.name)+" WHERE __parent_id=? ORDER BY __position", parent)
		if err != nil {
			return err
		}
		all := [][]any{}
		for rows.Next() {
			raw, err := scanRaw(rows, len(table.columns)+1)
			if err != nil {
				_ = rows.Close()
				return err
			}
			all = append(all, raw)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		for _, raw := range all {
			pos := 1
			nested := []collectionValue{}
			var key reflect.Value
			if table.key != nil {
				key = reflect.New(table.key.typ).Elem()
				if err := table.key.decode(key, raw, &pos, &nested); err != nil {
					return err
				}
			}
			elem := reflect.New(table.root.typ).Elem()
			if err := table.root.decode(elem, raw, &pos, &nested); err != nil {
				return err
			}
			if err := t.loadChildren(raw[0], nested); err != nil {
				return err
			}
			if table.key != nil {
				child.value.SetMapIndex(key, elem)
			} else {
				child.value.Set(reflect.Append(child.value, elem))
			}
		}
	}
	return nil
}
func placeholders(n int) string { return strings.TrimSuffix(strings.Repeat("?,", n), ",") }
func (t *Tx) schema(typ reflect.Type) (*tableSchema, error) {
	v, ok := t.store.models.Load(typ)
	if !ok {
		return nil, fmt.Errorf("store model %s is not registered", typ)
	}
	return v.(*tableSchema), nil
}

func (s *tableSchema) parentKeyKind() string {
	if s.parent.parent != nil {
		return "INTEGER"
	}
	for _, c := range s.parent.columns {
		if c.name == s.parent.primary {
			return c.kind
		}
	}
	return "TEXT"
}

// Consecutive wrappers each need an independent presence column. Ordinary
// optional structs retain the same flattened field names as required structs.
func wrappedName(name string, typ reflect.Type) string {
	if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
		return name + "__value"
	}
	if _, ok := typ.MethodByName("Unwrap"); ok {
		return name + "__value"
	}
	return name
}
