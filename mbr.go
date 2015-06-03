package mbr

import (
	"errors"
	"io"
)

var ErrorBadMbrSign = errors.New("MBR: Bad signature")
var ErrorPartitionsIntersection = errors.New("MBR: Partitions have intersections")
var ErrorPartitionLastSectorHigh = errors.New("MBR: Last sector have very high number")
var ErrorPartitionBootFlag = errors.New("MBR: Bad value in boot flag")

type MBR struct {
	bytes []byte
}

type MBRPartition struct {
	bytes []byte
}

const mbrFirstPartEntryOffset = 446 // bytes
const mbrPartEntrySize = 16         // bytes
const mbrSize = 512                 // bytes
const mbrSignOffset = 510           // bytes

const partitionBootableOffset = 0   // bytes
const partitionTypeOffset = 4       // bytes
const partitionLBAStartOffset = 8   // bytes
const partitionLBALengthOffset = 12 // bytes

const partitionEmptyType = 0
const partitionNumFirst = 1
const partitionNumLast = 4
const partitionBootableValue = 0x80
const partitionNonBootableValue = 0

/*
Read MBR from disk.
Example:
f, _ := os.Open("/dev/sda")
Mbr, err := mbr.Read(f)
if err != nil ...
f.Close()
*/
func Read(disk io.Reader) (*MBR, error) {
	var this *MBR = &MBR{}
	this.bytes = make([]byte, mbrSize)
	_, err := disk.Read(this.bytes)
	if err != nil {
		return this, err
	}

	return this, this.Check()
}

func (this *MBR) Check() error {
	// Check signature
	if this.bytes[mbrSignOffset] != 0x55 || this.bytes[mbrSignOffset+1] != 0xAA {
		return ErrorBadMbrSign
	}

	// Check partitions
	for l := partitionNumFirst; l <= partitionNumLast; l++ {
		lp := this.GetPartition(l)
		if lp.IsEmpty() {
			continue
		}

		// Check if partition last sector out of uint32 bounds
		if uint64(lp.GetLBAStart())+uint64(lp.GetLBALen()) > uint64(0xFFFFFFFF) {
			return ErrorPartitionLastSectorHigh
		}

		// Check partition bootable status
		if lp.bytes[partitionBootableOffset] != partitionBootableValue && lp.bytes[partitionBootableOffset] != partitionNonBootableValue {
			return ErrorPartitionBootFlag
		}

		// Check if partitions have intersections
		for r := partitionNumFirst; r <= partitionNumLast; r++ {
			if l == r {
				continue
			}
			rp := this.GetPartition(r)
			if rp.IsEmpty() {
				continue
			}

			if lp.GetLBAStart() > rp.GetLBAStart() && uint64(lp.GetLBAStart()) < uint64(rp.GetLBAStart())+uint64(rp.GetLBALen()) {
				return ErrorPartitionsIntersection
			}
		}
	}

	return nil
}

func (this *MBR) FixSignature() {
	this.bytes[mbrSignOffset] = 0x55
	this.bytes[mbrSignOffset+1] = 0xAA
}

/*
Write MBR to disk
Example:
f, _ := os.OpenFile("/dev/sda", os.O_RDWR | os.O_SYNC, 0600)
err := Mbr.Write(f)
if err != nil ...
f.Close()
*/
func (this MBR) Write(disk io.Writer) error {
	_, err := disk.Write(this.bytes)
	return err
}

func (this MBR) GetPartition(num int) *MBRPartition {
	if num < partitionNumFirst || num > partitionNumLast {
		return nil
	}

	var part *MBRPartition = &MBRPartition{}
	partStart := mbrFirstPartEntryOffset + (num-1)*mbrPartEntrySize
	part.bytes = this.bytes[partStart : partStart+mbrPartEntrySize]
	return part
}

/*
Return number of first sector of partition. Numbers starts from 1.
*/
func (this *MBRPartition) GetLBAStart() uint32 {
	return readLittleEndianUINT32(this.bytes[partitionLBAStartOffset : partitionLBAStartOffset+4])
}

/*
Return count of sectors in partition.
*/
func (this *MBRPartition) GetLBALen() uint32 {
	return readLittleEndianUINT32(this.bytes[partitionLBALengthOffset : partitionLBALengthOffset+4])
}

/*
Return true if partition have empty type
*/
func (this *MBRPartition) IsEmpty() bool {
	return this.bytes[partitionTypeOffset] == partitionEmptyType
}

/*
Set start sector of partition. Number of sector starts from 1. 0 - invalid value.
*/
func (this *MBRPartition) SetLBAStart(startSector uint32) {
	writeLittleEndianUINT32(this.bytes[partitionLBAStartOffset:partitionLBAStartOffset+4], startSector)
}

/*
Set length of partition in sectors.
*/
func (this *MBRPartition) SetLBALen(sectorCount uint32) {
	writeLittleEndianUINT32(this.bytes[partitionLBALengthOffset:partitionLBALengthOffset+4], sectorCount)
}

func writeLittleEndianUINT32(buf []byte, val uint32) {
	buf[0] = byte(val & 0xFF)
	buf[1] = byte(val >> 8 & 0xFF)
	buf[2] = byte(val >> 16 & 0xFF)
	buf[3] = byte(val >> 24 & 0xFF)
}

func readLittleEndianUINT32(buf []byte) uint32 {
	return uint32(buf[0]) + uint32(buf[1])<<8 + uint32(buf[2])<<16 + uint32(buf[3])<<24
}
