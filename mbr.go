package mbr

import (
	"errors"
	"io"
)

var ErrorBadMbrSign = errors.New("MBR: Bad signature")
var ErrorPartitionsIntersection = errors.New("MBR: Partitions have intersections")
var ErrorPartitionLastSectorHigh = errors.New("MBR: Last sector have very high number")
var ErrorPartitionBootFlag = errors.New("MBR: Bad value in boot flag")
var ErrorDiskSizeNotEvenSectors = errors.New("MBR: Disk size is not evenly divisible by sector size")
var ErrorInvalidProtectiveType = errors.New("Invalid value for ProtectiveType")

type MBR struct {
	bytes []byte
}

type MBRPartition struct {
	Num int
	bytes []byte
}

type PartitionType byte

const (
	PART_EMPTY              = PartitionType(0)
	PART_LINUX_SWAP_SOLARIS = PartitionType(0x82)
	PART_LVM                = PartitionType(0x8E)
	PART_HYBRID_GPT         = PartitionType(0xED)
	PART_GPT                = PartitionType(0xEE)
)

type ProtectiveType int

const (
	DiskSize          = ProtectiveType(1)
	MaxSize           = ProtectiveType(2)
	DefaultProtective = ProtectiveType(0)
)

const mbrFirstPartEntryOffset = 446 // bytes
const mbrPartEntrySize = 16         // bytes
const mbrSize = 512                 // bytes
const mbrSignOffset = 510           // bytes

const partitionBootableOffset = 0   // bytes
const partitionTypeOffset = 4       // bytes
const partitionLBAStartOffset = 8   // bytes
const partitionLBALengthOffset = 12 // bytes

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
	maxLen := uint64(0xFFFFFFFF)
	for l := partitionNumFirst; l <= partitionNumLast; l++ {
		lp := this.GetPartition(l)
		if lp.IsEmpty() {
			continue
		}

		// Check if partition last sector out of uint32 bounds
		last := uint64(lp.GetLBAStart()) + uint64(lp.GetLBALen())
		if last > maxLen {
			// Most/All GPT partitioners write a length for the Protective MBR of maxLen
			// even though it is strictly out of bounds. Specifically allow for that.
			if !(l == partitionNumFirst && last == maxLen+1 && lp.GetType() == PART_GPT) {
				return ErrorPartitionLastSectorHigh
			}
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

	var part *MBRPartition = &MBRPartition{Num:num}
	partStart := mbrFirstPartEntryOffset + (num-1)*mbrPartEntrySize
	part.bytes = this.bytes[partStart : partStart+mbrPartEntrySize]
	return part
}

func (this MBR) GetAllPartitions() []*MBRPartition {
	res := make([]*MBRPartition, 4)
	for i := 0; i < 4; i++ {
		res[i] = this.GetPartition(i + 1)
	}
	return res
}

func (this MBR) IsGPT() bool {
	for _, part := range this.GetAllPartitions() {
		if part.GetType() == PART_GPT || part.GetType() == PART_HYBRID_GPT {
			return true
		}
	}
	return false
}

// MakeProtective - Make this MBR a GPT Protective MBR
//   sectorSize is either 512 or 4096. diskSize is the size of entire disk in bytes.
//   https://en.wikipedia.org/wiki/GUID_Partition_Table#Protective_MBR_(LBA_0)
//
//  ProtectiveType value determines how the size of the partition is set.
//    DefaultProtective - implementation default value
//    MaxSize - Size of the ProtectiveMBR partition will be set to 0xFFFFFFFF
//        While this is strictly outside the UEFI spec, it is the behavior
//        of linux and windows partitioners.
//    DiskSize - the actual length of the partition size size up to 0xFFFFFFFF - 1
func (this *MBR) MakeProtective(sectorSize int, diskSize uint64, pType ProtectiveType) error {

	if diskSize%uint64(sectorSize) != 0 {
		return ErrorDiskSizeNotEvenSectors
	}
	this.FixSignature()

	ptLBAStart := uint32(1)
	ptLBALen := uint32(0xFFFFFFFF)

	if pType == DiskSize {
		max := uint64(0xFFFFFFFF)
		actual := diskSize/uint64(sectorSize) - uint64(ptLBAStart)
		if actual > max {
			ptLBALen = uint32(max)
		} else {
			ptLBALen = uint32(actual)
		}
	} else if pType != MaxSize && pType != DefaultProtective {
		return ErrorInvalidProtectiveType
	}

	pt := this.GetPartition(1)
	pt.SetType(PART_GPT)
	pt.SetLBAStart(ptLBAStart)
	pt.SetLBALen(ptLBALen)
	pt.bytes[partitionBootableOffset] = partitionNonBootableValue

	// zero the other partitions.
	for pnum := 2; pnum <= 4; pnum++ {
		pt := this.GetPartition(pnum)
		pt.SetType(PART_EMPTY)
		pt.SetLBAStart(0)
		pt.SetLBALen(0)
		pt.bytes[partitionBootableOffset] = partitionNonBootableValue
	}

	return nil
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
Return number of last setor if partition.

If last sector num more then max uint32 - panic. It mean error in metadata.
*/
func (this *MBRPartition) GetLBALast() uint32 {
	last := uint64(this.GetLBAStart()) + uint64(this.GetLBALen()) - 1

	// If last > max uint32 - panic
	if last > uint64(0xFFFFFFFF) {
		panic(errors.New("Overflow while calc last sector. Max sector number in mbr must be less or equal 0xFFFFFFFF"))
	}
	return uint32(last)
}

func (this *MBRPartition) GetType() PartitionType {
	return PartitionType(this.bytes[partitionTypeOffset])
}
func (this *MBRPartition) SetType(t PartitionType) {
	this.bytes[partitionTypeOffset] = byte(t)
}

/*
Return true if partition have empty type
*/
func (this *MBRPartition) IsEmpty() bool {
	return this.GetType() == PART_EMPTY
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

/*
Set the Bootable flag on this partition.
*/
func (this *MBRPartition) SetBootable(bootable bool) {
	if bootable {
		this.bytes[partitionBootableOffset] = partitionBootableValue
	} else {
		this.bytes[partitionBootableOffset] = partitionNonBootableValue
	}
}

/*
Return true if this partition's bootable flag is set.
*/
func (this *MBRPartition) IsBootable() bool {
	return this.bytes[partitionBootableOffset] == partitionBootableValue
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
