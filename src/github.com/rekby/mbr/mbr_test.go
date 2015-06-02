package parttable

import (
	"testing"
	"bytes"
)

func Test_writeLittleEndianUINT32(t *testing.T){
	buf := make([]byte, 4)
	writeLittleEndianUINT32(buf, 67305985)
	if buf[0] != 1 || buf[1] != 2 || buf[2] != 3 || buf[3] != 4{
		t.Error("Error")
	}
}

func Test_readLittleEndianUINT32(t *testing.T){
	buf := []byte{1,2,3,4}
	if readLittleEndianUINT32(buf) != 67305985 {
		t.Error("Error")
	}
}

func Test_fixSignature(t *testing.T){
	if mbrSignOffset != 510 {
		t.Error("MBR Offset error")
	}

	var buf bytes.Buffer
	buf.Write([512]byte{})
	mbr, _ := Read(buf)
	mbr.FixSignature()
	if mbr.bytes[mbrSignOffset] != 0x55 || mbr.bytes[mbrSignOffset] != 0xAA {
		t.Error("Error")
	}
}