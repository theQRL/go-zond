// Copyright 2021 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package t8ntool

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/go-zond/common/math"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/log"
	"github.com/theQRL/go-zond/rlp"
	"github.com/urfave/cli/v2"
)

//go:generate go run github.com/fjl/gencodec -type header -field-override headerMarshaling -out gen_header.go
type header struct {
	ParentHash      common.Hash     `json:"parentHash"`
	Coinbase        *common.Address `json:"miner"`
	Root            common.Hash     `json:"stateRoot"        gencodec:"required"`
	TxHash          *common.Hash    `json:"transactionsRoot"`
	ReceiptHash     *common.Hash    `json:"receiptsRoot"`
	Bloom           types.Bloom     `json:"logsBloom"`
	Number          *big.Int        `json:"number"           gencodec:"required"`
	GasLimit        uint64          `json:"gasLimit"         gencodec:"required"`
	GasUsed         uint64          `json:"gasUsed"`
	Time            uint64          `json:"timestamp"        gencodec:"required"`
	Extra           []byte          `json:"extraData"`
	Random          common.Hash     `json:"prevRandao"`
	BaseFee         *big.Int        `json:"baseFeePerGas" rlp:"optional"`
	WithdrawalsHash *common.Hash    `json:"withdrawalsRoot" rlp:"optional"`
}

type headerMarshaling struct {
	Number   *math.HexOrDecimal256
	GasLimit math.HexOrDecimal64
	GasUsed  math.HexOrDecimal64
	Time     math.HexOrDecimal64
	Extra    hexutil.Bytes
	BaseFee  *math.HexOrDecimal256
}

type bbInput struct {
	Header      *header             `json:"header,omitempty"`
	TxRlp       string              `json:"txs,omitempty"`
	Withdrawals []*types.Withdrawal `json:"withdrawals,omitempty"`

	Ethash bool                 `json:"-"`
	Txs    []*types.Transaction `json:"-"`
}

// ToBlock converts i into a *types.Block
func (i *bbInput) ToBlock() *types.Block {
	header := &types.Header{
		ParentHash:      i.Header.ParentHash,
		Coinbase:        common.Address{},
		Root:            i.Header.Root,
		TxHash:          types.EmptyTxsHash,
		ReceiptHash:     types.EmptyReceiptsHash,
		Bloom:           i.Header.Bloom,
		Number:          i.Header.Number,
		GasLimit:        i.Header.GasLimit,
		GasUsed:         i.Header.GasUsed,
		Time:            i.Header.Time,
		Extra:           i.Header.Extra,
		Random:          i.Header.Random,
		BaseFee:         i.Header.BaseFee,
		WithdrawalsHash: i.Header.WithdrawalsHash,
	}

	// Fill optional values.
	if i.Header.Coinbase != nil {
		header.Coinbase = *i.Header.Coinbase
	}
	if i.Header.TxHash != nil {
		header.TxHash = *i.Header.TxHash
	}
	if i.Header.ReceiptHash != nil {
		header.ReceiptHash = *i.Header.ReceiptHash
	}
	return types.NewBlockWithHeader(header).WithBody(i.Txs).WithWithdrawals(i.Withdrawals)
}

// SealBlock seals the given block using the configured engine.
func (i *bbInput) SealBlock(block *types.Block) (*types.Block, error) {
	switch {
	default:
		return block, nil
	}
}

// BuildBlock constructs a block from the given inputs.
func BuildBlock(ctx *cli.Context) error {
	// Configure the go-ethereum logger
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(ctx.Int(VerbosityFlag.Name)))
	log.Root().SetHandler(glogger)

	baseDir, err := createBasedir(ctx)
	if err != nil {
		return NewError(ErrorIO, fmt.Errorf("failed creating output basedir: %v", err))
	}
	inputData, err := readInput(ctx)
	if err != nil {
		return err
	}
	block := inputData.ToBlock()
	block, err = inputData.SealBlock(block)
	if err != nil {
		return err
	}
	return dispatchBlock(ctx, baseDir, block)
}

func readInput(ctx *cli.Context) (*bbInput, error) {
	var (
		headerStr      = ctx.String(InputHeaderFlag.Name)
		withdrawalsStr = ctx.String(InputWithdrawalsFlag.Name)
		txsStr         = ctx.String(InputTxsRlpFlag.Name)
		inputData      = &bbInput{}
	)
	if headerStr == stdinSelector || txsStr == stdinSelector {
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(inputData); err != nil {
			return nil, NewError(ErrorJson, fmt.Errorf("failed unmarshaling stdin: %v", err))
		}
	}
	if headerStr != stdinSelector {
		var env header
		if err := readFile(headerStr, "header", &env); err != nil {
			return nil, err
		}
		inputData.Header = &env
	}
	if withdrawalsStr != stdinSelector && withdrawalsStr != "" {
		var withdrawals []*types.Withdrawal
		if err := readFile(withdrawalsStr, "withdrawals", &withdrawals); err != nil {
			return nil, err
		}
		inputData.Withdrawals = withdrawals
	}
	if txsStr != stdinSelector {
		var txs string
		if err := readFile(txsStr, "txs", &txs); err != nil {
			return nil, err
		}
		inputData.TxRlp = txs
	}
	// Deserialize rlp txs
	var (
		txs = []*types.Transaction{}
	)
	if inputData.TxRlp != "" {
		if err := rlp.DecodeBytes(common.FromHex(inputData.TxRlp), &txs); err != nil {
			return nil, NewError(ErrorRlp, fmt.Errorf("unable to decode transaction from rlp data: %v", err))
		}
		inputData.Txs = txs
	}

	return inputData, nil
}

// dispatchBlock writes the output data to either stderr or stdout, or to the specified
// files
func dispatchBlock(ctx *cli.Context, baseDir string, block *types.Block) error {
	raw, _ := rlp.EncodeToBytes(block)
	type blockInfo struct {
		Rlp  hexutil.Bytes `json:"rlp"`
		Hash common.Hash   `json:"hash"`
	}
	enc := blockInfo{
		Rlp:  raw,
		Hash: block.Hash(),
	}
	b, err := json.MarshalIndent(enc, "", "  ")
	if err != nil {
		return NewError(ErrorJson, fmt.Errorf("failed marshalling output: %v", err))
	}
	switch dest := ctx.String(OutputBlockFlag.Name); dest {
	case "stdout":
		os.Stdout.Write(b)
		os.Stdout.WriteString("\n")
	case "stderr":
		os.Stderr.Write(b)
		os.Stderr.WriteString("\n")
	default:
		if err := saveFile(baseDir, dest, enc); err != nil {
			return err
		}
	}
	return nil
}
