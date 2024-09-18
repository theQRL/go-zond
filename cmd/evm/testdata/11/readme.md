## Test missing basefee

In this test, the `currentBaseFee` is missing from the env portion. 
On a live blockchain, the basefee is present in the header, and verified as part of header validation. 

In `evm t8n`, we don't have blocks, so it needs to be added in the `env`instead. 

When it's missing, an error is expected. 

```
$ go run . t8n --state.fork=Shanghai --input.alloc=testdata/11/alloc.json --input.txs=testdata/11/txs.json --input.env=testdata/11/env.json --output.alloc=stdout --output.result=stdout
ERROR(3): EIP-1559 config but missing 'parentBaseFee' in env section
```