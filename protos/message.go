package protos

type MessageType int

const (
	_ MessageType = iota
	MessageTypeHello
	MessageTypeData
)

type Message struct {
	Type    MessageType
	Data    []byte
	Address string
}
