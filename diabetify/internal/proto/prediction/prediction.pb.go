// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v5.29.3
// source: internal/proto/prediction/prediction.proto

package prediction

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PredictionRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Features      []float64              `protobuf:"fixed64,1,rep,packed,name=features,proto3" json:"features,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PredictionRequest) Reset() {
	*x = PredictionRequest{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PredictionRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PredictionRequest) ProtoMessage() {}

func (x *PredictionRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PredictionRequest.ProtoReflect.Descriptor instead.
func (*PredictionRequest) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{0}
}

func (x *PredictionRequest) GetFeatures() []float64 {
	if x != nil {
		return x.Features
	}
	return nil
}

type FeatureExplanation struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Contribution  float64                `protobuf:"fixed64,1,opt,name=contribution,proto3" json:"contribution,omitempty"`
	Impact        int32                  `protobuf:"varint,2,opt,name=impact,proto3" json:"impact,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FeatureExplanation) Reset() {
	*x = FeatureExplanation{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FeatureExplanation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FeatureExplanation) ProtoMessage() {}

func (x *FeatureExplanation) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FeatureExplanation.ProtoReflect.Descriptor instead.
func (*FeatureExplanation) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{1}
}

func (x *FeatureExplanation) GetContribution() float64 {
	if x != nil {
		return x.Contribution
	}
	return 0
}

func (x *FeatureExplanation) GetImpact() int32 {
	if x != nil {
		return x.Impact
	}
	return 0
}

type PredictionResponse struct {
	state         protoimpl.MessageState         `protogen:"open.v1"`
	Prediction    float64                        `protobuf:"fixed64,1,opt,name=prediction,proto3" json:"prediction,omitempty"`
	Explanation   map[string]*FeatureExplanation `protobuf:"bytes,2,rep,name=explanation,proto3" json:"explanation,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	ElapsedTime   float64                        `protobuf:"fixed64,3,opt,name=elapsed_time,json=elapsedTime,proto3" json:"elapsed_time,omitempty"`
	Timestamp     string                         `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PredictionResponse) Reset() {
	*x = PredictionResponse{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PredictionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PredictionResponse) ProtoMessage() {}

func (x *PredictionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PredictionResponse.ProtoReflect.Descriptor instead.
func (*PredictionResponse) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{2}
}

func (x *PredictionResponse) GetPrediction() float64 {
	if x != nil {
		return x.Prediction
	}
	return 0
}

func (x *PredictionResponse) GetExplanation() map[string]*FeatureExplanation {
	if x != nil {
		return x.Explanation
	}
	return nil
}

func (x *PredictionResponse) GetElapsedTime() float64 {
	if x != nil {
		return x.ElapsedTime
	}
	return 0
}

func (x *PredictionResponse) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

type UpdateModelRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	XNew          []*FeatureVector       `protobuf:"bytes,1,rep,name=X_new,json=XNew,proto3" json:"X_new,omitempty"`
	YNew          []float64              `protobuf:"fixed64,2,rep,packed,name=y_new,json=yNew,proto3" json:"y_new,omitempty"`
	XVal          []*FeatureVector       `protobuf:"bytes,3,rep,name=X_val,json=XVal,proto3" json:"X_val,omitempty"`
	YVal          []float64              `protobuf:"fixed64,4,rep,packed,name=y_val,json=yVal,proto3" json:"y_val,omitempty"`
	Epochs        int32                  `protobuf:"varint,5,opt,name=epochs,proto3" json:"epochs,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdateModelRequest) Reset() {
	*x = UpdateModelRequest{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateModelRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateModelRequest) ProtoMessage() {}

func (x *UpdateModelRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateModelRequest.ProtoReflect.Descriptor instead.
func (*UpdateModelRequest) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{3}
}

func (x *UpdateModelRequest) GetXNew() []*FeatureVector {
	if x != nil {
		return x.XNew
	}
	return nil
}

func (x *UpdateModelRequest) GetYNew() []float64 {
	if x != nil {
		return x.YNew
	}
	return nil
}

func (x *UpdateModelRequest) GetXVal() []*FeatureVector {
	if x != nil {
		return x.XVal
	}
	return nil
}

func (x *UpdateModelRequest) GetYVal() []float64 {
	if x != nil {
		return x.YVal
	}
	return nil
}

func (x *UpdateModelRequest) GetEpochs() int32 {
	if x != nil {
		return x.Epochs
	}
	return 0
}

type FeatureVector struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Values        []float64              `protobuf:"fixed64,1,rep,packed,name=values,proto3" json:"values,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FeatureVector) Reset() {
	*x = FeatureVector{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FeatureVector) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FeatureVector) ProtoMessage() {}

func (x *FeatureVector) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FeatureVector.ProtoReflect.Descriptor instead.
func (*FeatureVector) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{4}
}

func (x *FeatureVector) GetValues() []float64 {
	if x != nil {
		return x.Values
	}
	return nil
}

type UpdateModelResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Status        string                 `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	AucBefore     float64                `protobuf:"fixed64,2,opt,name=auc_before,json=aucBefore,proto3" json:"auc_before,omitempty"`
	AucAfter      float64                `protobuf:"fixed64,3,opt,name=auc_after,json=aucAfter,proto3" json:"auc_after,omitempty"`
	PrAucBefore   float64                `protobuf:"fixed64,4,opt,name=pr_auc_before,json=prAucBefore,proto3" json:"pr_auc_before,omitempty"`
	PrAucAfter    float64                `protobuf:"fixed64,5,opt,name=pr_auc_after,json=prAucAfter,proto3" json:"pr_auc_after,omitempty"`
	ElapsedTime   float64                `protobuf:"fixed64,6,opt,name=elapsed_time,json=elapsedTime,proto3" json:"elapsed_time,omitempty"`
	Timestamp     string                 `protobuf:"bytes,7,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UpdateModelResponse) Reset() {
	*x = UpdateModelResponse{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UpdateModelResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpdateModelResponse) ProtoMessage() {}

func (x *UpdateModelResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpdateModelResponse.ProtoReflect.Descriptor instead.
func (*UpdateModelResponse) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{5}
}

func (x *UpdateModelResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *UpdateModelResponse) GetAucBefore() float64 {
	if x != nil {
		return x.AucBefore
	}
	return 0
}

func (x *UpdateModelResponse) GetAucAfter() float64 {
	if x != nil {
		return x.AucAfter
	}
	return 0
}

func (x *UpdateModelResponse) GetPrAucBefore() float64 {
	if x != nil {
		return x.PrAucBefore
	}
	return 0
}

func (x *UpdateModelResponse) GetPrAucAfter() float64 {
	if x != nil {
		return x.PrAucAfter
	}
	return 0
}

func (x *UpdateModelResponse) GetElapsedTime() float64 {
	if x != nil {
		return x.ElapsedTime
	}
	return 0
}

func (x *UpdateModelResponse) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

type HealthCheckRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HealthCheckRequest) Reset() {
	*x = HealthCheckRequest{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HealthCheckRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HealthCheckRequest) ProtoMessage() {}

func (x *HealthCheckRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HealthCheckRequest.ProtoReflect.Descriptor instead.
func (*HealthCheckRequest) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{6}
}

type HealthCheckResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Status        string                 `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	Timestamp     string                 `protobuf:"bytes,2,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *HealthCheckResponse) Reset() {
	*x = HealthCheckResponse{}
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *HealthCheckResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HealthCheckResponse) ProtoMessage() {}

func (x *HealthCheckResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_proto_prediction_prediction_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HealthCheckResponse.ProtoReflect.Descriptor instead.
func (*HealthCheckResponse) Descriptor() ([]byte, []int) {
	return file_internal_proto_prediction_prediction_proto_rawDescGZIP(), []int{7}
}

func (x *HealthCheckResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *HealthCheckResponse) GetTimestamp() string {
	if x != nil {
		return x.Timestamp
	}
	return ""
}

var File_internal_proto_prediction_prediction_proto protoreflect.FileDescriptor

const file_internal_proto_prediction_prediction_proto_rawDesc = "" +
	"\n" +
	"*internal/proto/prediction/prediction.proto\x12\n" +
	"prediction\"/\n" +
	"\x11PredictionRequest\x12\x1a\n" +
	"\bfeatures\x18\x01 \x03(\x01R\bfeatures\"P\n" +
	"\x12FeatureExplanation\x12\"\n" +
	"\fcontribution\x18\x01 \x01(\x01R\fcontribution\x12\x16\n" +
	"\x06impact\x18\x02 \x01(\x05R\x06impact\"\xa8\x02\n" +
	"\x12PredictionResponse\x12\x1e\n" +
	"\n" +
	"prediction\x18\x01 \x01(\x01R\n" +
	"prediction\x12Q\n" +
	"\vexplanation\x18\x02 \x03(\v2/.prediction.PredictionResponse.ExplanationEntryR\vexplanation\x12!\n" +
	"\felapsed_time\x18\x03 \x01(\x01R\velapsedTime\x12\x1c\n" +
	"\ttimestamp\x18\x04 \x01(\tR\ttimestamp\x1a^\n" +
	"\x10ExplanationEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x124\n" +
	"\x05value\x18\x02 \x01(\v2\x1e.prediction.FeatureExplanationR\x05value:\x028\x01\"\xb6\x01\n" +
	"\x12UpdateModelRequest\x12.\n" +
	"\x05X_new\x18\x01 \x03(\v2\x19.prediction.FeatureVectorR\x04XNew\x12\x13\n" +
	"\x05y_new\x18\x02 \x03(\x01R\x04yNew\x12.\n" +
	"\x05X_val\x18\x03 \x03(\v2\x19.prediction.FeatureVectorR\x04XVal\x12\x13\n" +
	"\x05y_val\x18\x04 \x03(\x01R\x04yVal\x12\x16\n" +
	"\x06epochs\x18\x05 \x01(\x05R\x06epochs\"'\n" +
	"\rFeatureVector\x12\x16\n" +
	"\x06values\x18\x01 \x03(\x01R\x06values\"\xf0\x01\n" +
	"\x13UpdateModelResponse\x12\x16\n" +
	"\x06status\x18\x01 \x01(\tR\x06status\x12\x1d\n" +
	"\n" +
	"auc_before\x18\x02 \x01(\x01R\taucBefore\x12\x1b\n" +
	"\tauc_after\x18\x03 \x01(\x01R\baucAfter\x12\"\n" +
	"\rpr_auc_before\x18\x04 \x01(\x01R\vprAucBefore\x12 \n" +
	"\fpr_auc_after\x18\x05 \x01(\x01R\n" +
	"prAucAfter\x12!\n" +
	"\felapsed_time\x18\x06 \x01(\x01R\velapsedTime\x12\x1c\n" +
	"\ttimestamp\x18\a \x01(\tR\ttimestamp\"\x14\n" +
	"\x12HealthCheckRequest\"K\n" +
	"\x13HealthCheckResponse\x12\x16\n" +
	"\x06status\x18\x01 \x01(\tR\x06status\x12\x1c\n" +
	"\ttimestamp\x18\x02 \x01(\tR\ttimestamp2\x83\x02\n" +
	"\x11PredictionService\x12J\n" +
	"\aPredict\x12\x1d.prediction.PredictionRequest\x1a\x1e.prediction.PredictionResponse\"\x00\x12P\n" +
	"\vUpdateModel\x12\x1e.prediction.UpdateModelRequest\x1a\x1f.prediction.UpdateModelResponse\"\x00\x12P\n" +
	"\vHealthCheck\x12\x1e.prediction.HealthCheckRequest\x1a\x1f.prediction.HealthCheckResponse\"\x00B%Z#diabetify/internal/proto/predictionb\x06proto3"

var (
	file_internal_proto_prediction_prediction_proto_rawDescOnce sync.Once
	file_internal_proto_prediction_prediction_proto_rawDescData []byte
)

func file_internal_proto_prediction_prediction_proto_rawDescGZIP() []byte {
	file_internal_proto_prediction_prediction_proto_rawDescOnce.Do(func() {
		file_internal_proto_prediction_prediction_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_internal_proto_prediction_prediction_proto_rawDesc), len(file_internal_proto_prediction_prediction_proto_rawDesc)))
	})
	return file_internal_proto_prediction_prediction_proto_rawDescData
}

var file_internal_proto_prediction_prediction_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_internal_proto_prediction_prediction_proto_goTypes = []any{
	(*PredictionRequest)(nil),   // 0: prediction.PredictionRequest
	(*FeatureExplanation)(nil),  // 1: prediction.FeatureExplanation
	(*PredictionResponse)(nil),  // 2: prediction.PredictionResponse
	(*UpdateModelRequest)(nil),  // 3: prediction.UpdateModelRequest
	(*FeatureVector)(nil),       // 4: prediction.FeatureVector
	(*UpdateModelResponse)(nil), // 5: prediction.UpdateModelResponse
	(*HealthCheckRequest)(nil),  // 6: prediction.HealthCheckRequest
	(*HealthCheckResponse)(nil), // 7: prediction.HealthCheckResponse
	nil,                         // 8: prediction.PredictionResponse.ExplanationEntry
}
var file_internal_proto_prediction_prediction_proto_depIdxs = []int32{
	8, // 0: prediction.PredictionResponse.explanation:type_name -> prediction.PredictionResponse.ExplanationEntry
	4, // 1: prediction.UpdateModelRequest.X_new:type_name -> prediction.FeatureVector
	4, // 2: prediction.UpdateModelRequest.X_val:type_name -> prediction.FeatureVector
	1, // 3: prediction.PredictionResponse.ExplanationEntry.value:type_name -> prediction.FeatureExplanation
	0, // 4: prediction.PredictionService.Predict:input_type -> prediction.PredictionRequest
	3, // 5: prediction.PredictionService.UpdateModel:input_type -> prediction.UpdateModelRequest
	6, // 6: prediction.PredictionService.HealthCheck:input_type -> prediction.HealthCheckRequest
	2, // 7: prediction.PredictionService.Predict:output_type -> prediction.PredictionResponse
	5, // 8: prediction.PredictionService.UpdateModel:output_type -> prediction.UpdateModelResponse
	7, // 9: prediction.PredictionService.HealthCheck:output_type -> prediction.HealthCheckResponse
	7, // [7:10] is the sub-list for method output_type
	4, // [4:7] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_internal_proto_prediction_prediction_proto_init() }
func file_internal_proto_prediction_prediction_proto_init() {
	if File_internal_proto_prediction_prediction_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_internal_proto_prediction_prediction_proto_rawDesc), len(file_internal_proto_prediction_prediction_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_internal_proto_prediction_prediction_proto_goTypes,
		DependencyIndexes: file_internal_proto_prediction_prediction_proto_depIdxs,
		MessageInfos:      file_internal_proto_prediction_prediction_proto_msgTypes,
	}.Build()
	File_internal_proto_prediction_prediction_proto = out.File
	file_internal_proto_prediction_prediction_proto_goTypes = nil
	file_internal_proto_prediction_prediction_proto_depIdxs = nil
}
