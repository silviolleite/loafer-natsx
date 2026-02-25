package typed

import "encoding/json"

// Codec defines how to serialize and deserialize a value of type T.
type Codec[T any] interface {
	// Encode serializes v into a byte slice.
	Encode(v T) ([]byte, error)

	// Decode deserializes data into a value of type T.
	Decode(data []byte) (T, error)
}

// JSONCodec is a Codec implementation that uses encoding/json.
type JSONCodec[T any] struct{}

// Encode serializes v to JSON.
func (JSONCodec[T]) Encode(v T) ([]byte, error) {
	return json.Marshal(v)
}

// Decode deserializes JSON data into a value of type T.
func (JSONCodec[T]) Decode(data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}
