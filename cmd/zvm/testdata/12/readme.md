## Test 1559 balance + gasCap

This test contains an EIP-1559 consensus issue which happened on Ropsten, where
`gzond` did not properly account for the value transfer while doing the check on `max_fee_per_gas * gas_limit`.

Before the issue was fixed, this invocation allowed the transaction to pass into a block:
```
$ go run . t8n --state.fork=Shanghai --input.alloc=testdata/12/alloc.json --input.txs=testdata/12/txs.json --input.env=testdata/12/env.json --output.alloc=stdout --output.result=stdout
```

With the fix applied, the result is: 
```
go run . t8n --state.fork=Shanghai --input.alloc=testdata/12/alloc.json --input.txs=testdata/12/txs.json --input.env=testdata/12/env.json --output.alloc=stdout --output.result=stdout
INFO [08-29|20:12:04.348] rejected tx                              index=0 hash=1d8f98..d32abf from=Z204cC644e26BDF879db422658eDEE62e302c3Da8 error="insufficient funds for gas * price + value: address Z204cC644e26BDF879db422658eDEE62e302c3Da8 have 84000000 want 84000032"
INFO [08-29|20:12:04.348] Trie dumping started                     root=67e50f..797459
INFO [08-29|20:12:04.348] Trie dumping complete                    accounts=1 elapsed="17.958µs"
{
  "alloc": {
    "Z204cc644e26bdf879db422658edee62e302c3da8": {
      "balance": "0x501bd00"
    }
  },
  "result": {
    "stateRoot": "0x67e50ffaf8e3008e329e71e0cd510967fba149576162d566f33df16f5d797459",
    "txRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
    "receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
    "logsHash": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
    "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "receipts": [],
    "rejected": [
      {
        "index": 0,
        "error": "insufficient funds for gas * price + value: address Z204cC644e26BDF879db422658eDEE62e302c3Da8 have 84000000 want 84000032"
      }
    ],
    "gasUsed": "0x0",
    "currentBaseFee": "0x20",
    "withdrawalsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
  }
}
```

The transaction is rejected. 