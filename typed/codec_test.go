package typed_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/silviolleite/loafer-natsx/typed"
)

func TestJSONCodec_RoundTrip(t *testing.T) {
	codec := typed.JSONCodec[order]{}

	orig := order{ID: "abc-123", Amount: 42.5}
	data, err := codec.Encode(orig)
	require.NoError(t, err)

	decoded, err := codec.Decode(data)
	require.NoError(t, err)
	assert.Equal(t, orig, decoded)
}

func TestJSONCodec_DecodeInvalidData(t *testing.T) {
	codec := typed.JSONCodec[order]{}

	_, err := codec.Decode([]byte("not json"))
	assert.Error(t, err)
}
