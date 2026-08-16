package serde

type Serializer interface {
	Serialize() []byte
}

func SerializeObjects(objects ...Serializer) []byte {
	var buffer []byte
	for _, object := range objects {
		buffer = append(buffer, object.Serialize()...)
	}

	return buffer
}
