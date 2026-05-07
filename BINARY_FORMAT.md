# aytechnet/decimal binary format

This document specifies the open binary encoding used by `Decimal.MarshalBinary` /
`UnmarshalBinary` (and the equivalents on `Weight` and `Length`). It is designed to be
compact (typically 2–4 bytes for ordinary values), self-describing, and implementable by
third-party libraries that want to interoperate with this package.

The format has two layers:

* **v1 layer** — the original 1-byte header + optional uvarint mantissa, sufficient for any
  value the 8-byte `Decimal` type can represent (mantissa in `[-MaxInt, MaxInt]`, exponent
  in `[-16, 15]`). v1 bytes remain valid v2 input.
* **v2 extension layer** — extra opcodes that lift the mantissa/exponent restrictions and
  add explicit unit support for `Weight` / `Length`. Extension opcodes occupy bit patterns
  that, in v1, were used for redundant NaN encodings; v2 fixes NaN to a single canonical
  byte (`0x42` for `NaN`, `0xC2` for the rare `-NaN`) and reuses the rest.

## Header byte layout

Every encoded stream starts with one header byte:

```
  bit 7  (S) : sign of the mantissa (0 = positive, 1 = negative)
  bit 6  (L) : loss flag             (0 = exact,    1 = imprecise)
  bits 5..1  : signed 5-bit value E in two's complement (range [-16, 15])
  bit 0  (M) : 0 = no mantissa varint follows in v1
               1 = a uvarint mantissa follows in v1 (= "v1 normal")
```

The interpretation of the header depends on `M` and the total stream length:

| `M` | length     | meaning                                                                |
|-----|------------|------------------------------------------------------------------------|
| 1   | ≥ 2 bytes  | **Format A — v1 normal** (header + mantissa)                          |
| 0   | == 1 byte  | **Format B — v1 magic** (single-byte special value)                   |
| 0   | ≥ 2 bytes  | **Format C — v2 extension** (opcode + extension data)                 |

A reader wanting full backward compatibility MUST handle all three.

## Format A — v1 normal

The header byte holds `S | L | E | 1`, followed by an unsigned LEB128 varint
(`encoding/binary.Uvarint`) encoding the **absolute value of the mantissa**.

The decoded value is

```
  value = (S=0 ? +1 : -1) * mantissa * 10^E
```

The encoder is allowed (and the existing aytechnet implementation does) to fold the high
bit of the 57-bit mantissa into bit 0 of the header byte and force-set bit 0; the decoder
xors that bit out before OR-ing the uvarint mantissa back in. Implementations that prefer
to keep bit 0 strictly as the "mantissa-flag" are interoperable as long as bit 0 is set
to 1 and the full mantissa is in the uvarint.

Total size: 2 to 10 bytes (uvarint of `MaxInt = 2^57-1` fits in 9 bytes).

### Examples

| Decimal value | bytes              |
|---------------|--------------------|
| `1`           | `01 01`            |
| `-1`          | `81 01`            |
| `100`         | `01 64`            |
| `12345`       | `01 b9 60`         |
| `1.234`       | `3b d2 09`         |
| `-1.234`      | `bb d2 09`         |

## Format B — v1 magic

A single byte with `M = 0` encodes a magic value:

| byte   | binary       | meaning                                              |
|--------|--------------|------------------------------------------------------|
| `0x00` | `0000_0000`  | `Null` (uninitialized; treated as 0 in arithmetic)   |
| `0x80` | `1000_0000`  | `Zero` (explicit zero)                               |
| `0xC0` | `1100_0000`  | `NearZero` — magnitude smaller than representable    |
| `0x60` | `0110_0000`  | `+NearZero` (`~+0`)                                  |
| `0xE0` | `1110_0000`  | `-NearZero` (`~-0`)                                  |
| `0x5E` | `0101_1110`  | `+Infinity`                                          |
| `0xDE` | `1101_1110`  | `-Infinity`                                          |
| `0x42` | `0100_0010`  | `NaN`                                                |
| `0xC2` | `1100_0010`  | `NaN` with sign bit (rare; same semantics as `0x42`) |

In v1 every byte of the form `loss=1 ∧ E ∈ {1..14, -15..-1} ∧ M=0` encoded a NaN
("NaN-boxing"). v2 narrows NaN to `0x42`/`0xC2` only and reuses the other bytes for
extension opcodes (Format C). v2 readers MUST treat any single-byte `loss=1, M=0` input
as a magic value (NaN, `±~0`, `±Inf`, `~0`) for backward compatibility — only multi-byte
streams are interpreted as Format C.

## Format C — v2 extensions

When the header byte has `M = 0` AND the stream is at least 2 bytes long, the byte is an
**extension opcode**. The bits in the opcode are decoded as follows:

```
  bit 7  (S) : sign of the mantissa
  bit 6  (L) : loss flag
  bits 5..1  : type marker E, signed 5-bit value:
                 +2  →  Decimal  (positive exponent follows)
                 -2  →  Decimal  (negative exponent follows; |exp| is encoded)
                 +4  →  Weight   (positive exponent)
                 -4  →  Weight   (negative exponent)
                 +6  →  Length   (positive exponent)
                 -6  →  Length   (negative exponent)
  bit 0      : always 0 in this format
```

After the opcode byte the format is:

* For Decimal extension: `uvarint(|exp|)`, `uvarint(|m|)`
* For Weight / Length extension: `uvarint(unit)`, `uvarint(|exp|)`, `uvarint(|m|)`

The decoded value is

```
  value = (S=0 ? +1 : -1) * |m| * 10^((E<0 ? -|exp| : |exp|))
```

with `loss = L`. For Weight/Length, `unit` is the index into the package's unit table
(reproduced below).

Total size: 3 to 28 bytes (Decimal) or 4 to 37 bytes (Weight/Length, including unit).

### Decimal extension opcodes

| opcode | sign m | loss | sign exp | example value     |
|--------|--------|------|----------|-------------------|
| `0x04` | +      | exact | +       | `12345 * 10^29`   |
| `0x84` | -      | exact | +       | `-12345 * 10^29`  |
| `0x3C` | +      | exact | -       | `12345 * 10^-29`  |
| `0xBC` | -      | exact | -       | `-12345 * 10^-29` |
| `0x44` | +      | loss  | +       |                   |
| `0xC4` | -      | loss  | +       |                   |
| `0x7C` | +      | loss  | -       |                   |
| `0xFC` | -      | loss  | -       |                   |

### Weight extension opcodes

Same axes, with type marker `±4` (`exp_bits = 4` for positive exp, `28` for negative).

| opcode | sign m | loss | sign exp |
|--------|--------|------|----------|
| `0x08` | +      | exact | +       |
| `0x88` | -      | exact | +       |
| `0x38` | +      | exact | -       |
| `0xB8` | -      | exact | -       |
| `0x48` | +      | loss  | +       |
| `0xC8` | -      | loss  | +       |
| `0x78` | +      | loss  | -       |
| `0xF8` | -      | loss  | -       |

### Length extension opcodes

Type marker `±6`.

| opcode | sign m | loss | sign exp |
|--------|--------|------|----------|
| `0x0C` | +      | exact | +       |
| `0x8C` | -      | exact | +       |
| `0x34` | +      | exact | -       |
| `0xB4` | -      | exact | -       |
| `0x4C` | +      | loss  | +       |
| `0xCC` | -      | loss  | +       |
| `0x74` | +      | loss  | -       |
| `0xF4` | -      | loss  | -       |

### Unit tables

#### Weight (`weightUnits`)

| code | unit  | coefficient (kg)                        |
|------|-------|-----------------------------------------|
| 0    | `kg`  | 1 (default — encoded as Decimal)        |
| 1    | `t`   | 10^3                                    |
| 2    | `kt`  | 10^6                                    |
| 3    | `Mt`  | 10^9                                    |
| 4    | `Gt`  | 10^12                                   |
| 5    | `g`   | 10^-3                                   |
| 6    | `mg`  | 10^-6                                   |
| 7    | `µg`  | 10^-9                                   |
| 8    | `ng`  | 10^-12                                  |
| 9    | `pg`  | 10^-15                                  |
| 10–11| —     | reserved                                |
| 12   | `lb`  | 0.45359237 (NIST 1959 exact)            |
| 13   | `oz`  | 0.028349523125                          |
| 14   | `lb t`| 0.3732417216                            |
| 15   | `oz t`| 0.0311034768                            |

#### Length (`lengthUnits`)

| code | unit  | coefficient (m)                         |
|------|-------|-----------------------------------------|
| 0    | `m`   | 1 (default — encoded as Decimal)        |
| 1    | `km`  | 10^3                                    |
| 2    | `dm`  | 10^-1                                   |
| 3    | `cm`  | 10^-2                                   |
| 4    | `mm`  | 10^-3                                   |
| 5    | `µm`  | 10^-6                                   |
| 6    | `nm`  | 10^-9                                   |
| 7    | `pm`  | 10^-12                                  |
| 8–10 | —     | reserved                                |
| 11   | `au`  | 1.495978707 × 10^11 (UAI 2012 exact)    |
| 12   | `in`  | 0.0254 (NIST 1959 exact)                |
| 13   | `ft`  | 0.3048                                  |
| 14   | `yd`  | 0.9144                                  |
| 15   | `mi`  | 1609.344                                |

## Default-unit shortcut

A `Weight` whose unit is `kg` (code 0) and a `Length` whose unit is `m` (code 0) are
encoded **as a plain `Decimal`** (Format A or B) — no opcode, no unit byte. This:

* keeps the v1 Decimal stream representation untouched (no breakage when going from v1 to v2);
* makes a `Decimal` `123` and a `Weight` `123kg` and a `Length` `123m` produce the **same
  byte sequence**;
* saves 1 to 2 bytes per encoded value when the default unit is in use, which is the most
  common case.

## Cross-type reading

| reader is        | accepts v1 Decimal | accepts Decimal ext | accepts Weight ext | accepts Length ext |
|------------------|:------------------:|:-------------------:|:------------------:|:------------------:|
| `Decimal`        | ✓                 | ✓                  | ✓ (unit dropped, scalar kept) | ✓ (unit dropped) |
| `Weight`         | ✓ (assumes `kg`)  | ✓ (assumes `kg`)   | ✓                 | ✗ (`ErrFormat`)   |
| `Length`         | ✓ (assumes `m`)   | ✓ (assumes `m`)    | ✗ (`ErrFormat`)   | ✓                 |

Reading a `Weight 5g` as a `Decimal` returns `5` (the scalar `m × 10^exp` of the
encoded value, **not** `0.005` — no unit conversion is performed). This is symmetric
with writing: `Decimal 5` → `Weight 5kg` → same bytes.

Magic values (NaN, ±Inf, ±~0, NearZero) are encoded with Format B and **do not carry a
unit**. `Weight NaN with unit g` round-trips to `Weight NaN with unit kg`, which is
acceptable because the unit of a non-finite magnitude is not well-defined.

## Test vectors

Output of `MarshalBinary` for the canonical `Decimal` package (verified by the test
suite):

```
Decimal 0          = 00
Decimal Zero       = 80
Decimal 1          = 01 01
Decimal -1         = 81 01
Decimal 100        = 01 64
Decimal 12345      = 01 b9 60
Decimal 1.234      = 3b d2 09
Decimal -1.234     = bb d2 09
Decimal 99999999999.999 = 3b ff ff e8 83 b1 de 16
Decimal NaN        = 42
Decimal +Inf       = 5e
Decimal -Inf       = de
Decimal NearZero   = c0
Decimal +~0        = 60
Decimal -~0        = e0

Weight 5kg         = 01 05            (= Decimal 5)
Weight 5g          = 08 05 00 05      (opcode Weight exact +exp +m, unit=g, exp=0, m=5)
Weight -3g         = 88 05 00 03
Weight 11lb        = 08 0c 00 0b      (unit=lb, exp=0, m=11)

Length 1m          = 01 01            (= Decimal 1)
Length 1ft         = 0c 0d 00 01      (opcode Length exact +exp +m, unit=ft, exp=0, m=1)
Length 1au         = 0c 0b 00 01      (unit=au, exp=0, m=1)
```

## Versioning and forward compatibility

The format has no explicit version byte. Forward extensions are accommodated by:

* The reserved opcode space — currently 12 of ~94 free non-v1 byte values are used.
  Future types can claim more `±expBits` markers (e.g. `±8`, `±10`).
* The reserved unit codes (10–11 in Weight, 8–10 in Length) for new units within the
  existing types.

A v2 reader presented with an unknown opcode SHOULD return `ErrFormat` rather than
silently mis-decoding.
