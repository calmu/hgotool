// Package doris_streamload
//
// ----------------develop info----------------
//
//	@Author xunmuhuang@rastar.com
//	@DateTime 2026-2-5 11:34
//
// --------------------------------------------
package doris_streamload

import "errors"

var (
	ErrInvalidParam            = errors.New("invalid param")
	ErrLimitMaxBytes           = errors.New("batch bytes limit exceeded")
	ErrLimitMaxSize            = errors.New("batch size limit exceeded")
	ErrCompressFail            = errors.New("compress fail")
	ErrCreateRequestFail       = errors.New("create request fail")
	ErrSendFail                = errors.New("send fail")
	ErrReadResponseFail        = errors.New("read response fail")
	ErrHttpStatusNotOk         = errors.New("http status not ok")
	ErrUnmarshalResponseFail   = errors.New("unmarshal response fail")
	ErrEmptyResponse           = errors.New("empty response")
	ErrReturnLabelAlreadyExist = errors.New("return label already exist")
	ErrReturnLabelFail         = errors.New("return label fail")
)
