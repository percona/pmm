// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0-devel
// 	protoc        (unknown)
// source: management/v1/pagination.proto

package managementv1

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// PageParams represents page request parameters for pagination.
type PageParams struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Maximum number of results per page.
	PageSize int32 `protobuf:"varint,1,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	// Index of the requested page, starts from 0.
	Index int32 `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
}

func (x *PageParams) Reset() {
	*x = PageParams{}
	if protoimpl.UnsafeEnabled {
		mi := &file_management_v1_pagination_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageParams) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageParams) ProtoMessage() {}

func (x *PageParams) ProtoReflect() protoreflect.Message {
	mi := &file_management_v1_pagination_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageParams.ProtoReflect.Descriptor instead.
func (*PageParams) Descriptor() ([]byte, []int) {
	return file_management_v1_pagination_proto_rawDescGZIP(), []int{0}
}

func (x *PageParams) GetPageSize() int32 {
	if x != nil {
		return x.PageSize
	}
	return 0
}

func (x *PageParams) GetIndex() int32 {
	if x != nil {
		return x.Index
	}
	return 0
}

// PageTotals represents total values for pagination.
type PageTotals struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Total number of results.
	TotalItems int32 `protobuf:"varint,1,opt,name=total_items,json=totalItems,proto3" json:"total_items,omitempty"`
	// Total number of pages.
	TotalPages int32 `protobuf:"varint,2,opt,name=total_pages,json=totalPages,proto3" json:"total_pages,omitempty"`
}

func (x *PageTotals) Reset() {
	*x = PageTotals{}
	if protoimpl.UnsafeEnabled {
		mi := &file_management_v1_pagination_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PageTotals) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PageTotals) ProtoMessage() {}

func (x *PageTotals) ProtoReflect() protoreflect.Message {
	mi := &file_management_v1_pagination_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PageTotals.ProtoReflect.Descriptor instead.
func (*PageTotals) Descriptor() ([]byte, []int) {
	return file_management_v1_pagination_proto_rawDescGZIP(), []int{1}
}

func (x *PageTotals) GetTotalItems() int32 {
	if x != nil {
		return x.TotalItems
	}
	return 0
}

func (x *PageTotals) GetTotalPages() int32 {
	if x != nil {
		return x.TotalPages
	}
	return 0
}

var File_management_v1_pagination_proto protoreflect.FileDescriptor

var file_management_v1_pagination_proto_rawDesc = []byte{
	0x0a, 0x1e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x2f,
	0x70, 0x61, 0x67, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0d, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x1a,
	0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61,
	0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x5a, 0x0a, 0x0a, 0x50, 0x61, 0x67, 0x65,
	0x50, 0x61, 0x72, 0x61, 0x6d, 0x73, 0x12, 0x24, 0x0a, 0x09, 0x70, 0x61, 0x67, 0x65, 0x5f, 0x73,
	0x69, 0x7a, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x42, 0x07, 0xfa, 0x42, 0x04, 0x1a, 0x02,
	0x20, 0x00, 0x52, 0x08, 0x70, 0x61, 0x67, 0x65, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x26, 0x0a, 0x05,
	0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x42, 0x10, 0xfa, 0x42, 0x0d,
	0x1a, 0x0b, 0x20, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x52, 0x05, 0x69,
	0x6e, 0x64, 0x65, 0x78, 0x22, 0x4e, 0x0a, 0x0a, 0x50, 0x61, 0x67, 0x65, 0x54, 0x6f, 0x74, 0x61,
	0x6c, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x69, 0x74, 0x65, 0x6d,
	0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x49, 0x74,
	0x65, 0x6d, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x70, 0x61, 0x67,
	0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x50,
	0x61, 0x67, 0x65, 0x73, 0x42, 0xb0, 0x01, 0x0a, 0x11, 0x63, 0x6f, 0x6d, 0x2e, 0x6d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x42, 0x0f, 0x50, 0x61, 0x67, 0x69,
	0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x35, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x72, 0x63, 0x6f, 0x6e,
	0x61, 0x2f, 0x70, 0x6d, 0x6d, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65,
	0x6d, 0x65, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x3b, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d, 0x65,
	0x6e, 0x74, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x4d, 0x58, 0x58, 0xaa, 0x02, 0x0d, 0x4d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x0d, 0x4d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x19, 0x4d, 0x61, 0x6e,
	0x61, 0x67, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0e, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x6d,
	0x65, 0x6e, 0x74, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_management_v1_pagination_proto_rawDescOnce sync.Once
	file_management_v1_pagination_proto_rawDescData = file_management_v1_pagination_proto_rawDesc
)

func file_management_v1_pagination_proto_rawDescGZIP() []byte {
	file_management_v1_pagination_proto_rawDescOnce.Do(func() {
		file_management_v1_pagination_proto_rawDescData = protoimpl.X.CompressGZIP(file_management_v1_pagination_proto_rawDescData)
	})
	return file_management_v1_pagination_proto_rawDescData
}

var (
	file_management_v1_pagination_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
	file_management_v1_pagination_proto_goTypes  = []interface{}{
		(*PageParams)(nil), // 0: management.v1.PageParams
		(*PageTotals)(nil), // 1: management.v1.PageTotals
	}
)

var file_management_v1_pagination_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_management_v1_pagination_proto_init() }
func file_management_v1_pagination_proto_init() {
	if File_management_v1_pagination_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_management_v1_pagination_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageParams); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_management_v1_pagination_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PageTotals); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_management_v1_pagination_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_management_v1_pagination_proto_goTypes,
		DependencyIndexes: file_management_v1_pagination_proto_depIdxs,
		MessageInfos:      file_management_v1_pagination_proto_msgTypes,
	}.Build()
	File_management_v1_pagination_proto = out.File
	file_management_v1_pagination_proto_rawDesc = nil
	file_management_v1_pagination_proto_goTypes = nil
	file_management_v1_pagination_proto_depIdxs = nil
}