package entity

import "context"

type Audio struct {
	Ext  string
	Body []byte
}

type AudioConvertRequest struct {
	Ctx   context.Context
	Audio Audio
	Ch    chan<- Audio
}
