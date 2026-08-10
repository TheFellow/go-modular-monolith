package store

import (
	"database/sql"
	"encoding/json"
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

func idString(id reflect.Value) string { return fmt.Sprint(id.Interface()) }

func (t *Tx) Insert(values ...any) error {
	for _, value := range values {
		if err := t.insert(value); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tx) insert(value any) error {
	v, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	if id.IsZero() && id.CanSet() && ((id.Kind() >= reflect.Uint && id.Kind() <= reflect.Uint64) || (id.Kind() >= reflect.Int && id.Kind() <= reflect.Int64)) {
		var maxID sql.NullInt64
		err = t.tx.QueryRowContext(t.ctx, "SELECT max(CAST(json_extract(data, '$."+typ.Field(0).Name+"') AS INTEGER)) FROM records WHERE model = ?", modelName(typ)).Scan(&maxID)
		if err != nil {
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
	data, err := json.Marshal(v.Interface())
	if err != nil {
		return err
	}
	_, err = t.tx.ExecContext(t.ctx, "INSERT INTO records(model,id,data) VALUES(?,?,?)", modelName(typ), idString(id), string(data))
	if err != nil && isUniqueConstraint(err) {
		return errors.Conflictf("unique constraint: %w", err)
	}
	return err
}

func (t *Tx) Update(value any) error {
	_, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	if id.IsZero() {
		return errors.Invalidf("record ID is required")
	}
	data, err := json.Marshal(reflect.ValueOf(value).Elem().Interface())
	if err != nil {
		return err
	}
	r, err := t.tx.ExecContext(t.ctx, "UPDATE records SET data=? WHERE model=? AND id=?", string(data), modelName(typ), idString(id))
	if err != nil {
		if isUniqueConstraint(err) {
			return errors.Conflictf("unique constraint: %w", err)
		}
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return errors.NotFoundf("record absent")
	}
	return nil
}

func (t *Tx) Get(value any) error {
	_, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	if id.IsZero() {
		return errors.Invalidf("record ID is required")
	}
	var data string
	err = t.tx.QueryRowContext(t.ctx, "SELECT data FROM records WHERE model=? AND id=?", modelName(typ), idString(id)).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.NotFoundf("record absent")
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), value)
}

func (t *Tx) Delete(value any) error {
	_, typ, id, err := rowInfo(value)
	if err != nil {
		return err
	}
	r, err := t.tx.ExecContext(t.ctx, "DELETE FROM records WHERE model=? AND id=?", modelName(typ), idString(id))
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return errors.NotFoundf("record absent")
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
		return value.UTC().Format(time.RFC3339Nano)
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

func queryFieldExpression(typ reflect.Type, name string) (string, bool) {
	expression := "json_extract(data, '$." + name + "')"
	field, ok := typ.FieldByName(name)
	timeField := ok && (field.Type == reflect.TypeFor[time.Time]() || (field.Type.Kind() == reflect.Pointer && field.Type.Elem() == reflect.TypeFor[time.Time]()))
	if timeField {
		expression = "julianday(" + expression + ")"
	}
	return expression, timeField
}

func (q *Query[T]) sql() (string, []any) {
	var z T
	typ := reflect.TypeOf(z)
	name := modelName(typ)
	b := strings.Builder{}
	b.WriteString("SELECT data FROM records WHERE model=?")
	args := []any{name}
	for _, p := range q.predicates {
		path, timeField := queryFieldExpression(typ, p.field)
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
				if timeField {
					b.WriteString("julianday(?)")
				} else {
					b.WriteByte('?')
				}
				args = append(args, normalize(v))
			}
			b.WriteByte(')')
		} else if len(p.values) == 1 {
			placeholder := "?"
			if timeField {
				placeholder = "julianday(?)"
			}
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
			expression, _ := queryFieldExpression(typ, o.field)
			b.WriteString(expression)
			if o.desc {
				b.WriteString(" DESC")
			}
		}
	}
	return b.String(), args
}

func (q *Query[T]) List() ([]T, error) {
	stmt, args := q.sql()
	rows, err := q.tx.tx.QueryContext(q.tx.ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []T{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			return nil, err
		}
		ok := true
		for _, fn := range q.residual {
			if !fn(v) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, v)
		}
	}
	return out, rows.Err()
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
