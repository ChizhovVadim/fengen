package main

import (
	"github.com/ChizhovVadim/CounterGo/common"
)

//Average cost: 0.071597
type MaterialEvalService struct {
}

func NewMaterialEvalService() *MaterialEvalService {
	return &MaterialEvalService{}
}

func (e *MaterialEvalService) Evaluate(p *common.Position) int {
	var eval = 100*(common.PopCount(p.Pawns&p.White)-common.PopCount(p.Pawns&p.Black)) +
		400*(common.PopCount(p.Knights&p.White)-common.PopCount(p.Knights&p.Black)) +
		400*(common.PopCount(p.Bishops&p.White)-common.PopCount(p.Bishops&p.Black)) +
		600*(common.PopCount(p.Rooks&p.White)-common.PopCount(p.Rooks&p.Black)) +
		1200*(common.PopCount(p.Queens&p.White)-common.PopCount(p.Queens&p.Black))
	if !p.WhiteMove {
		eval = -eval
	}
	return eval
}
