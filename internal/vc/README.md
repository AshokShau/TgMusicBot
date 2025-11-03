# Voice Chat Module

## Directory Structure

```
pkg/vc/
├── ntgcalls/           # Low-level ntgcalls bindings
│   ├── client.go       # Client implementation
│   ├── media_*.go      # Media handling (audio/video)
│   ├── types.go        # Type definitions
│   └── ...
└── ubot/               # Userbot utilities
    ├── connect_call.go # Call connection logic
    ├── context.go      # Call context management
    └── ...
```

## Credits
This implementation is based on the official Go examples from the [pytgcalls/ntgcalls](https://github.com/pytgcalls/ntgcalls) repository, specifically from the [examples/go](https://github.com/pytgcalls/ntgcalls/tree/master/examples/go) directory. Special thanks to [Laky-64](https://github.com/Laky-64) for the original Go implementation.

### License

This code is used under the same license as the original ntgcalls project (GNU Lesser General Public License v3.0). Please see the [original repository](https://github.com/pytgcalls/ntgcalls) for more details.
