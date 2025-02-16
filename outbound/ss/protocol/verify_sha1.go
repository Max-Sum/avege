package protocol

import (
	"bytes"
	"encoding/binary"

	"github.com/Max-Sum/avege/common"
	"github.com/Max-Sum/avege/outbound/ss/ssr"
)

func init() {
	register("verify_sha1", newVerifySHA1)
	register("ota", newVerifySHA1)
}

type verifySHA1 struct {
	ssr.ServerInfoForObfs
	hasSentHeader bool
	chunkId       uint32
}

const (
	oneTimeAuthMask byte = 0x10
)

func newVerifySHA1() IProtocol {
	a := &verifySHA1{}
	return a
}

func (v *verifySHA1) otaConnectAuth(data []byte) []byte {
	return append(data, common.HmacSHA1(append(v.IV, v.Key...), data)...)
}

func (v *verifySHA1) otaReqChunkAuth(chunkId uint32, data []byte) []byte {
	nb := make([]byte, 2)
	binary.BigEndian.PutUint16(nb, uint16(len(data)))
	chunkIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkIdBytes, chunkId)
	header := append(nb, common.HmacSHA1(append(v.IV, chunkIdBytes...), data)...)
	return append(header, data...)
}

func (v *verifySHA1) otaVerifyAuth(iv []byte, chunkId uint32, data []byte, expectedHmacSha1 []byte) bool {
	chunkIdBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkIdBytes, chunkId)
	actualHmacSha1 := common.HmacSHA1(append(iv, chunkIdBytes...), data)
	return bytes.Equal(expectedHmacSha1, actualHmacSha1)
}

func (v *verifySHA1) getAndIncreaseChunkId() (chunkId uint32) {
	chunkId = v.chunkId
	v.chunkId += 1
	return
}

func (v *verifySHA1) SetServerInfo(s *ssr.ServerInfoForObfs) {
	v.ServerInfoForObfs = *s
}

func (v *verifySHA1) GetServerInfo() (s *ssr.ServerInfoForObfs) {
	return &v.ServerInfoForObfs
}

func (v *verifySHA1) SetData(data interface{}) {

}

func (v *verifySHA1) GetData() interface{} {
	return nil
}

func (v *verifySHA1) PreEncrypt(data []byte) (encryptedData []byte, err error) {
	dataLength := len(data)
	offset := 0
	if !v.hasSentHeader {
		data[0] |= oneTimeAuthMask
		encryptedData = v.otaConnectAuth(data[:v.HeadLen])
		v.hasSentHeader = true
		dataLength -= v.HeadLen
		offset += v.HeadLen
	}
	const blockSize = 4096
	for dataLength > blockSize {
		chunkId := v.getAndIncreaseChunkId()
		b := v.otaReqChunkAuth(chunkId, data[offset:offset+blockSize])
		encryptedData = append(encryptedData, b...)
		dataLength -= blockSize
		offset += blockSize
	}
	if dataLength > 0 {
		chunkId := v.getAndIncreaseChunkId()
		b := v.otaReqChunkAuth(chunkId, data[offset:])
		encryptedData = append(encryptedData, b...)
	}
	return
}

func (v *verifySHA1) PostDecrypt(data []byte) (decryptedData []byte, err error) {
	return data, nil
}
