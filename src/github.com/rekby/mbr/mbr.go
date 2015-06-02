package parttable

import (
	"io"
	"errors"
)

var ErrorBadMbrSign = errors.New("MBR: Bad signature")
var ErrorPartitionsIntersection = errors.New("MBR: Partitions have intersections")
var ErrorPartitionLastSectorHigh = errors.New("MBR: Last sector have very high number")

type MBR struct {
	bytes []byte
}

type MBRPartition struct{
	bytes []byte
}

const mbrFirstPartEntryOffset = 446 // bytes
const mbrPartEntrySize = 16 // bytes
const mbrSize = 512 // bytes
const mbrSignOffset = 510 // bytes

const partitionTypeOffset = 4 // bytes
const partitionLBAStartOffset = 8 // bytes
const partitionLBALengthOffset = 12 // bytes

const partitionEmptyType = 0
const partitionNumFirst = 1
const partitionNumLast = 4

/*
Read MBR from disk.
Example:
f, _ := os.Open("/dev/sda")
Mbr, err := mbr.Read(f)
if err != nil ...
f.Close()
 */
func Read(disk io.ReaderAt) (*MBR, error) {
	var this *MBR = &MBR{}
	this.bytes = make([]byte, mbrSize)
	_, err := disk.ReadAt(this.bytes, 0)
	if err != nil {
		return this, err
	}

	return this, this.Check()
}

func (this *MBR) Check() error {
	// Check signature
	if this.bytes[mbrSignOffset] != 0x55 || this.bytes[mbrSignOffset+1] != 0xAA{
		return ErrorBadMbrSign
	}

	// Check if partitions have intersections
	for l := partitionNumFirst; l <= partitionNumLast; l++ {
		lp := this.GetPartition(l)
		if lp.IsEmpty(){
			continue
		}
		for r := partitionNumFirst; r <= partitionNumLast; r++ {
			if l == r {
				continue
			}
			rp := this.GetPartition(r)
			if rp.IsEmpty() {
				continue
			}

			if lp.GetLBAStart() > rp.GetLBAStart() && uint64(lp.GetLBAStart()) < uint64(rp.GetLBAStart()) + uint64(rp.GetLBALen()){
				return ErrorPartitionsIntersection
			}
		}
	}

	// Check if last partition sector have more then 32 uint number
	for i := partitionNumFirst; i <= partitionNumLast; i++ {
		part := this.GetPartition(i)
		if part.IsEmpty() {
			continue
		}
		if uint64(part.GetLBAStart()) + uint64(part.GetLBALen()) > uint64(0xFFFFFFFF) {
			return ErrorPartitionLastSectorHigh
		}
	}
	return nil
}

func (this *MBR) FixSignature(){
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
func (this MBR) Write(disk io.WriterAt) error {
	_, err := disk.WriteAt(this.bytes, 0)
	return err
}

func (this MBR) GetPartition(num int) *MBRPartition{
	if num < partitionNumFirst || num > partitionNumLast {
		return nil
	}

	var part MBRPartition
	partStart := mbrFirstPartEntryOffset + (num-1) * mbrPartEntrySize
	part.bytes = this.bytes[partStart:partStart + mbrPartEntrySize]
	return part
}

/*
Return number of first sector of partition. Numbers starts from 1.
 */
func (this *MBRPartition) GetLBAStart() uint32{
	return readLittleEndianUINT32(this.bytes[partitionLBAStartOffset:partitionLBAStartOffset+4])
}

/*
Return count of sectors in partition.
 */
func (this *MBRPartition) GetLBALen() uint32{
	return readLittleEndianUINT32(this.bytes[partitionLBALengthOffset:partitionLBALengthOffset+4])
}

/*
Return true if partition have empty type
 */
func (this *MBRPartition) IsEmpty()bool{
	return this.bytes[partitionTypeOffset] == partitionEmptyType
}

/*
Set start sector of partition. Number of sector starts from 1. 0 - invalid value.
 */
func (this *MBRPartition) SetLBAStart(startSector uint32){
	writeLittleEndianUINT32(this.bytes[partitionLBAStartOffset:partitionLBAStartOffset+4], startSector)
}

/*
Set length of partition in sectors.
 */
func (this *MBRPartition) SetLBALen(sectorCount uint32){
	writeLittleEndianUINT32(this.bytes[partitionLBALengthOffset:partitionLBALengthOffset+4], sectorCount)
}

func writeLittleEndianUINT32(buf []byte, val uint32){
	buf[0] = val & 0xFF
	buf[1] = val >> 8 & 0xFF
	buf[2] = val >> 16 & 0xFF
	buf[3] = val >> 24 & 0xFF
}

func readLittleEndianUINT32(buf []byte) uint32 {
	return byte[0] + byte[1] << 8 + byte[2] << 16 + byte[3] << 24
}