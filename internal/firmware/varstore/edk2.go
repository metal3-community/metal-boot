package varstore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"

	"github.com/bmcpi/pibmc/internal/firmware/efi"
	"github.com/go-logr/logr"
)

type Edk2VarStore struct {
	filename string
	filedata []byte
	start    int
	end      int

	Logger logr.Logger
}

func NewEdk2VarStore(filename string) *Edk2VarStore {
	vs := &Edk2VarStore{filename: filename}
	vs.readFile()
	vs.parseVolume()
	return vs
}

func (vs *Edk2VarStore) GetVarList() (efi.EfiVarList, error) {
	pos := vs.start
	varlist := efi.EfiVarList{}
	for pos < vs.end {
		magic := binary.LittleEndian.Uint16(vs.filedata[pos:])
		if magic != 0x55aa {
			break
		}
		state := vs.filedata[pos+2]
		attr := binary.LittleEndian.Uint32(vs.filedata[pos+4:])
		count := binary.LittleEndian.Uint64(vs.filedata[pos+8:])

		pk := binary.LittleEndian.Uint32(vs.filedata[pos+32:])
		nsize := binary.LittleEndian.Uint32(vs.filedata[pos+36:])
		dsize := binary.LittleEndian.Uint32(vs.filedata[pos+40:])

		if state == 0x3f {
			varName := efi.FromUCS16(vs.filedata[pos+44+16:])
			varData := vs.filedata[uint32(pos)+44+16+nsize : uint32(pos)+44+16+nsize+dsize]
			varItem := efi.EfiVar{
				Name:  varName,
				Guid:  efi.ParseBinGUID(vs.filedata, pos+44),
				Attr:  attr,
				Data:  varData,
				Count: int(count),
				PkIdx: int(pk),
			}
			varItem.ParseTime(vs.filedata, pos+16)
			varlist[varItem.Name.String()] = &varItem
		}

		pos += 44 + 16 + int(nsize) + int(dsize)
		pos = (pos + 3) & ^3 // align
	}
	return varlist, nil
}

func (vs *Edk2VarStore) WriteVarStore(filename string, varlist efi.EfiVarList) error {
	vs.Logger.Info("writing raw edk2 varstore to %s", filename)
	blob, err := vs.bytesVarStore(varlist)
	if err != nil {
		vs.Logger.Error(err, "failed to convert varlist to bytes")
		return err
	}

	if err := os.WriteFile(filename, blob, 0o644); err != nil {
		vs.Logger.Error(err, "failed to write file", "filename", filename)
		return err
	}
	return nil
}

func (vs *Edk2VarStore) findNvData(data []byte) int {
	offset := 0
	for offset+64 < len(data) {
		guid := efi.ParseBinGUID(data, offset+16)
		if guid.String() == efi.NvData {
			return offset
		}
		if guid.String() == efi.Ffs {
			tlen := binary.LittleEndian.Uint64(data[offset+32 : offset+40])
			offset += int(tlen)
			continue
		}
		offset += 1024
	}
	return -1
}

func (vs *Edk2VarStore) readFile() error {
	vs.Logger.Info("reading raw edk2 varstore from %s", vs.filename)
	data, err := os.ReadFile(vs.filename)
	if err != nil {
		vs.Logger.Error(err, "failed to read file", "filename", vs.filename)
		return err
	}
	vs.filedata = data
	return nil
}

func (e *Edk2VarStore) parseVolume() error {
	offset := e.findNvData(e.filedata)
	if offset < 1 {
		return fmt.Errorf("%s: varstore not found", e.filename)
	}

	guid := efi.ParseBinGUID(e.filedata, offset+16)

	// Equivalent to struct.unpack_from("=QLLHHHxBLL", self.filedata, offset + 32)
	r := bytes.NewReader(e.filedata[offset+32:])

	var vlen uint64
	var sig, attr uint32
	var hlen, csum, xoff uint16
	var rev uint8
	var blocks, blksize uint32

	// Read in same order as Python struct unpacking
	if err := binary.Read(r, binary.LittleEndian, &vlen); err != nil {
		return fmt.Errorf("failed to read vlen: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &sig); err != nil {
		return fmt.Errorf("failed to read sig: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &attr); err != nil {
		return fmt.Errorf("failed to read attr: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &hlen); err != nil {
		return fmt.Errorf("failed to read hlen: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &csum); err != nil {
		return fmt.Errorf("failed to read csum: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &xoff); err != nil {
		return fmt.Errorf("failed to read xoff: %w", err)
	}

	// Skip the pad byte (equivalent to 'x' in struct format)
	if _, err := r.Seek(1, io.SeekCurrent); err != nil {
		return fmt.Errorf("failed to skip pad byte: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &rev); err != nil {
		return fmt.Errorf("failed to read rev: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &blocks); err != nil {
		return fmt.Errorf("failed to read blocks: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &blksize); err != nil {
		return fmt.Errorf("failed to read blksize: %w", err)
	}

	e.Logger.Info("vol=%s vlen=0x%x rev=%d blocks=%d*%d (0x%x)",
		efi.GuidName(guid), vlen, rev, blocks, blksize, blocks*blksize)

	if sig != 0x4856465f {
		err := fmt.Errorf("%s: invalid signature", e.filename)
		e.Logger.Error(err, "sig", sig)
		return err
	}

	if guid.String() != efi.NvData {
		err := fmt.Errorf("%s: not a volume", e.filename)
		e.Logger.Error(err, "guid", guid)
		return err
	}

	return e.parseVarstore(offset + int(hlen))
}

func (vs *Edk2VarStore) parseVarstore(start int) error {
	guid := efi.ParseBinGUID(vs.filedata, start)
	size := binary.LittleEndian.Uint32(vs.filedata[start+16 : start+20])
	storefmt := vs.filedata[start+20]
	state := vs.filedata[start+21]

	vs.Logger.Info("varstore=%s size=0x%x format=0x%x state=0x%x",
		efi.GuidName(guid), size, storefmt, state)

	if guid.String() != efi.AuthVars {
		return fmt.Errorf("%s: unknown varstore guid", vs.filename)
	}
	if storefmt != 0x5a {
		return fmt.Errorf("%s: unknown varstore format", vs.filename)
	}
	if state != 0xfe {
		return fmt.Errorf("%s: unknown varstore state", vs.filename)
	}

	vs.start = start + 16 + 12
	vs.end = start + int(size)
	vs.Logger.Info("var store range: 0x%x -> 0x%x", vs.start, vs.end)
	return nil
}

// BytesVar converts an EFI variable to its binary representation
func (vs *Edk2VarStore) bytesVar(v *efi.EfiVar) []byte {
	// Allocate a buffer for the binary data
	buf := new(bytes.Buffer)

	// Equivalent to struct.pack("=HBxLQ", 0x55aa, 0x3f, var.attr, var.count)
	binary.Write(buf, binary.LittleEndian, uint16(0x55aa))
	binary.Write(buf, binary.LittleEndian, uint8(0x3f))
	binary.Write(buf, binary.LittleEndian, uint8(0)) // padding byte (x)
	binary.Write(buf, binary.LittleEndian, uint32(v.Attr))
	binary.Write(buf, binary.LittleEndian, uint64(v.Count))

	// Append time bytes
	timeBytes := v.BytesTime()
	buf.Write(timeBytes)

	// Equivalent to struct.pack("=LLL", var.pkidx, var.name.size(), len(var.data))
	binary.Write(buf, binary.LittleEndian, uint32(v.PkIdx))
	binary.Write(buf, binary.LittleEndian, uint32(v.Name.Size()))
	binary.Write(buf, binary.LittleEndian, uint32(len(v.Data)))

	// Append GUID bytes in little-endian format
	buf.Write(v.Guid.BytesLE())

	// Append name bytes
	buf.Write(v.Name.Bytes())

	// Append data bytes
	buf.Write(v.Data)

	// Pad to 4-byte boundary with 0xFF bytes
	blob := buf.Bytes()
	padding := (4 - len(blob)%4) % 4
	for range padding {
		blob = append(blob, 0xFF)
	}

	return blob
}

func (vs *Edk2VarStore) bytesVarList(varlist efi.EfiVarList) ([]byte, error) {
	blob := []byte{}
	keys := make([]string, 0, len(varlist))
	for k := range varlist {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		blob = append(blob, vs.bytesVar(varlist[key])...)
	}
	if len(blob) > vs.end-vs.start {
		err := fmt.Errorf("varstore is too small: %d > %d", len(blob), vs.end-vs.start)
		vs.Logger.Error(err, "size", len(blob), "max", vs.end-vs.start)
		return nil, err
	}
	return blob, nil
}

func (vs *Edk2VarStore) bytesVarStore(varlist efi.EfiVarList) ([]byte, error) {
	blob := slices.Clone(vs.filedata[:vs.start])

	// Append the variable list
	newVarList, err := vs.bytesVarList(varlist)
	if err != nil {
		vs.Logger.Error(err, "failed to convert varlist to bytes")
		return nil, err
	}

	blob = append(blob, newVarList...)
	for len(blob) < vs.end {
		blob = append(blob, 0xff)
	}
	blob = append(blob, vs.filedata[vs.end:]...)
	return blob, nil
}
