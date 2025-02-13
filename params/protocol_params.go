// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

const (
	GasLimitBoundDivisor uint64 = 1024               // The bound divisor of the gas limit, used in update calculations.
	MinGasLimit          uint64 = 5000               // Minimum the gas limit may ever be.
	MaxGasLimit          uint64 = 20000000           // Maximum the gas limit
	GenesisGasLimit      uint64 = 4712388            // Gas limit of the Genesis block.

	MaximumExtraDataSize  uint64 = 32    // Maximum size extra data may be after Genesis.
	SloadGas              uint64 = 50    // Multiplied by the number of 32-byte words that are copied (round up) for any *COPY operation and added.
	CallValueTransferGas  uint64 = 9000  // Paid for CALL when the value transfer is non-zero.
	CallNewAccountGas     uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.
	TxGas                 uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation uint64 = 53000 // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	TxDataZeroGas         uint64 = 4     // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	QuadCoeffDiv          uint64 = 512   // Divisor for the quadratic particle of the memory cost equation.
	LogDataGas            uint64 = 8     // Per byte in a LOG* operation's data.
	CallStipend           uint64 = 2300  // Free gas given at beginning of call.

	Keccak256Gas     uint64 = 30 // Once per KECCAK256 operation.
	Keccak256WordGas uint64 = 6  // Once per word of the KECCAK256 operation's data.
	InitCodeWordGas  uint64 = 2  // Once per word of the init code when creating a contract.

	SstoreSentryGasEIP2200            uint64 = 2300  // Minimum gas required to be present for an SSTORE call, not consumed
	SstoreSetGasEIP2200               uint64 = 20000 // Once per SSTORE operation from clean zero to non-zero
	SstoreResetGasEIP2200             uint64 = 5000  // Once per SSTORE operation from clean non-zero to something else
	SstoreClearsScheduleRefundEIP2200 uint64 = 15000 // Once per SSTORE operation for clearing an originally existing storage slot

	ColdAccountAccessCostEIP2929 = uint64(2600) // COLD_ACCOUNT_ACCESS_COST
	ColdSloadCostEIP2929         = uint64(2100) // COLD_SLOAD_COST
	WarmStorageReadCostEIP2929   = uint64(100)  // WARM_STORAGE_READ_COST

	// In EIP-2200: SstoreResetGas was 5000.
	// In EIP-2929: SstoreResetGas was changed to '5000 - COLD_SLOAD_COST'.
	// In EIP-3529: SSTORE_CLEARS_SCHEDULE is defined as SSTORE_RESET_GAS + ACCESS_LIST_STORAGE_KEY_COST
	// Which becomes: 5000 - 2100 + 1900 = 4800
	SstoreClearsScheduleRefundEIP3529 uint64 = SstoreResetGasEIP2200 - ColdSloadCostEIP2929 + TxAccessListStorageKeyGas

	JumpdestGas uint64 = 1 // Once per JUMPDEST operation.

	CreateDataGas   uint64 = 200   //
	CallCreateDepth uint64 = 1024  // Maximum depth of call/create stack.
	ExpGas          uint64 = 10    // Once per EXP instruction
	LogGas          uint64 = 375   // Per LOG* operation.
	CopyGas         uint64 = 3     //
	StackLimit      uint64 = 1024  // Maximum size of VM stack allowed.
	LogTopicGas     uint64 = 375   // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	CreateGas       uint64 = 32000 // Once per CREATE operation & contract-creation transaction.
	Create2Gas      uint64 = 32000 // Once per CREATE2 operation
	MemoryGas       uint64 = 3     // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.

	TxDataNonZeroGasEIP2028   uint64 = 16   // Per byte of non zero data attached to a transaction after EIP 2028
	TxAccessListAddressGas    uint64 = 2400 // Per address specified in EIP 2930 access list
	TxAccessListStorageKeyGas uint64 = 1900 // Per storage key specified in EIP 2930 access list

	// These have been changed during the course of the chain
	SloadGasEIP2200 uint64 = 800 // Cost of SLOAD

	// EXP has a dynamic portion depending on the size of the exponent
	ExpByteEIP158 uint64 = 50 // was raised to 50 during Eip158

	DefaultBaseFeeChangeDenominator = 8          // Bounds the amount the base fee can change between blocks.
	DefaultElasticityMultiplier     = 2          // Bounds the maximum gas limit a block may have.
	InitialBaseFee                  = 1000000000 // Initial base fee.

	MaxCodeSize     = 24576           // Maximum bytecode to permit for a contract
	MaxInitCodeSize = 2 * MaxCodeSize // Maximum initcode to permit in a creation transaction and create instructions

	// Precompiled contract gas prices
	DepositrootGas     uint64 = 3000 // Deposit root operation gas price
	Sha256BaseGas      uint64 = 60   // Base price for a SHA256 operation
	Sha256PerWordGas   uint64 = 12   // Per-word price for a SHA256 operation
	IdentityBaseGas    uint64 = 15   // Base price for a data copy operation
	IdentityPerWordGas uint64 = 3    // Per-work price for a data copy operation

	Bn256AddGasIstanbul             uint64 = 150   // Gas needed for an elliptic curve addition
	Bn256ScalarMulGasIstanbul       uint64 = 6000  // Gas needed for an elliptic curve scalar multiplication
	Bn256PairingBaseGasIstanbul     uint64 = 45000 // Base price for an elliptic curve pairing check
	Bn256PairingPerPointGasIstanbul uint64 = 34000 // Per-point price for an elliptic curve pairing check

	// The Refund Quotient is the cap on how much of the used gas can be refunded. Before EIP-3529,
	// up to half the consumed gas could be refunded. Redefined as 1/5th in EIP-3529
	RefundQuotient        uint64 = 2
	RefundQuotientEIP3529 uint64 = 5
)
