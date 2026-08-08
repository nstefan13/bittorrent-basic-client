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
  - [x] Refactor
    - [x] Return nil for the value when you have an error
    - [x] Parsing list: forward errors
    - [x] No more else if's
- [x] Metainfo Parser
- [ ] Request Peers
  - [ ] write about our mocked tracker
  - [x] write about our support for only HTTP\* trackers
  - [x] refactor: move all errors to a errors.go for each package
  - [x] try to run gremlins and mention the coverage
  - [x] test the request creation
  - [x] test the response:
    - [x] handle the compact format
    - [x] handle the loose format
    - [x] handle the failure reason
- [ ] property based testing / fuzzing for encoder/decoder

## Later

- [ ] DecodeInt: return math/big.Int instead of int

## Out of scope

- Seeding
- DHT / trackerless torrents
- Magnet links
