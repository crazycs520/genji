package record

import (
	"fmt"
	"testing"

	"github.com/asdine/genji/value"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	data, err := Encode(FieldBuffer([]Field{
		NewInt64Field("age", 10),
		NewStringField("name", "john"),
	}))
	require.NoError(t, err)

	var f Format
	err = f.Decode(data)
	require.NoError(t, err)
	require.Equal(t, len(f.Body), f.Header.BodySize())
	require.EqualValues(t, 2, f.Header.FieldsCount)
	require.Len(t, f.Header.FieldHeaders, 2)

	require.EqualValues(t, "age", f.Header.FieldHeaders[0].Name)
	require.EqualValues(t, 3, f.Header.FieldHeaders[0].NameSize)
	require.EqualValues(t, 8, f.Header.FieldHeaders[0].Size)
	require.EqualValues(t, value.Int64, f.Header.FieldHeaders[0].Type)
	require.EqualValues(t, 0, f.Header.FieldHeaders[0].Offset)

	require.EqualValues(t, "name", f.Header.FieldHeaders[1].Name)
	require.EqualValues(t, 4, f.Header.FieldHeaders[1].NameSize)
	require.EqualValues(t, 4, f.Header.FieldHeaders[1].Size)
	require.EqualValues(t, value.String, f.Header.FieldHeaders[1].Type)
	require.EqualValues(t, 8, f.Header.FieldHeaders[1].Offset)
}

func TestDecodeField(t *testing.T) {
	rec := FieldBuffer([]Field{
		NewInt64Field("age", 10),
		NewStringField("name", "john"),
	})

	data, err := Encode(rec)
	require.NoError(t, err)

	f, err := DecodeField(data, "age")
	require.NoError(t, err)
	require.Equal(t, rec[0], f)

	f, err = DecodeField(data, "name")
	require.NoError(t, err)
	require.Equal(t, rec[1], f)
}

func TestEncodedRecord(t *testing.T) {
	rec := FieldBuffer([]Field{
		NewInt64Field("age", 10),
		NewStringField("name", "john"),
	})

	data, err := Encode(rec)
	require.NoError(t, err)

	ec := EncodedRecord(data)
	f, err := ec.GetField("age")
	require.NoError(t, err)
	require.Equal(t, rec[0], f)

	var i int
	err = ec.Iterate(func(f Field) error {
		require.Equal(t, rec[i], f)
		i++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, i)
}

func BenchmarkEncode(b *testing.B) {
	var fields []Field

	for i := int64(0); i < 100; i++ {
		fields = append(fields, NewInt64Field(fmt.Sprintf("name-%d", i), i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Encode(FieldBuffer(fields))
	}
}

func BenchmarkFormatDecode(b *testing.B) {
	var fields []Field

	for i := int64(0); i < 100; i++ {
		fields = append(fields, NewInt64Field(fmt.Sprintf("name-%d", i), i))
	}

	data, err := Encode(FieldBuffer(fields))
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var f Format
		f.Decode(data)
	}
}

func BenchmarkDecodeField(b *testing.B) {
	var fields []Field

	for i := int64(0); i < 100; i++ {
		fields = append(fields, NewInt64Field(fmt.Sprintf("name-%d", i), i))
	}
	data, err := Encode(FieldBuffer(fields))
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeField(data, "name-99")
	}
}

func BenchmarkEncodedRecord(b *testing.B) {
	var fields []Field

	for i := int64(0); i < 100; i++ {
		fields = append(fields, NewInt64Field(fmt.Sprintf("name-%d", i), i))
	}
	data, err := Encode(FieldBuffer(fields))
	require.NoError(b, err)

	ec := EncodedRecord(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ec.Iterate(func(Field) error {
			return nil
		})
	}
}
