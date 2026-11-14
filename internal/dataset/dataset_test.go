package dataset

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable(t *testing.T) {
	stubFetch(t, map[string]string{"u": "a,b,c,d\nx,NA,+3,true\ny,,4.5,no\n"})

	tb, err := OpenCSV("u")
	require.NoError(t, err)
	defer tb.Close()

	ok, err := tb.Next()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "x", tb.Str("a"))
	assert.Equal(t, "", tb.Str("b"), "NA is blank")
	assert.Equal(t, "", tb.Str("missing"))
	assert.Equal(t, 3.0, tb.Num("c"), "leading + is allowed")
	assert.True(t, tb.Bool("d"))
	assert.True(t, tb.Has("a"))
	assert.False(t, tb.Has("z"))

	ok, _ = tb.Next()
	require.True(t, ok)
	assert.Equal(t, 0.0, tb.Num("b"))
	assert.Equal(t, 4, tb.Int("c"))
	assert.False(t, tb.Bool("d"))

	ok, _ = tb.Next()
	assert.False(t, ok)
}

func TestOpenCSVGzip(t *testing.T) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte("a\n1\n"))
	w.Close()
	stubFetch(t, map[string]string{"https://x/f.csv.gz": b.String()})

	tb, err := OpenCSV("https://x/f.csv.gz")
	require.NoError(t, err)
	defer tb.Close()
	ok, _ := tb.Next()
	require.True(t, ok)
	assert.Equal(t, 1, tb.Int("a"))
}

func TestOpenCSVNotFound(t *testing.T) {
	stubFetch(t, nil)
	_, err := OpenCSV("nope")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCache(t *testing.T) {
	var c Cache[int]
	var calls atomic.Int32
	load := func(y int) (*int, error) { calls.Add(1); v := y * 2; return &v, nil }

	a, _ := c.Load(5, load)
	b, _ := c.Load(5, load)
	assert.Equal(t, 10, *a)
	assert.Same(t, a, b)
	assert.EqualValues(t, 1, calls.Load())

	c.Load(6, load)
	assert.EqualValues(t, 2, calls.Load())

	c.Reset()
	c.Load(5, load)
	assert.EqualValues(t, 3, calls.Load())
}

func TestCacheCachesErrors(t *testing.T) {
	var c Cache[int]
	var calls int
	boom := errors.New("boom")
	load := func(int) (*int, error) { calls++; return nil, boom }

	_, err := c.Load(1, load)
	assert.ErrorIs(t, err, boom)
	c.Load(1, load)
	assert.Equal(t, 1, calls)
}

// stubFetch replaces Fetch with fixtures keyed by URL. Any other URL behaves
// like a 404.
func stubFetch(t *testing.T, files map[string]string) {
	t.Helper()

	orig := Fetch
	Fetch = func(url string) (io.ReadCloser, error) {
		body, ok := files[url]
		if !ok {
			return nil, fmt.Errorf("%s: %w", url, ErrNotFound)
		}
		return io.NopCloser(strings.NewReader(body)), nil
	}
	t.Cleanup(func() { Fetch = orig })
}
