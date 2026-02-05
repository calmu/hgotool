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
	ErrInvalidParam  = errors.New("invalid param")
	ErrLimitMaxBytes = errors.New("batch bytes limit exceeded")
	ErrLimitMaxSize  = errors.New("batch size limit exceeded")
)
