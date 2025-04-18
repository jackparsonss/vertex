package types

type ParamInfo struct {
	Name string
	Type string
}

type FunctionInfo struct {
	Name             string
	Path             string
	Method           string
	Params           []ParamInfo
	ReturnType       string
	IsStruct         bool
	IsSlice          bool
	IsMethod         bool
	ReceiverTypeName string
	StructName       string
	PackageName      string
}
