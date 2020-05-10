package document

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArrayContains(t *testing.T) {
	arr := NewValueBuffer(
		NewIntValue(1),
		NewTextValue("foo"),
		NewBlobValue([]byte{1, 2, 3}),
	)

	ok, err := ArrayContains(arr, NewFloat64Value(1))
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = ArrayContains(arr, NewBlobValue([]byte("foo")))
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = ArrayContains(arr, NewBlobValue([]byte("bar")))
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSortArray(t *testing.T) {
	tests := []struct {
		name     string
		arr      string
		expected string
	}{
		{"empty array", `[]`, `[]`},
		{"numbers", `[1.4,3,2.1,-5]`, `[-5,1.4,2.1,3]`},
		{"text", `["foo","bar",""]`, `["","bar","foo"]`},
		{"arrays", `[[1, 2],[-1,10],[]]`, `[[],[-1,10],[1,2]]`},
		{"documents", `[{"z":10},{"a":40},{}]`, `[{"z":10},{"a":40},{}]`},
		{"mixed", `["foo",["a"],{},null,true,10]`, `[null,true,10,"foo",["a"],{}]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var arr ValueBuffer
			require.NoError(t, arr.UnmarshalJSON([]byte(test.arr)))
			output, err := SortArray(arr)
			require.NoError(t, err)
			actual, err := json.Marshal(output)
			require.NoError(t, err)
			require.Equal(t, test.expected, string(actual))
		})
	}
}