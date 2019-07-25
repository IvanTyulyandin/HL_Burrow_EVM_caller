package main

import "C"
import (
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"
	"github.com/hyperledger/burrow/execution/evm"
	"github.com/hyperledger/burrow/logging"
	"github.com/tmthrgd/go-hex"
	"golang.org/x/crypto/ripemd160"
	"os"
	"strconv"
)

func newParams() evm.Params {
	return evm.Params{
		BlockHeight: 0,
		BlockTime:   0,
		GasLimit:    0,
	}
}

// toEVMaddress converts any string to EVM address
func toEVMaddress(name string) crypto.Address {
	hasher := ripemd160.New()
	hasher.Write([]byte(name))
	return crypto.MustAddressFromBytes(hasher.Sum(nil))
}

func blockHashGetter(height uint64) []byte {
	return binary.LeftPadWord256([]byte(fmt.Sprintf("block_hash_%d", height))).Bytes()
}

func newLogger() *logging.Logger {
	return &logging.Logger{
		Info:   log.NewLogfmtLogger(os.Stdout),
		Trace:  log.NewLogfmtLogger(os.Stdout),
		Output: new(log.SwapLogger),
	}
}

func VmCall(code, input, caller, callee *C.char) (*C.char, bool) {

	// Convert string into EVM address
	account1 := toEVMaddress(C.GoString(caller))

	// if callee is empty -> new contract creation
	goCallee := C.GoString(callee)
	account2 := crypto.Address{}

	if goCallee != "" {
		// take this assignment from
		// https://github.com/hyperledger/sawtooth-seth/blob/master/processor/src/seth_tp/handler/handler.go#L159
		account2 = account1
	}

	var gas uint64 = 1000000
	goByteCode := hex.MustDecodeString(C.GoString(code))
	goInput := hex.MustDecodeString(C.GoString(input))
	output, err := ourVm.Call(evmState, evm.NewNoopEventSink(), account1, account2,
		goByteCode, goInput, 0, &gas)

	if err := evmState.Sync(); err != nil {
		panic("Sync error")
	}
	// Transform output data to a string value.
	// It is a problem to convert []byte, which contains 0 byte inside, to C string.
	// Conversion to C.CString will cut all data after the 0 byte.
	res := ""
	for _, dataAsInt := range output {

		// change base to hex
		tmp := strconv.FormatInt(int64(dataAsInt), 16)

		// save bytecode structure, where hex value f should be 0f, and so on
		if len(tmp) < 2 {
			// len 1 at least after conversion from variable output
			tmp = "0" + tmp
		}
		res += tmp
	}

	if err == nil {
		return C.CString(res), true
	} else {
		fmt.Println(err)
		fmt.Println("NOT NIL")
		return C.CString(res), false
	}
}

var appState = NewIrohaAppState()
var logger =logging.NewNoopLogger()
var ourVm = evm.NewVM(newParams(), crypto.ZeroAddress, nil, logger)
var evmState = evm.NewState(appState, blockHashGetter)


/*
Bytecode was taken from Remix IDE, compiler version 0.5.10+commit.5a6ea5b1.Emscripten.clang
pragma solidity ^0.5.4;
contract SimpleStorage {

	uint256 data;

	function get() public view returns (uint256) {
		return data;
	}
	
	function set(uint256 newData) public {
		data = newData;
		return;
	}
}
*/
var code = C.CString("608060405234801561001057600080fd5b5060c68061001f600039" +
	"6000f3fe6080604052348015600f57600080fd5b506004361060325760003560e01c806" +
	"360fe47b11460375780636d4ce63c146062575b600080fd5b6060600480360360208110" +
	"15604b57600080fd5b8101908080359060200190929190505050607e565b005b6068608" +
	"8565b6040518082815260200191505060405180910390f35b8060008190555050565b60" +
	"00805490509056fea265627a7a72305820a191db5c7b4d4786fc90adff0e100187127c5" +
	"4e0e902d124a41606297538376964736f6c634300050a0032")

var set = C.CString("60fe47b1" +
	"0000000000000000000000000000000000000000000000000000000000000001")

var get = C.CString("6d4ce63c")

var caller = C.CString("Caller")
var callee = C.CString("callee")

func main() {
	evmState.CreateAccount(toEVMaddress(C.GoString(caller)))
	evmState.Sync()
	fmt.Println(appState.accounts)
	fmt.Println(appState.storage)
	//VmCall(code, code, caller, callee)
	output, _ := VmCall(code, code, caller, callee)
	output, _ = VmCall(output, set, caller, callee)
	for _, acc := range appState.accounts {
		fmt.Println(acc.EVMCode)
	}
	fmt.Println(appState.accounts)
	fmt.Println(appState.storage)
}
