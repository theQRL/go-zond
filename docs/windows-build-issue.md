# Windows Build Issue with Go 1.23.6

## Issue Description
When building go-zond on Windows using Go version 1.23.6, the build fails with the following error:
```
# github.com/fjl/memsize
C:\Users\USERNAME\go\pkg\mod\github.com\fjl\memsize@v0.0.1-0.20220121193149-6d896a72e289\memsize.go:50:13: undefined: runtime.stopTheWorld
```

## Affected Versions
- Go 1.23.6 on Windows: ❌ Fails
- Go 1.22.12 and earlier: ✅ Works

## Root Cause
The issue is caused by changes in Go 1.23.6 that affect the `runtime.stopTheWorld` function, which is used by the `github.com/fjl/memsize` package. This internal runtime function has been removed or modified in Go 1.23.6.

## Solution
1. **Recommended**: Use Go 1.22.12 or earlier for Windows builds
2. **Alternative**: Use multiple Go versions with `gvm` or similar tools

## Workaround Details
1. Download Go 1.22.12 from the official Go website
2. Install Go 1.22.12
3. Set your Go version to 1.22.12:
   ```bash
   go env -w GOTOOLCHAIN=go1.22.12
   ```
4. Rebuild the project

## Future Updates
This issue will be addressed in future releases of go-zond. We are working on updating the memory size calculation implementation to be compatible with newer Go versions.

## Related Issues
- GitHub Issue: [#66](https://github.com/theQRL/go-zond/issues/66)
- Related Package: [github.com/fjl/memsize](https://github.com/fjl/memsize) 