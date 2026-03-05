# What Vega Would Need to Serialize .vgc Files

Here's what a `.vgc` (Vega compiled) format could look like, following the same conventions as the languages above:

```
=== VEGA COMPILED OBJECT (.vgc) ===

HEADER (fixed 16 bytes):
  4 bytes  magic: 0x56 0x45 0x47 0x41  ("VEGA")
  2 bytes  format version (increment on breaking bytecode changes)
  2 bytes  flags (bit 0: debug info included, bit 1: source hash present)
  4 bytes  source file CRC32 or 0 if unknown
  4 bytes  total file size (for fast corruption check)

SECTION TABLE (after header):
  2 bytes  section count
  [per section:]
    1 byte   section id
    4 bytes  offset from start of file
    4 bytes  byte length

SECTION 0 — CONSTANTS:
  4 bytes  count
  [per constant:]
    1 byte   TypeTag
    2 bytes  data length
    N bytes  raw data (LittleEndian encoded, same as value.Encode)

SECTION 1 — NAMES (interned string table):
  4 bytes  count
  [per name:]
    2 bytes  string length
    N bytes  UTF-8 bytes

SECTION 2 — INSTRUCTIONS (main body):
  4 bytes  count
  [per instruction, fixed 10 bytes each:]
    1 byte   OperationCode
    3 bytes  Argument  (fits int24 — max 16M instructions is more than enough)
    3 bytes  Offset
    1 byte   Extra
    2 bytes  Size

  (SourceLine is omitted unless flags bit 0 is set)

SECTION 3 — DEBUG INFO (optional, flags bit 0):
  4 bytes  count (must equal instruction count)
  [per instruction:]
    2 bytes  source line number

SECTION 4 — FUNCTIONS:
  4 bytes  count
  [per function:]
    2 bytes  name index into NAMES section
    4 bytes  frame size (FunctionDef.FrameSize)
    1 byte   param count
    [per param:]
      1 byte   TypeTag
    [embedded ByteCode, same structure as sections 0-3 but length-prefixed:]
      4 bytes  constants count + constants payload
      4 bytes  names count + names payload
      4 bytes  instructions count + instructions payload
```

Key design decisions, mirroring what the other formats do:
* Fixed magic + version: invalidate stale cached files when bytecode format changes. Without this, you'll silently execute outdated bytecode and get confusing runtime errors.
* CRC/hash of the source file: when loading, re-check whether the source has changed. If it has, recompile; otherwise load the .vgc. This is exactly what CPython's .pyc timestamp/hash field does.
* Section table: allows a loader to skip sections it doesn't understand (forward compatibility), and to seek directly to the code section without parsing everything.
* Fixed-width instructions: your Instruction struct has multiple int fields that are serialized as int64 by default. Compressing them to int24 each saves 40% of instruction stream size with no semantic loss (16 million slot IDs or constant indices is unreachable in practice).
* Stripped debug info by default: SourceLine is only needed for error messages. Strip it from production builds; include it in debug builds. Java does this via the optional LineNumberTable attribute; Lua includes lineinfo unconditionally in luac output but strips it with luac -s.
* Functions embedded recursively: matching Lua's Proto*[] — functions contain their own bytecode inline rather than being a flat list at the top level, which is how your ByteCode.Functions map is already structured.