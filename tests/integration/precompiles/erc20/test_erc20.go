package erc20

import (
	"math/big"

	"github.com/yihuang/go-abi"

	"github.com/cosmos/evm/precompiles/erc20"
)

func (s *PrecompileTestSuite) TestIsTransaction() {
	s.SetupTest()

	// Queries
	balanceOfCall := &erc20.BalanceOfCall{}
	s.Require().False(s.precompile.IsTransaction(balanceOfCall.GetMethodID()))
	decimalsCall := &erc20.DecimalsCall{}
	s.Require().False(s.precompile.IsTransaction(decimalsCall.GetMethodID()))
	nameCall := &erc20.NameCall{}
	s.Require().False(s.precompile.IsTransaction(nameCall.GetMethodID()))
	symbolCall := &erc20.SymbolCall{}
	s.Require().False(s.precompile.IsTransaction(symbolCall.GetMethodID()))
	totalSupplyCall := &erc20.TotalSupplyCall{}
	s.Require().False(s.precompile.IsTransaction(totalSupplyCall.GetMethodID()))

	// Transactions
	approveCall := &erc20.ApproveCall{}
	s.Require().True(s.precompile.IsTransaction(approveCall.GetMethodID()))
	transferCall := &erc20.TransferCall{}
	s.Require().True(s.precompile.IsTransaction(transferCall.GetMethodID()))
	transferFromCall := &erc20.TransferFromCall{}
	s.Require().True(s.precompile.IsTransaction(transferFromCall.GetMethodID()))
}

func (s *PrecompileTestSuite) TestRequiredGas() {
	s.SetupTest()

	testcases := []struct {
		name     string
		malleate func() []byte
		expGas   uint64
	}{
		{
			name: erc20.BalanceOfMethod,
			malleate: func() []byte {
				call := erc20.BalanceOfCall{
					Account: s.keyring.GetAddr(0),
				}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasBalanceOf,
		},
		{
			name: erc20.DecimalsMethod,
			malleate: func() []byte {
				call := erc20.DecimalsCall{}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasDecimals,
		},
		{
			name: erc20.NameMethod,
			malleate: func() []byte {
				call := erc20.NameCall{}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasName,
		},
		{
			name: erc20.SymbolMethod,
			malleate: func() []byte {
				call := erc20.SymbolCall{}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasSymbol,
		},
		{
			name: erc20.TotalSupplyMethod,
			malleate: func() []byte {
				call := erc20.TotalSupplyCall{}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasTotalSupply,
		},
		{
			name: erc20.ApproveMethod,
			malleate: func() []byte {
				call := erc20.ApproveCall{
					Spender: s.keyring.GetAddr(0),
					Amount:  big.NewInt(1),
				}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasApprove,
		},
		{
			name: erc20.TransferMethod,
			malleate: func() []byte {
				call := erc20.TransferCall{
					To:     s.keyring.GetAddr(0),
					Amount: big.NewInt(1),
				}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasTransfer,
		},
		{
			name: erc20.TransferFromMethod,
			malleate: func() []byte {
				call := erc20.TransferFromCall{
					From:   s.keyring.GetAddr(0),
					To:     s.keyring.GetAddr(0),
					Amount: big.NewInt(1),
				}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasTransferFrom,
		},
		{
			name: erc20.AllowanceMethod,
			malleate: func() []byte {
				call := erc20.AllowanceCall{
					Owner:  s.keyring.GetAddr(0),
					Spender: s.keyring.GetAddr(0),
				}
				bz, err := call.EncodeWithSelector()
				s.Require().NoError(err, "expected no error packing ABI")
				return bz
			},
			expGas: erc20.GasAllowance,
		},
		{
			name: "invalid method",
			malleate: func() []byte {
				return []byte("invalid method")
			},
			expGas: 0,
		},
		{
			name: "input bytes too short",
			malleate: func() []byte {
				return []byte{0x00, 0x00, 0x00}
			},
			expGas: 0,
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			input := tc.malleate()

			s.Require().Equal(tc.expGas, s.precompile.RequiredGas(input))
		})
	}
}
