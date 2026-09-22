package store

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/TheFellow/go-modular-monolith/pkg/errors"
)

func rowInfo(value any) (reflect.Value, reflect.Type, reflect.Value, error) {
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return reflect.Value{}, nil, reflect.Value{}, errors.Invalidf("record must be a non-nil pointer")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct || v.NumField() == 0 {
		return reflect.Value{}, nil, reflect.Value{}, errors.Invalidf("record must point to a non-empty struct")
	}
	id := v.Field(0)
	return v, v.Type(), id, nil
}

func revisionField(v reflect.Value) (reflect.Value, bool) {
	t := v.Type()
	for i := range t.NumField() {
		if t.Field(i).Tag.Get("store") == "revision" {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func revisionValue(v reflect.Value) (uint64, bool, error) {
	field, ok := revisionField(v)
	if !ok {
		return 0, false, nil
	}
	if field.Kind() < reflect.Uint || field.Kind() > reflect.Uint64 {
		return 0, true, errors.Invalidf("record revision must be an unsigned integer")
	}
	return field.Uint(), true, nil
}

func setRevision(v reflect.Value, revision uint64) error {
	field, ok := revisionField(v)
	if !ok {
		return nil
	}
	if !field.CanSet() || field.Kind() < reflect.Uint || field.Kind() > reflect.Uint64 {
		return errors.Invalidf("record revision must be a settable unsigned integer")
	}
	field.SetUint(revision)
	return nil
}

func (t *Tx) Insert(values ...any) error {
	for _, value := range values {
		if err := t.insert(value); err != nil {
			return err
		}
	}
	return nil
}

// mutate makes each aggregate write atomic even when its caller handles an error.
func (t *Tx) mutate(fn func() error) error {
	if _, err := t.tx.ExecContext(t.ctx, "SAVEPOINT store_mutation"); err != nil {
		return err
	}
	if err := fn(); err != nil {
		_, _ = t.tx.ExecContext(context.WithoutCancel(t.ctx), "ROLLBACK TO store_mutation")
		_, _ = t.tx.ExecContext(context.WithoutCancel(t.ctx), "RELEASE store_mutation")
		if isUniqueConstraint(err) {
			return errors.Conflictf("unique constraint: %w", err)
		}
		return err
	}
	_, err := t.tx.ExecContext(t.ctx, "RELEASE store_mutation")
	return err
}
func (t *Tx) insert(value any) error {
	v, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	schema, err := t.schema(typ)
	if err != nil {
		return err
	}
	if revision, ok, err := revisionValue(v); err != nil {
		return err
	} else if ok && revision != 0 {
		return errors.Invalidf("new record revision must be zero")
	}
	err = t.mutate(func() error {
		if id.IsZero() && ((id.Kind() >= reflect.Uint && id.Kind() <= reflect.Uint64) || (id.Kind() >= reflect.Int && id.Kind() <= reflect.Int64)) {
			var maxID sql.NullInt64
			if err := t.tx.QueryRowContext(t.ctx, "SELECT MAX("+safeName(schema.primary)+") FROM "+safeName(schema.name)).Scan(&maxID); err != nil {
				return err
			}
			if id.Kind() >= reflect.Uint && id.Kind() <= reflect.Uint64 {
				id.SetUint(uint64(maxID.Int64 + 1))
			} else {
				id.SetInt(maxID.Int64 + 1)
			}
		}
		if id.IsZero() {
			return errors.Invalidf("record ID is required")
		}
		values := []any{}
		children := []collectionValue{}
		if err := schema.root.encode(v, &values, &children); err != nil {
			return err
		}
		names := []string{}
		for _, c := range schema.columns {
			names = append(names, safeName(c.name))
		}
		if _, err := t.tx.ExecContext(t.ctx, "INSERT INTO "+safeName(schema.name)+" ("+strings.Join(names, ",")+") VALUES ("+placeholders(len(values))+")", values...); err != nil {
			return err
		}
		return t.saveChildren(id.Interface(), children)
	})
	if err != nil {
		return err
	}
	return setRevision(v, 1)
}
func (t *Tx) Update(value any) error {
	v, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	schema, err := t.schema(typ)
	if err != nil {
		return err
	}
	if id.IsZero() {
		return errors.Invalidf("record ID is required")
	}
	revision, ok, err := revisionValue(v)
	if err != nil {
		return err
	}
	if !ok || revision == 0 {
		return errors.Invalidf("record revision is required for update")
	}
	err = t.mutate(func() error {
		values := []any{}
		children := []collectionValue{}
		if err := schema.root.encode(v, &values, &children); err != nil {
			return err
		}
		assigns := []string{`"__revision"="__revision"+1`}
		for _, c := range schema.columns {
			assigns = append(assigns, safeName(c.name)+"=?")
		}
		values = append(values, id.Interface(), revision)
		result, err := t.tx.ExecContext(t.ctx, "UPDATE "+safeName(schema.name)+" SET "+strings.Join(assigns, ",")+" WHERE "+safeName(schema.primary)+"=? AND __revision=?", values...)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if count == 0 {
			return t.stale(schema, id.Interface(), revision)
		}
		return t.saveChildren(id.Interface(), children)
	})
	if err != nil {
		return err
	}
	return setRevision(v, revision+1)
}
func (t *Tx) stale(schema *tableSchema, id any, revision uint64) error {
	var current uint64
	err := t.tx.QueryRowContext(t.ctx, "SELECT __revision FROM "+safeName(schema.name)+" WHERE "+safeName(schema.primary)+"=?", id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.NotFoundf("record absent")
	}
	if err != nil {
		return err
	}
	return errors.Conflictf("record changed: expected revision %d, current revision %d", revision, current)
}
func (t *Tx) Get(value any) error {
	v, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	schema, err := t.schema(typ)
	if err != nil {
		return err
	}
	if id.IsZero() {
		return errors.Invalidf("record ID is required")
	}
	raw, err := scanRaw(t.tx.QueryRowContext(t.ctx, "SELECT "+schema.selection()+" FROM "+safeName(schema.name)+" WHERE "+safeName(schema.primary)+"=?", id.Interface()), len(schema.columns)+1)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.NotFoundf("record absent")
	}
	if err != nil {
		return err
	}
	loaded := reflect.New(typ).Elem()
	if err := t.hydrate(schema, loaded, raw); err != nil {
		return err
	}
	v.Set(loaded)
	return nil
}
func (t *Tx) hydrate(schema *tableSchema, v reflect.Value, raw []any) error {
	pos := 1
	children := []collectionValue{}
	if err := schema.root.decode(v, raw, &pos, &children); err != nil {
		return err
	}
	if err := t.loadChildren(v.Field(0).Interface(), children); err != nil {
		return err
	}
	return setRevision(v, uint64(raw[0].(int64)))
}
func (t *Tx) Delete(value any) error {
	v, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	schema, err := t.schema(typ)
	if err != nil {
		return err
	}
	if id.IsZero() {
		return errors.Invalidf("record ID is required")
	}
	revision, ok, err := revisionValue(v)
	if err != nil {
		return err
	}
	if !ok || revision == 0 {
		return errors.Invalidf("record revision is required for delete")
	}
	result, err := t.tx.ExecContext(t.ctx, "DELETE FROM "+safeName(schema.name)+" WHERE "+safeName(schema.primary)+"=? AND __revision=?", id.Interface(), revision)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return t.stale(schema, id.Interface(), revision)
	}
	return nil
}

type predicate struct {
	field, op string
	values    []any
}
type ordering struct {
	field string
	desc  bool
}
type Query[T any] struct {
	tx         *Tx
	predicates []predicate
	residual   []func(T) bool
	order      []ordering
}

func QueryTx[T any](tx *Tx) *Query[T] { return &Query[T]{tx: tx} }
func (q *Query[T]) FilterEqual(f string, v ...any) *Query[T] {
	q.predicates = append(q.predicates, predicate{f, "=", v})
	return q
}
func (q *Query[T]) FilterNotEqual(f string, v ...any) *Query[T] {
	q.predicates = append(q.predicates, predicate{f, "!=", v})
	return q
}
func (q *Query[T]) FilterGreater(f string, v any) *Query[T] {
	q.predicates = append(q.predicates, predicate{f, ">", []any{v}})
	return q
}
func (q *Query[T]) FilterGreaterEqual(f string, v any) *Query[T] {
	q.predicates = append(q.predicates, predicate{f, ">=", []any{v}})
	return q
}
func (q *Query[T]) FilterLess(f string, v any) *Query[T] {
	q.predicates = append(q.predicates, predicate{f, "<", []any{v}})
	return q
}
func (q *Query[T]) FilterLessEqual(f string, v any) *Query[T] {
	q.predicates = append(q.predicates, predicate{f, "<=", []any{v}})
	return q
}
func (q *Query[T]) FilterID(v any) *Query[T] { return q.FilterEqual(firstField[T](), v) }
func (q *Query[T]) FilterIDs(v []string) *Query[T] {
	vals := make([]any, len(v))
	for i := range v {
		vals[i] = v[i]
	}
	return q.FilterEqual(firstField[T](), vals...)
}
func (q *Query[T]) FilterFn(fn func(T) bool) *Query[T] { q.residual = append(q.residual, fn); return q }
func (q *Query[T]) SortAsc(f ...string) *Query[T] {
	for _, x := range f {
		q.order = append(q.order, ordering{x, false})
	}
	return q
}
func (q *Query[T]) SortDesc(f ...string) *Query[T] {
	for _, x := range f {
		q.order = append(q.order, ordering{x, true})
	}
	return q
}

func firstField[T any]() string { var z T; return reflect.TypeOf(z).Field(0).Name }
func normalize(v any) any {
	if value, ok := asTime(v); ok {
		return value.UTC().Format(timestampLayout)
	}
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.String {
		return fmt.Sprint(v)
	}
	return v
}

func asTime(v any) (time.Time, bool) {
	rv := reflect.ValueOf(v)
	t := reflect.TypeFor[time.Time]()
	if rv.IsValid() && rv.Type().ConvertibleTo(t) {
		return rv.Convert(t).Interface().(time.Time), true
	}
	return time.Time{}, false
}

func queryFieldExpression(schema *tableSchema, name string) string {
	field, ok := schema.root.typ.FieldByName(name)
	if !ok {
		return ""
	}
	if field.Tag.Get("store") == "revision" {
		return safeName("__revision")
	}
	column := snake(name)
	for _, c := range schema.columns {
		if c.name == column {
			if field.Type.Kind() == reflect.Pointer {
				return "CASE WHEN " + safeName(column+"__present") + " THEN " + safeName(column) + " END"
			}
			if _, ok := field.Type.MethodByName("Unwrap"); ok {
				return "CASE WHEN " + safeName(column+"__present") + " THEN " + safeName(column) + " END"
			}
			return safeName(column)
		}
	}
	return ""
}

func (q *Query[T]) sql() (string, []any) {
	var z T
	typ := reflect.TypeOf(z)
	schema, err := q.tx.schema(typ)
	if err != nil {
		return "", nil
	}
	b := strings.Builder{}
	b.WriteString("SELECT " + schema.selection() + " FROM " + safeName(schema.name) + " WHERE 1=1")
	args := []any{}
	for _, p := range q.predicates {
		path := queryFieldExpression(schema, p.field)
		if path == "" {
			return "", nil
		}
		if len(p.values) == 0 {
			if p.op == "=" {
				b.WriteString(" AND 0")
			}
			continue
		}
		if len(p.values) > 1 {
			b.WriteString(" AND " + path)
			if p.op == "!=" {
				b.WriteString(" NOT")
			}
			b.WriteString(" IN (")
			for i, v := range p.values {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteByte('?')
				args = append(args, normalize(v))
			}
			b.WriteByte(')')
		} else if len(p.values) == 1 {
			placeholder := "?"

			b.WriteString(" AND " + path + " " + p.op + " " + placeholder)
			args = append(args, normalize(p.values[0]))
		}
	}
	if len(q.order) > 0 {
		b.WriteString(" ORDER BY ")
		for i, o := range q.order {
			if i > 0 {
				b.WriteByte(',')
			}
			expression := queryFieldExpression(schema, o.field)
			if expression == "" {
				return "", nil
			}
			b.WriteString(expression)
			if o.desc {
				b.WriteString(" DESC")
			}
		}
	}
	return b.String(), args
}

func (q *Query[T]) List() ([]T, error) {
	schema, err := q.tx.schema(reflect.TypeFor[T]())
	if err != nil {
		return nil, err
	}
	stmt, args := q.sql()
	if stmt == "" {
		return nil, errors.Invalidf("unknown or unsupported query field")
	}
	rows, err := q.tx.tx.QueryContext(q.tx.ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	all := [][]any{}
	for rows.Next() {
		raw, err := scanRaw(rows, len(schema.columns)+1)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		all = append(all, raw)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	out := []T{}
	for _, raw := range all {
		var value T
		if err := q.tx.hydrate(schema, reflect.ValueOf(&value).Elem(), raw); err != nil {
			return nil, err
		}
		include := true
		for _, fn := range q.residual {
			if !fn(value) {
				include = false
				break
			}
		}
		if include {
			out = append(out, value)
		}
	}
	return out, nil
}
func (q *Query[T]) Get() (T, error) {
	var zero T
	rows, err := q.List()
	if err != nil {
		return zero, err
	}
	if len(rows) == 0 {
		return zero, errors.NotFoundf("record absent")
	}
	if len(rows) > 1 {
		return zero, fmt.Errorf("query returned %s rows", strconv.Itoa(len(rows)))
	}
	return rows[0], nil
}

func (q *Query[T]) All() iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		rows, err := q.List()
		if err != nil {
			var zero T
			yield(zero, err)
			return
		}
		for _, row := range rows {
			if !yield(row, nil) {
				return
			}
		}
	}
}
func (q *Query[T]) Delete() (int, error) {
	rows, err := q.List()
	if err != nil {
		return 0, err
	}
	for i := range rows {
		if err := q.tx.Delete(&rows[i]); err != nil {
			return i, err
		}
	}
	return len(rows), nil
}
