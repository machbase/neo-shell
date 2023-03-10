package packet4

import (
	"bytes"
	"fmt"
	"io"
)

type UnsubscribePacket struct {
	FixedHeader
	MessageID uint16
	Topics    []string
}

func (u *UnsubscribePacket) String() string {
	return fmt.Sprintf("%s MessageID: %d", u.FixedHeader, u.MessageID)
}

func (u *UnsubscribePacket) Write(w io.Writer) (int64, error) {
	var body bytes.Buffer
	body.Write(encodeUint16(u.MessageID))
	for _, topic := range u.Topics {
		body.Write(encodeString(topic))
	}
	u.FixedHeader.RemainingLength = body.Len()
	packet := u.FixedHeader.pack()
	packet.Write(body.Bytes())
	nbytes, err := packet.WriteTo(w)

	return nbytes, err
}

// Unpack decodes the details of a ControlPacket after the fixed
// header has been read
func (u *UnsubscribePacket) Unpack(b io.Reader) error {
	var err error
	u.MessageID, err = decodeUint16(b)
	if err != nil {
		return err
	}

	for topic, err := decodeString(b); err == nil && topic != ""; topic, err = decodeString(b) {
		u.Topics = append(u.Topics, topic)
	}

	return err
}

// Details returns a Details struct containing the Qos and
// MessageID of this ControlPacket
func (u *UnsubscribePacket) Details() Details {
	return Details{Qos: 1, MessageID: u.MessageID}
}
