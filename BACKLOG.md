## Now

- [ ] Bencode Decoder
  - [x] DecodeAny
    - [x] type alliases for the types specified in BEP3
          (so we stop using magical constants)
    - [x] error for EOF
    - [x] forward errors
  - [ ] Lists
  - [ ] Dictionaries

## Later

- [ ] DecodeInt: return math/big.Int instead of int

## Out of scope

- DHT / trackerless torrents
- Seeding
- Magnet links
