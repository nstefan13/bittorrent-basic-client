## Now

- [x] Bencode Decoder
  - [x] DecodeAny
    - [x] type alliases for the types specified in BEP3
          (so we stop using magical constants)
    - [x] error for EOF
    - [x] forward errors
  - [x] Lists
    - [x] Errors for areWeFinished
    - [x] Update DecodeAny
  - [x] Dictionaries
  - [ ] Refactor
    - [x] Return nil for the value when you have an error
    - [x] Parsing list: forward errors
    - [x] No more else if's

## Later

- [ ] DecodeInt: return math/big.Int instead of int

## Out of scope

- DHT / trackerless torrents
- Seeding
- Magnet links
