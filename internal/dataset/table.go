package dataset

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Table is a streaming CSV reader with access to columns by header name.
type Table struct {
	r   *csv.Reader
	idx map[string]int
	rec []string
	rc  io.Closer
}

// OpenCSV downloads a CSV (gzipped if the URL ends in .gz) and reads its header.
func OpenCSV(url string) (*Table, error) {
	body, err := Fetch(url)
	if err != nil {
		return nil, err
	}

	var src io.Reader = body
	if strings.HasSuffix(url, ".gz") {
		gz, err := gzip.NewReader(body)
		if err != nil {
			body.Close()
			return nil, fmt.Errorf("%s: %w", url, err)
		}
		src = gz
	}

	r := csv.NewReader(src)
	r.FieldsPerRecord = -1
	r.ReuseRecord = true

	header, err := r.Read()
	if err != nil {
		body.Close()
		return nil, fmt.Errorf("%s: reading header: %w", url, err)
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}

	return &Table{r: r, idx: idx, rc: body}, nil
}

func (t *Table) Close() { t.rc.Close() }

// Next advances to the next record. It returns an error on a malformed file.
func (t *Table) Next() (bool, error) {
	rec, err := t.r.Read()
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	t.rec = rec
	return true, nil
}

// Str is the current record's value for a column, or "" if the column is
// missing or the value is NA.
func (t *Table) Str(col string) string {
	i, ok := t.idx[col]
	if !ok || i >= len(t.rec) {
		return ""
	}
	v := t.rec[i]
	if v == "NA" {
		return ""
	}
	return v
}

// Num parses a column as a number; blanks and unparseable values are 0. A
// leading "+" (as in plus/minus columns) is allowed.
func (t *Table) Num(col string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimPrefix(t.Str(col), "+"), 64)
	return v
}

func (t *Table) Int(col string) int { return int(t.Num(col)) }

func (t *Table) Bool(col string) bool { return strings.EqualFold(t.Str(col), "true") }

// Has reports whether the file has the column.
func (t *Table) Has(col string) bool {
	_, ok := t.idx[col]
	return ok
}
