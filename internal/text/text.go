// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package text

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/scanner"
)

const (
	// Filename references the Filename field of a Failure.
	Filename FailureField = iota
	// Line references the Line field of a Failure.
	Line
	// Column references the Column field of a Failure.
	Column
	// ID references the ID field of a Failure.
	ID
	// Message references the Message field of a Failure.
	Message
)

var (
	_defaultFailureFields = []FailureField{
		Filename,
		Line,
		Column,
		Message,
	}
	_failureFieldToString = map[FailureField]string{
		Filename: "filename",
		Line:     "line",
		Column:   "column",
		ID:       "id",
		Message:  "message",
	}
	_stringToFailureField = map[string]FailureField{
		"filename": Filename,
		"line":     Line,
		"column":   Column,
		"id":       ID,
		"message":  Message,
	}
)

// FailureField references a field of a Failure.
type FailureField int

// String implements fmt.Stringer.
func (f FailureField) String() string {
	if s, ok := _failureFieldToString[f]; ok {
		return s
	}
	return strconv.Itoa(int(f))
}

// ParseFailureFields parses FailureFields from the given string.
// FailureFields are expected to be colon-separated in the given string.
// Input is case-insensitive. If the string is empty, _defaultFailureFields will be
// returned.
func ParseFailureFields(s string) ([]FailureField, error) {
	if len(s) == 0 {
		return _defaultFailureFields, nil
	}
	fields := strings.Split(s, ":")
	failureFields := make([]FailureField, len(fields))
	for i, f := range fields {
		ff, err := parseFailureField(f)
		if err != nil {
			return nil, err
		}
		failureFields[i] = ff
	}
	return failureFields, nil
}

// Failure is a failure with a position in text.
type Failure struct {
	Filename string
	Line     int
	Column   int
	ID       string
	Message  string
}

// FailureWriter is a writer that Failure.Println can accept.
//
// Both bytes.Buffer and bufio.Writer implement this.
type FailureWriter interface {
	WriteRune(rune) (int, error)
	WriteString(string) (int, error)
}

// Fprintln prints the Failure to the writer with the given ordered fields.
func (f *Failure) Fprintln(writer FailureWriter, fields ...FailureField) error {
	if len(fields) == 0 {
		fields = _defaultFailureFields
	}
	written := false
	for i, field := range fields {
		printColon := true
		switch field {
		case Filename:
			filename := f.Filename
			if filename == "" {
				filename = "<input>"
			}
			if _, err := writer.WriteString(filename); err != nil {
				return err
			}
			written = true
		case Line:
			line := strconv.Itoa(f.Line)
			if line == "0" {
				line = "1"
			}
			if _, err := writer.WriteString(line); err != nil {
				return err
			}
			written = true
		case Column:
			column := strconv.Itoa(f.Column)
			if column == "0" {
				column = "1"
			}
			if _, err := writer.WriteString(column); err != nil {
				return err
			}
			written = true
		case ID:
			if f.ID != "" {
				if _, err := writer.WriteString(f.ID); err != nil {
					return err
				}
				written = true
			} else {
				printColon = false
			}
		case Message:
			if f.Message != "" {
				if _, err := writer.WriteString(f.Message); err != nil {
					return err
				}
				written = true
			} else {
				printColon = false
			}
		default:
			return fmt.Errorf("unknown FailureField: %v", field)
		}
		if printColon && i != len(fields)-1 {
			if _, err := writer.WriteRune(':'); err != nil {
				return err
			}
			written = true
		}
	}
	if written {
		_, err := writer.WriteRune('\n')
		return err
	}
	return nil
}

// String implements fmt.Stringer. The Failure is
// printed in the following format:
//
//  "<filename>:<line>:<column>:<id> <message>"
func (f *Failure) String() string {
	filename := f.Filename
	if filename == "" {
		filename = "<input>"
	}
	line := strconv.Itoa(f.Line)
	if line == "0" {
		line = "1"
	}
	column := strconv.Itoa(f.Column)
	if column == "0" {
		column = "1"
	}

	return fmt.Sprintf("%s:%s:%s:%s %s", filename, line, column, f.ID, f.Message)
}

// NewFailuref is a helper that returns a new Failure.
func NewFailuref(position scanner.Position, id string, format string, args ...interface{}) *Failure {
	return &Failure{
		ID:       id,
		Filename: position.Filename,
		Line:     position.Line,
		Column:   position.Column,
		Message:  fmt.Sprintf(format, args...),
	}
}

// SortFailures sorts the Failures by the following precedence:
//
//  filename > line > column > id > message
func SortFailures(failures []*Failure) {
	sort.Stable(sortFailures(failures))
}

type sortFailures []*Failure

func (f sortFailures) Len() int          { return len(f) }
func (f sortFailures) Swap(i int, j int) { f[i], f[j] = f[j], f[i] }
func (f sortFailures) Less(i int, j int) bool {
	if f[i] == nil && f[j] == nil {
		return false
	}
	if f[i] == nil && f[j] != nil {
		return true
	}
	if f[i] != nil && f[j] == nil {
		return false
	}
	if f[i].Filename < f[j].Filename {
		return true
	}
	if f[i].Filename > f[j].Filename {
		return false
	}
	if f[i].Line < f[j].Line {
		return true
	}
	if f[i].Line > f[j].Line {
		return false
	}
	if f[i].Column < f[j].Column {
		return true
	}
	if f[i].Column > f[j].Column {
		return false
	}
	if f[i].ID < f[j].ID {
		return true
	}
	if f[i].ID > f[j].ID {
		return false
	}
	if f[i].Message < f[j].Message {
		return true
	}
	return false
}

// parseFailureField parses the FailureField from the given string.
// Input is case-insensitive.
func parseFailureField(s string) (FailureField, error) {
	failureField, ok := _stringToFailureField[strings.ToLower(s)]
	if !ok {
		return 0, fmt.Errorf("could not parse %s to a FailureField", s)
	}
	return failureField, nil
}