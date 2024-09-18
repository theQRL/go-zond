## EIP-2930 testing

This test contains testcases for EIP-2930, which uses transactions with access lists. 

### Prestate

The alloc portion contains one contract (`0x000000000000000000000000000000000000aaaa`), containing the 
following code: `0x5854505854`: `PC ;SLOAD; POP; PC; SLOAD`.

Essentially, this contract does `SLOAD(0)` and `SLOAD(3)`.

The alloc also contains some funds on `0x204cc644e26bdf879db422658edee62e302c3da8`. 

## Transactions

There are three transactions, each invokes the contract above. 

1. ACL-transaction, which contains some non-used slots
2. Regular transaction
3. ACL-transaction, which contains the slots `1` and `3` in `0x000000000000000000000000000000000000aaaa`

## Execution 

Running it yields: 
```
$ go run . t8n --state.fork=Shanghai --input.alloc=testdata/8/alloc.json --input.txs=testdata/8/txs.json --input.env=testdata/8/env.json --trace 2>/dev/null && cat trace-* | grep SLOAD
{"pc":1,"op":84,"gas":"0x484be","gasCost":"0x834","memSize":0,"stack":["0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":4,"op":84,"gas":"0x47c86","gasCost":"0x834","memSize":0,"stack":["0x3"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":1,"op":84,"gas":"0x49cf6","gasCost":"0x834","memSize":0,"stack":["0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":4,"op":84,"gas":"0x494be","gasCost":"0x834","memSize":0,"stack":["0x3"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":1,"op":84,"gas":"0x484be","gasCost":"0x64","memSize":0,"stack":["0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":4,"op":84,"gas":"0x48456","gasCost":"0x64","memSize":0,"stack":["0x3"],"depth":1,"refund":0,"opName":"SLOAD"}
```

Simlarly, we can provide the input transactions via `stdin` instead of as file: 

```
$ cat testdata/8/txs.json | jq "{txs: .}" \
  | go run . t8n --state.fork=Shanghai \
     --input.alloc=testdata/8/alloc.json \
     --input.txs=stdin \
     --input.env=testdata/8/env.json \
     --trace  \
     2>/dev/null \
  && cat trace-* | grep SLOAD
{"pc":1,"op":84,"gas":"0x484be","gasCost":"0x834","memSize":0,"stack":["0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":4,"op":84,"gas":"0x47c86","gasCost":"0x834","memSize":0,"stack":["0x3"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":1,"op":84,"gas":"0x49cf6","gasCost":"0x834","memSize":0,"stack":["0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":4,"op":84,"gas":"0x494be","gasCost":"0x834","memSize":0,"stack":["0x3"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":1,"op":84,"gas":"0x484be","gasCost":"0x64","memSize":0,"stack":["0x0"],"depth":1,"refund":0,"opName":"SLOAD"}
{"pc":4,"op":84,"gas":"0x48456","gasCost":"0x64","memSize":0,"stack":["0x3"],"depth":1,"refund":0,"opName":"SLOAD"}
```
