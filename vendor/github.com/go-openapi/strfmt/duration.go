// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strfmt

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	d := Duration(0)
	// register this format in the default registry
	Default.Add("duration", &d, IsDuration)
}

const (
	hoursInDay = 24
	daysInWeek = 7
)

var (
	timeUnits = [][]string{
		{"ns", "nano"},
		{"us", "µs", "micro"},
		{"ms", "milli"},
		{"s", "sec"},
		{"m", "min"},
		{"h", "hr", "hour"},
		{"d", "day"},
		{"w", "wk", "week"},
	}

	timeMultiplier = map[string]time.Duration{
		"ns": time.Nanosecond,
		"us": time.Microsecond,
		"ms": time.Millisecond,
		"s":  time.Second,
		"m":  time.Minute,
		"h":  time.Hour,
		"d":  hoursInDay * time.Hour,
		"w":  hoursInDay * daysInWeek * time.Hour,
	}

	durationMatcher = regexp.MustCompile(`(((?:-\s?)?\d+)\s*([A-Za-zµ]+))`)
)

// IsDuration returns true if the provided string is a valid duration
func IsDuration(str string) bool {
	_, err := ParseDuration(str)
	return err == nil
}

// Duration represents a duration
//
// Duration stores a period of time as a nanosecond count, with the largest
// repesentable duration being approximately 290 years.
//
// swagger:strfmt duration
type Duration time.Duration

// MarshalText turns this instance into text
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// UnmarshalText hydrates this instance from text
func (d *Duration) UnmarshalText(data []byte) error { // validation is performed later on
	dd, err := ParseDuration(string(data))
	if err != nil {
		return err
	}
	*d = Duration(dd)
	return nil
}

// ParseDuration parses a duration from a string, compatible with scala duration syntax
func ParseDuration(cand string) (time.Duration, error) {
	if dur, err := time.ParseDuration(cand); err == nil {
		return dur, nil
	}

	var dur time.Duration
	ok := false
	for _, match := range durationMatcher.FindAllStringSubmatch(cand, -1) {

		// remove possible leading - and spaces
		value, negative := strings.CutPrefix(match[2], "-")

		// if the string is a valid duration, parse it
		factor, err := strconv.Atoi(strings.TrimSpace(value)) // converts string to int
		if err != nil {
			return 0, err
		}

		if negative {
			factor = -factor
		}

		unit := strings.ToLower(strings.TrimSpace(match[3]))

		for _, variants := range timeUnits {
			last := len(variants) - 1
			multiplier := timeMultiplier[variants[0]]

			for i, variant := range variants {
				if (last == i && strings.HasPrefix(unit, variant)) || strings.EqualFold(variant, unit) {
					ok = true
					dur += (time.Duration(factor) * multiplier)
				}
			}
		}
	}

	if ok {
		return dur, nil
	}
	return 0, fmt.Errorf("unable to parse %s as duration: %w", cand, ErrFormat)
}

// Scan reads a Duration value from database driver type.
func (d *Duration) Scan(raw interface{}) error {
	switch v := raw.(type) {
	// TODO: case []byte: // ?
	case int64:
		*d = Duration(v)
	case float64:
		*d = Duration(int64(v))
	case nil:
		*d = Duration(0)
	default:
		return fmt.Errorf("cannot sql.Scan() strfmt.Duration from: %#v: %w", v, ErrFormat)
	}

	return nil
}

// Value converts Duration to a primitive value ready to be written to a database.
func (d Duration) Value() (driver.Value, error) {
	return driver.Value(int64(d)), nil
}

// String converts this duration to a string
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON returns the Duration as JSON
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON sets the Duration from JSON
func (d *Duration) UnmarshalJSON(data []byte) error {
	if string(data) == jsonNull {
		return nil
	}

	var dstr string
	if err := json.Unmarshal(data, &dstr); err != nil {
		return err
	}
	tt, err := ParseDuration(dstr)
	if err != nil {
		return err
	}
	*d = Duration(tt)
	return nil
}

func (d Duration) MarshalBSON() ([]byte, error) {
	return bson.Marshal(bson.M{"data": d.String()})
}

func (d *Duration) UnmarshalBSON(data []byte) error {
	var m bson.M
	if err := bson.Unmarshal(data, &m); err != nil {
		return err
	}

	if data, ok := m["data"].(string); ok {
		rd, err := ParseDuration(data)
		if err != nil {
			return err
		}
		*d = Duration(rd)
		return nil
	}

	return fmt.Errorf("couldn't unmarshal bson bytes value as Date: %w", ErrFormat)
}

// DeepCopyInto copies the receiver and writes its value into out.
func (d *Duration) DeepCopyInto(out *Duration) {
	*out = *d
}

// DeepCopy copies the receiver into a new Duration.
func (d *Duration) DeepCopy() *Duration {
	if d == nil {
		return nil
	}
	out := new(Duration)
	d.DeepCopyInto(out)
	return out
}
