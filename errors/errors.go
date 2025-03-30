package errors

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

// ErrorCode 定义错误码类型
type ErrorCode int32

const (
	// 系统级错误码
	Unknown         ErrorCode = 10000
	Internal        ErrorCode = 10001
	InvalidArgument ErrorCode = 10002
	NotFound        ErrorCode = 10003
	Unauthorized    ErrorCode = 10004
	Timeout         ErrorCode = 10005

	// 业务级错误码 (11000-19999)
	BusinessError ErrorCode = 11000
)

// Error 定义错误结构
type Error struct {
	Code    ErrorCode
	Message string
	Details []proto.Message
}

func (e *Error) Error() string {
	return fmt.Sprintf("error: code = %d message = %s", e.Code, e.Message)
}

// New 创建新的错误
func New(code ErrorCode, message string) error {
	return &Error{
		Code:    code,
		Message: message,
		Details: make([]proto.Message, 0),
	}
}

// WithDetails 添加错误详情
func WithDetails(err error, details ...proto.Message) error {
	var e *Error
	if errors.As(err, &e) {
		e.Details = append(e.Details, details...)
		return e
	}
	return err
}

// FromError 从普通错误转换为 gRPC 错误
func FromError(err error) error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		st := status.New(codes.Code(e.Code), e.Message)
		if len(e.Details) > 0 {
			details := make([]protoadapt.MessageV1, 0, len(e.Details))
			for _, d := range e.Details {
				if msg := protoadapt.MessageV1Of(d); msg != nil {
					details = append(details, msg)
				}
			}
			if std, err := st.WithDetails(details...); err == nil {
				return std.Err()
			}
		}
		return st.Err()
	}

	return status.Error(codes.Unknown, err.Error())
}

// ToError 从 gRPC 错误转换为普通错误
func ToError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return New(Unknown, err.Error())
	}

	e := &Error{
		Code:    ErrorCode(st.Code()),
		Message: st.Message(),
		Details: make([]proto.Message, 0, len(st.Details())),
	}

	for _, detail := range st.Details() {
		if msg, ok := detail.(proto.Message); ok {
			e.Details = append(e.Details, msg)
		}
	}

	return e
}

func ErrorHandlingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			// 确保返回标准化的错误
			return nil, FromError(err)
		}
		return resp, nil
	}
}
