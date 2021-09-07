// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: supplyESDT.proto

package esdtSupply

import (
	fmt "fmt"
	github_com_ElrondNetwork_elrond_go_core_data "github.com/ElrondNetwork/elrond-go-core/data"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_big "math/big"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// SupplyESDT is used to store information a shard esdt token supply
type SupplyESDT struct {
	Supply *math_big.Int `protobuf:"bytes,1,opt,name=Supply,proto3,casttypewith=math/big.Int;github.com/ElrondNetwork/elrond-go-core/data.BigIntCaster" json:"value"`
}

func (m *SupplyESDT) Reset()      { *m = SupplyESDT{} }
func (*SupplyESDT) ProtoMessage() {}
func (*SupplyESDT) Descriptor() ([]byte, []int) {
	return fileDescriptor_173c6d56cc05b222, []int{0}
}
func (m *SupplyESDT) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SupplyESDT) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	b = b[:cap(b)]
	n, err := m.MarshalToSizedBuffer(b)
	if err != nil {
		return nil, err
	}
	return b[:n], nil
}
func (m *SupplyESDT) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SupplyESDT.Merge(m, src)
}
func (m *SupplyESDT) XXX_Size() int {
	return m.Size()
}
func (m *SupplyESDT) XXX_DiscardUnknown() {
	xxx_messageInfo_SupplyESDT.DiscardUnknown(m)
}

var xxx_messageInfo_SupplyESDT proto.InternalMessageInfo

func (m *SupplyESDT) GetSupply() *math_big.Int {
	if m != nil {
		return m.Supply
	}
	return nil
}

func init() {
	proto.RegisterType((*SupplyESDT)(nil), "proto.SupplyESDT")
}

func init() { proto.RegisterFile("supplyESDT.proto", fileDescriptor_173c6d56cc05b222) }

var fileDescriptor_173c6d56cc05b222 = []byte{
	// 246 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0x28, 0x2e, 0x2d, 0x28,
	0xc8, 0xa9, 0x74, 0x0d, 0x76, 0x09, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x53,
	0x52, 0xba, 0xe9, 0x99, 0x25, 0x19, 0xa5, 0x49, 0x7a, 0xc9, 0xf9, 0xb9, 0xfa, 0xe9, 0xf9, 0xe9,
	0xf9, 0xfa, 0x60, 0xe1, 0xa4, 0xd2, 0x34, 0x30, 0x0f, 0xcc, 0x01, 0xb3, 0x20, 0xba, 0x94, 0x2a,
	0xb9, 0xb8, 0x82, 0xe1, 0x26, 0x09, 0x65, 0x73, 0xb1, 0x41, 0x78, 0x12, 0x8c, 0x0a, 0x8c, 0x1a,
	0x3c, 0x4e, 0xc1, 0xaf, 0xee, 0xc9, 0xb3, 0x96, 0x25, 0xe6, 0x94, 0xa6, 0xae, 0xba, 0x2f, 0xef,
	0x96, 0x9b, 0x58, 0x92, 0xa1, 0x9f, 0x94, 0x99, 0xae, 0xe7, 0x99, 0x57, 0x62, 0x8d, 0x64, 0x8d,
	0x6b, 0x4e, 0x51, 0x7e, 0x5e, 0x8a, 0x5f, 0x6a, 0x49, 0x79, 0x7e, 0x51, 0xb6, 0x7e, 0x2a, 0x98,
	0xa7, 0x9b, 0x9e, 0xaf, 0x9b, 0x9c, 0x5f, 0x94, 0xaa, 0x9f, 0x92, 0x58, 0x92, 0xa8, 0xe7, 0x94,
	0x99, 0xee, 0x99, 0x57, 0xe2, 0x9c, 0x58, 0x5c, 0x92, 0x5a, 0x14, 0x04, 0xb5, 0xc2, 0xc9, 0xe5,
	0xc2, 0x43, 0x39, 0x86, 0x1b, 0x0f, 0xe5, 0x18, 0x3e, 0x3c, 0x94, 0x63, 0x6c, 0x78, 0x24, 0xc7,
	0xb8, 0xe2, 0x91, 0x1c, 0xe3, 0x89, 0x47, 0x72, 0x8c, 0x17, 0x1e, 0xc9, 0x31, 0xde, 0x78, 0x24,
	0xc7, 0xf8, 0xe0, 0x91, 0x1c, 0xe3, 0x8b, 0x47, 0x72, 0x0c, 0x1f, 0x1e, 0xc9, 0x31, 0x4e, 0x78,
	0x2c, 0xc7, 0x70, 0xe1, 0xb1, 0x1c, 0xc3, 0x8d, 0xc7, 0x72, 0x0c, 0x51, 0x5c, 0xa9, 0xc5, 0x29,
	0x25, 0x10, 0x53, 0x92, 0xd8, 0xc0, 0xfe, 0x30, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x12, 0xd1,
	0xd0, 0x57, 0x11, 0x01, 0x00, 0x00,
}

func (this *SupplyESDT) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*SupplyESDT)
	if !ok {
		that2, ok := that.(SupplyESDT)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	{
		__caster := &github_com_ElrondNetwork_elrond_go_core_data.BigIntCaster{}
		if !__caster.Equal(this.Supply, that1.Supply) {
			return false
		}
	}
	return true
}
func (this *SupplyESDT) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 5)
	s = append(s, "&esdtSupply.SupplyESDT{")
	s = append(s, "Supply: "+fmt.Sprintf("%#v", this.Supply)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringSupplyESDT(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *SupplyESDT) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SupplyESDT) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SupplyESDT) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		__caster := &github_com_ElrondNetwork_elrond_go_core_data.BigIntCaster{}
		size := __caster.Size(m.Supply)
		i -= size
		if _, err := __caster.MarshalTo(m.Supply, dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintSupplyESDT(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintSupplyESDT(dAtA []byte, offset int, v uint64) int {
	offset -= sovSupplyESDT(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *SupplyESDT) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	{
		__caster := &github_com_ElrondNetwork_elrond_go_core_data.BigIntCaster{}
		l = __caster.Size(m.Supply)
		n += 1 + l + sovSupplyESDT(uint64(l))
	}
	return n
}

func sovSupplyESDT(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSupplyESDT(x uint64) (n int) {
	return sovSupplyESDT(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *SupplyESDT) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&SupplyESDT{`,
		`Supply:` + fmt.Sprintf("%v", this.Supply) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringSupplyESDT(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *SupplyESDT) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSupplyESDT
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: SupplyESDT: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SupplyESDT: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Supply", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSupplyESDT
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthSupplyESDT
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthSupplyESDT
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			{
				__caster := &github_com_ElrondNetwork_elrond_go_core_data.BigIntCaster{}
				if tmp, err := __caster.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
					return err
				} else {
					m.Supply = tmp
				}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSupplyESDT(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSupplyESDT
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthSupplyESDT
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipSupplyESDT(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSupplyESDT
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSupplyESDT
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSupplyESDT
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthSupplyESDT
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSupplyESDT
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSupplyESDT
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSupplyESDT        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSupplyESDT          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSupplyESDT = fmt.Errorf("proto: unexpected end of group")
)
