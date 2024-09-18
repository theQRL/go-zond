## EIP-1559 testing

This test contains testcases for EIP-1559, which uses an new transaction type and has a new block parameter. 

### Prestate

The alloc portion contains one contract (`0x000000000000000000000000000000000000aaaa`), containing the 
following code: `0x58585454`: `PC; PC; SLOAD; SLOAD`.

Essentially, this contract does `SLOAD(0)` and `SLOAD(1)`.

The alloc also contains some funds on `0x204cc644e26bdf879db422658edee62e302c3da8`. 

## Transactions

The transaction invokes the contract above. 

1. EIP-1559 ACL-transaction, which contains the `0x0` slot for `0xaaaa`

## Execution 

Running it yields: 
```
$ go run . t8n --state.fork=Shanghai --input.alloc=testdata/9/alloc.json --input.txs=testdata/9/txs.json --input.env=testdata/9/env.json --trace 2>/dev/null  && cat trace-*  | grep SLOAD
{"pc":2,"op":84,"gas":"0x48c28","gasCost":"0x834","memSize":0,"stack":["0x0","0x1"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":3,"op":84,"gas":"0x483f4","gasCost":"0x64","memSize":0,"stack":["0x0","0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
```

We can also get the post-alloc:
```
$ go run . t8n --state.fork=Shanghai --input.alloc=testdata/9/alloc.json --input.txs=testdata/9/txs.json --input.env=testdata/9/env.json --output.alloc=stdout 2>/dev/null
{
  "alloc": {
    "0x000000000000000000000000000000000000aaaa": {
      "code": "0x58585454",
      "balance": "0x3",
      "nonce": "0x1"
    },
    "0x204cc644e26bdf879db422658edee62e302c3da8": {
      "balance": "0xffe6fc39d8c920",
      "nonce": "0x1"
    },
    "0x2adc25665018aa1fe0e6bc666dac8fc2697ff9ba": {
      "balance": "0xd6e0"
    }
  }
}
```