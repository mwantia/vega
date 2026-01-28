package value

import (
	"fmt"
	"time"
)

// Time represents a point in time.
type Time struct {
	Value time.Time
}

var _ Value = (*Time)(nil)
var _ Numeric = (*Time)(nil)
var _ Comparable = (*Time)(nil)
var _ Methodable = (*Time)(nil)
var _ Memberable = (*Time)(nil)

// NewTime creates a new Time value from a time.Time.
func NewTime(t time.Time) *Time {
	return &Time{
		Value: t,
	}
}

// NewTimeNow creates a new Time value representing the current time.
func NewTimeNow() *Time {
	return &Time{
		Value: time.Now(),
	}
}

func (t *Time) Type() string {
	return TypeTime
}

func (t *Time) String() string {
	return t.Value.Format(time.RFC3339)
}

func (t *Time) Boolean() bool {
	return !t.Value.IsZero()
}

func (t *Time) Equal(other Value) bool {
	if o, ok := other.(*Time); ok {
		return t.Value.Equal(o.Value)
	}
	return false
}

// GetMember returns time properties as members.
func (t *Time) GetMember(name string) (Value, error) {
	switch name {
	case "year":
		return NewInteger(t.Value.Year()), nil
	case "month":
		return NewInteger(int(t.Value.Month())), nil
	case "day":
		return NewInteger(t.Value.Day()), nil
	case "hour":
		return NewInteger(t.Value.Hour()), nil
	case "minute":
		return NewInteger(t.Value.Minute()), nil
	case "second":
		return NewInteger(t.Value.Second()), nil
	case "weekday":
		return NewInteger(int(t.Value.Weekday())), nil
	case "yearday":
		return NewInteger(t.Value.YearDay()), nil
	case "unix":
		return NewInteger(int(t.Value.Unix())), nil
	case "unixmilli":
		return NewInteger(int(t.Value.UnixMilli())), nil
	case "unixnano":
		return NewInteger(int(t.Value.UnixNano())), nil
	}
	return nil, fmt.Errorf("unknown member '%s' for type '%s'", name, t.Type())
}

// SetMember returns an error as time members are read-only.
func (t *Time) SetMember(name string, val Value) (bool, error) {
	return false, fmt.Errorf("cannot set member '%s' on type '%s'", name, t.Type())
}

// Method handles time methods that take arguments or return transformed times.
func (t *Time) Method(name string, args []Value) (Value, error) {
	switch name {
	case "format":
		if len(args) != 1 {
			return nil, fmt.Errorf("format() expects 1 argument, got %d", len(args))
		}
		layout, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("format() argument must be string, got %s", args[0].Type())
		}
		return NewString(t.Value.Format(layout.Value)), nil
	case "utc":
		if len(args) != 0 {
			return nil, fmt.Errorf("utc() expects 0 arguments, got %d", len(args))
		}
		return NewTime(t.Value.UTC()), nil
	case "local":
		if len(args) != 0 {
			return nil, fmt.Errorf("local() expects 0 arguments, got %d", len(args))
		}
		return NewTime(t.Value.Local()), nil
	}
	return nil, fmt.Errorf("unknown method '%s' for type '%s'", name, t.Type())
}

func (t *Time) Compare(other Value) (int, bool) {
	if o, ok := other.(*Time); ok {
		if t.Value.Before(o.Value) {
			return -1, true
		} else if t.Value.After(o.Value) {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// Add adds seconds (integer or float) to the time.
func (t *Time) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *Integer:
		return NewTime(t.Value.Add(time.Duration(o.Value) * time.Second)), nil
	case *Float:
		return NewTime(t.Value.Add(time.Duration(o.Value * float64(time.Second)))), nil
	}
	return nil, fmt.Errorf("cannot add %s to time", other.Type())
}

// Sub subtracts seconds from the time, or returns difference between two times in seconds.
func (t *Time) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *Time:
		return NewFloat(t.Value.Sub(o.Value).Seconds()), nil
	case *Integer:
		return NewTime(t.Value.Add(time.Duration(-o.Value) * time.Second)), nil
	case *Float:
		return NewTime(t.Value.Add(time.Duration(-o.Value * float64(time.Second)))), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from time", other.Type())
}

func (t *Time) Mul(other Value) (Value, error) {
	return nil, fmt.Errorf("cannot multiply time")
}

func (t *Time) Div(other Value) (Value, error) {
	return nil, fmt.Errorf("cannot divide time")
}

func (t *Time) Mod(other Value) (Value, error) {
	return nil, fmt.Errorf("cannot modulo time")
}

func (t *Time) Neg() (Value, error) {
	return nil, fmt.Errorf("cannot negate time")
}
