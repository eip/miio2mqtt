package miio

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	h "github.com/eip/miio2mqtt/helpers"
)

// Packet represents a miIO protocol network packet
type Packet struct {
	Magic    uint16
	Length   uint16
	Unused   uint32
	DeviceID uint32
	Stamp    uint32
	Checksum [16]byte
	Data     []byte
}

var errInvalidMagicField = errors.New("invalid magic field")
var errInvalidDataLength = errors.New("invalid data length")
var errInvalidTokenLength = errors.New("invalid token length")
var errInvalidChecksum = errors.New("invalid checksum")
var errInvalidChecksumLength = errors.New("invalid checksum length")
var errInvalidBlockSize = errors.New("invalid block size")
var errInvalidPadding = errors.New("invalid padding")

// NewHelloPacket creates a Hello packet
func NewHelloPacket() *Packet {
	p := Packet{
		Magic:    0x2131,
		Length:   0x0020,
		Unused:   0xffffffff,
		DeviceID: 0xffffffff,
		Stamp:    0xffffffff,
		Checksum: [16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		Data:     []byte{},
	}
	return &p
}

// NewPacket creates a packet with the given properties
func NewPacket(deviceID uint32, time time.Time, data []byte) *Packet {
	p := Packet{
		Magic:    0x2131,
		Length:   0x0020 + uint16(len(data)),
		Unused:   0x00000000,
		DeviceID: deviceID,
		Stamp:    uint32(time.Unix()),
		Data:     make([]byte, len(data)),
	}
	copy(p.Data, data)
	// TODO calc checksum
	// checksum := bytes.Repeat([]byte{0xff}, 16)
	// copy(p.Checksum[:], checksum)
	return &p
}

// GetDeviceID extracts DeviceID from raw packet data
func GetDeviceID(data []byte) (uint32, error) {
	if len(data) < 32 {
		return 0, errInvalidDataLength
	}
	return binary.BigEndian.Uint32(data[8:12]), nil
}

// Decode creates a packet from the byte slice
func Decode(data []byte, token []byte) (*Packet, error) {
	p, err := decode(data)
	if err != nil {
		return nil, err
	}
	if err := p.Validate(token); err != nil {
		return nil, err
	}
	return p.decrypt(token)
}

func decode(data []byte) (*Packet, error) {
	if len(data) < 32 {
		return nil, errInvalidDataLength
	}
	p := Packet{}
	buf := bytes.NewReader(data)
	for _, v := range []interface{}{&p.Magic, &p.Length, &p.Unused, &p.DeviceID, &p.Stamp, p.Checksum[:]} {
		if err := binary.Read(buf, binary.BigEndian, v); err != nil {
			return nil, err
		}
	}
	p.Data = data[32:]
	return &p, nil
}

// Encode converts the packet into a byte slice
func (p *Packet) Encode(token []byte) ([]byte, error) {
	penc, err := p.encrypt(token)
	if err != nil {
		return nil, err
	}
	return penc.encode(nil)
}

func (p *Packet) encode(checksum []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	if len(checksum) == 0 {
		checksum = p.Checksum[:]
	} else if len(checksum) != 16 {
		return nil, errInvalidChecksumLength
	}
	for _, v := range []interface{}{p.Magic, p.Length, p.Unused, p.DeviceID, p.Stamp, checksum, p.Data} {
		if dataLen(v) == 0 { // empty []byte
			continue
		}
		if err := binary.Write(buf, binary.BigEndian, v); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// CalcChecksum calculates the packet checksum
func (p *Packet) CalcChecksum(token []byte) ([]byte, error) {
	if len(p.Data) == 0 {
		return nil, errInvalidDataLength
	}
	if len(token) != 16 {
		return nil, errInvalidTokenLength
	}
	data, err := p.encode(token)
	if err != nil {
		return nil, err
	}
	digest := md5.Sum(data)
	return digest[:], nil
}

// Validate checks the validity of the packet fields
func (p *Packet) Validate(token []byte) error {
	if p.Magic != 0x2131 {
		return errInvalidMagicField
	}
	if p.Length != 32+uint16(len(p.Data)) {
		return errInvalidDataLength
	}
	ok, err := p.validateChecksum(token)
	if err != nil {
		return err
	}
	if !ok {
		return errInvalidChecksum
	}
	return nil
}

// Str describes the packet as a string
func (p *Packet) Str() string {
	if p.Unused == 0xffffffff && p.DeviceID == 0xffffffff && p.Stamp == 0xffffffff {
		return "<Hello Packet>"
	}
	// ts := time.Unix(int64(p.Stamp), 0).UTC().Format("2006-01-02 15:04:05")
	ts := time.Duration(p.Stamp) * time.Second
	format := "{deviceID: %08x, time: %v}"
	if len(p.Data) == 0 {
		return fmt.Sprintf(format, p.DeviceID, ts)
	}
	format = format[:len(format)-1] + ", data: "
	if h.IsPrintableASCII(p.Data) {
		format += "%s}"
	} else {
		format += "%x}"
	}
	return fmt.Sprintf(format, p.DeviceID, ts, p.Data)
}

func (p *Packet) TimeStamp() int64 {
	return int64(p.Stamp) * int64(time.Second)
}

func (p *Packet) validateChecksum(token []byte) (bool, error) {
	if len(p.Data) == 0 {
		b := byte(0xff)
		for i, v := range p.Checksum {
			if i == 0 && v == 0x00 {
				b = v
				continue
			}
			if v != b {
				return false, nil
			}
		}
		return true, nil
	}
	if len(token) != 16 {
		return false, errInvalidTokenLength
	}
	checksum, err := p.CalcChecksum(token)
	if err != nil {
		return false, err
	}
	return bytes.Equal(checksum, p.Checksum[:]), nil
}

func (p *Packet) decrypt(token []byte) (*Packet, error) {
	if len(token) == 0 || len(p.Data) == 0 {
		return p, nil
	}
	if len(token) != 16 {
		return nil, errInvalidTokenLength
	}
	key := hash(token)
	iv := hash(append(key, token...))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(p.Data)%block.BlockSize() != 0 {
		return nil, errInvalidDataLength
	}
	stream := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(p.Data))
	stream.CryptBlocks(decrypted, p.Data)
	decrypted, _ = pkcs7strip(decrypted, block.BlockSize())
	p.Data = decrypted
	p.Length = 0x0020 + uint16(len(decrypted))
	for i := range p.Checksum {
		p.Checksum[i] = 0
	}
	return p, nil
}

func (p *Packet) encrypt(token []byte) (*Packet, error) {
	if len(token) == 0 || len(p.Data) == 0 {
		return p, nil
	}
	if len(token) != 16 {
		return nil, errInvalidTokenLength
	}
	key := hash(token)
	iv := hash(append(key, token...))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCBCEncrypter(block, iv)
	encrypted, _ := pkcs7pad(p.Data, block.BlockSize())
	stream.CryptBlocks(encrypted, encrypted)
	result := *p
	result.Data = encrypted
	result.Length = 0x0020 + uint16(len(encrypted))
	checksum, err := result.CalcChecksum(token)
	if err != nil {
		return nil, err
	}
	copy(result.Checksum[:], checksum)
	return &result, nil
}

func dataLen(data interface{}) int {
	switch data := data.(type) {
	case nil:
		return 0
	case []byte:
		return len(data)
	}
	return -1
}

func hash(data []byte) []byte {
	digest := md5.Sum(data)
	return digest[:]
}

func pkcs7pad(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 1 || blockSize > 255 {
		return nil, errInvalidBlockSize
	}
	if len(data) == 0 {
		return nil, errInvalidDataLength
	}
	padLen := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...), nil
}

func pkcs7strip(data []byte, blockSize int) ([]byte, error) {
	if blockSize <= 1 || blockSize > 255 {
		return nil, errInvalidBlockSize
	}
	length := len(data)
	if length == 0 || length%blockSize != 0 {
		return nil, errInvalidDataLength
	}
	padLen := int(data[length-1])
	ref := bytes.Repeat([]byte{byte(padLen)}, padLen%(blockSize+1))
	if padLen > blockSize || padLen == 0 || !bytes.HasSuffix(data, ref) {
		return nil, errInvalidPadding
	}
	return data[:length-padLen], nil
}
