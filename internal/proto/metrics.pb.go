package proto

import (
	proto "google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

type Metric_MType int32

const (
	Metric_GAUGE   Metric_MType = 0
	Metric_COUNTER Metric_MType = 1
)

var (
	Metric_MType_name = map[int32]string{
		0: "GAUGE",
		1: "COUNTER",
	}
	Metric_MType_value = map[string]int32{
		"GAUGE":   0,
		"COUNTER": 1,
	}
)

func (x Metric_MType) Enum() *Metric_MType {
	p := new(Metric_MType)
	*p = x
	return p
}

func (x Metric_MType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Metric_MType) Descriptor() protoreflect.EnumDescriptor {
	return file_metrics_proto_enumTypes[0].Descriptor()
}

func (Metric_MType) Type() protoreflect.EnumType {
	return &file_metrics_proto_enumTypes[0]
}

func (x Metric_MType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

type Metric struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id    string       `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Type  Metric_MType `protobuf:"varint,2,opt,name=type,proto3,enum=metrics.Metric_MType" json:"type,omitempty"`
	Delta int64        `protobuf:"varint,3,opt,name=delta,proto3" json:"delta,omitempty"`
	Value float64      `protobuf:"fixed64,4,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *Metric) Reset() {
	*x = Metric{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metrics_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metric) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metric) ProtoMessage() {}

func (x *Metric) ProtoReflect() protoreflect.Message {
	mi := &file_metrics_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Metric) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Metric) GetType() Metric_MType {
	if x != nil {
		return x.Type
	}
	return Metric_GAUGE
}

func (x *Metric) GetDelta() int64 {
	if x != nil {
		return x.Delta
	}
	return 0
}

func (x *Metric) GetValue() float64 {
	if x != nil {
		return x.Value
	}
	return 0
}

type UpdateMetricsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metrics []*Metric `protobuf:"bytes,1,rep,name=metrics,proto3" json:"metrics,omitempty"`
}

func (x *UpdateMetricsRequest) Reset() {
	*x = UpdateMetricsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metrics_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateMetricsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateMetricsRequest) ProtoMessage() {}

func (x *UpdateMetricsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_metrics_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *UpdateMetricsRequest) GetMetrics() []*Metric {
	if x != nil {
		return x.Metrics
	}
	return nil
}

type UpdateMetricsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *UpdateMetricsResponse) Reset() {
	*x = UpdateMetricsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metrics_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpdateMetricsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateMetricsResponse) ProtoMessage() {}

func (x *UpdateMetricsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_metrics_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

var File_metrics_proto protoreflect.FileDescriptor

var file_metrics_proto_rawDescOnce sync.Once
var file_metrics_proto_rawDescData []byte

func file_metrics_proto_rawDesc() []byte {
	file_metrics_proto_rawDescOnce.Do(func() {
		file_metrics_proto_rawDescData = buildMetricsFileDesc()
	})
	return file_metrics_proto_rawDescData
}

func buildMetricsFileDesc() []byte {
	syntax := "proto3"
	name := "metrics.proto"
	pkg := "metrics"
	goPkg := "go-metrics-and-alerts/internal/proto;proto"

	typeNameEnum := ".metrics.Metric.MType"
	typeNameMetric := ".metrics.Metric"

	fd := &descriptorpb.FileDescriptorProto{
		Syntax:  &syntax,
		Name:    &name,
		Package: &pkg,
		Options: &descriptorpb.FileOptions{GoPackage: &goPkg},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Metric"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("id"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: proto.String("type"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(), TypeName: &typeNameEnum},
					{Name: proto.String("delta"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum()},
					{Name: proto.String("value"), Number: proto.Int32(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE.Enum()},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: proto.String("MType"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: proto.String("GAUGE"), Number: proto.Int32(0)},
							{Name: proto.String("COUNTER"), Number: proto.Int32(1)},
						},
					},
				},
			},
			{
				Name: proto.String("UpdateMetricsRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: proto.String("metrics"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: &typeNameMetric},
				},
			},
			{
				Name: proto.String("UpdateMetricsResponse"),
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: proto.String("Metrics"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       proto.String("UpdateMetrics"),
						InputType:  proto.String(".metrics.UpdateMetricsRequest"),
						OutputType: proto.String(".metrics.UpdateMetricsResponse"),
					},
				},
			},
		},
	}

	data, _ := proto.Marshal(fd)
	return data
}

var file_metrics_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_metrics_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_metrics_proto_goTypes = []any{
	(*Metric)(nil),
	(*UpdateMetricsRequest)(nil),
	(*UpdateMetricsResponse)(nil),
	(Metric_MType)(0),
}
var file_metrics_proto_depIdxs = []int32{
	0, // 0: metrics.UpdateMetricsRequest.metrics:type_name -> metrics.Metric
	3, // 1: metrics.Metric.type:type_name -> metrics.Metric.MType
	1, // 2: metrics.Metrics.UpdateMetrics:input_type -> metrics.UpdateMetricsRequest
	2, // 3: metrics.Metrics.UpdateMetrics:output_type -> metrics.UpdateMetricsResponse
}

func init() { file_metrics_proto_init() }
func file_metrics_proto_init() {
	if File_metrics_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_metrics_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*Metric); i {
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
		file_metrics_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*UpdateMetricsRequest); i {
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
		file_metrics_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*UpdateMetricsResponse); i {
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
			RawDescriptor: file_metrics_proto_rawDesc(),
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_metrics_proto_goTypes,
		DependencyIndexes: file_metrics_proto_depIdxs,
		EnumInfos:         file_metrics_proto_enumTypes,
		MessageInfos:      file_metrics_proto_msgTypes,
	}.Build()
	File_metrics_proto = out.File
	file_metrics_proto_rawDescData = nil
	file_metrics_proto_goTypes = nil
	file_metrics_proto_depIdxs = nil
}
