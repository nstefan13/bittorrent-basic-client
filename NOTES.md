# Architecture

## Bencoding

Let's talk about the decoder (since the same principles apply to the encoder).

Some decoders, like the native encoding/json, parse a byte stream and map it on the fly to a concrete type. This has the nice advantage of being able to skip parts of the decoding process (if an API sends you a JSON with 100 fields, but you only need 20 of them, why waste CPU cycles decoding all the 100 fields when you could simply skip the 80 useless one entirely and only parse those that you truly need). This, however, requires the decoder to have 2 different responsabilities: (1) parse the stream of bytes and (2) handle the mapping from JSON datatypes to a given struct's concrete fields. That last part is very difficult to make right as it requires the usage of reflection which is notorious for bringing up a lot of edge-cases. Since my motivation is understanding and implementing a subset of the Bittorrent protocol (and not focus on the intricacies of the reflect API) I dediced on a different approach.

The strategy consists of:

1. First parsing the entire byte stream into some basic (usually loose) type (like `[]any`, `map[string]any` etc) and
2. Then use a specilized library (like `go-vip/mapstructure`) to map them to the concrete fields of a struct (aka. transform that first `map[string]any` into a usable `TrackerResponse`)

This separation of concerns allows me to handle the responsability of mapping the types to a more specialized library (or implement custom mapping logic) and fully concern myself with the Bittorrent protocol. It has a small disadvantage: it is necessary to parse the entire stream first (however, this is a small drawback since the Bittorrent protocol is very concise and most of the time it gives you only what you ask for).

The code responsible for the first step lives in `/bencoding`. It also provides aliases to those intermediate representation with loose types (so `[]any` becomes BList, `map[string]any` becomes BDict etc).

The code responsible for the second step lives in `/serde` (which stands for "SERialize & DEserialize"). In there one will find multiple structs that map to protocol-specific objects (like the `serde.MetaInfo`). The package allows one to create them from a byte stream via `Deserialize*` methods (like `DeserializeMetaInfo`) or turn them back into a byte stream via `Serialize*` methods. Moreover, it also provides helper methods that make it a little bit more convenient to handle the mapping between loose types and bytes (examples of that are `Decode` and `Encode`).

## Protocol Implementation

The code responsible for actually using the aformentioned structs in order to communicate with other nodes from the network is in `/protocol`.

An important struct implemented there is the `HTTPTracker` which allows one to communicate with a tracker. I've made the design decision all structs enabling the communication to another node should contains all the necessary information to issue VALID requests to that party (this is the reason why `HTTPTracker` also stores a `port` even if you will only use it for downloading).

# Future work

Below are some features that could be added into the future:

1. UDP Tracker (as a lot more trackers support it)
2. Seeding (it is only fair to further support the network if they support us)
3. Support for uTP
4. Support for DHT and Magnet Links

# Other design notes

## BENCODING

There are two main approaches for implementing the decoder: a recursion-based decoder or a state-machine-based decoder.

The first approach is slower (due to the recursion) and can cause a stack overflow if the decoded structure is deeply nested. However, it is much easier to implement.

The second approach is much faster at runtime, but it is very difficult to implement (not to mention error-prone).

Despite all that, I choose to implement a recursion-based decoder. Why? Because in real life, only an attacker's payload would be so deep as to trigger an overflow and we can prevent that by simply tracking the recusion's depth and returning an error if it crosses a specific boundry. Regarding the performance penalties caused by recursion, the program will spend the majority of its time downloading chunks instead of parsing bencoded strings (if we were to build a high-performance tracker, the story would've been different). As such, a recursion-based decoder represents a good choice for this particular project.

### Tradeoffs

During implementations, some tradeoffs were made:

- DecodeInt() returns an `int` instead of `math/big.Int` even though the specs say that the integer could be of any size:
  Reasoning:
  1. I can parse the string into an int without too much of a hustle using Go's Atoi
  2. In the real world you will almost never see an int value higher than 2^63-1
  3. Functions that use DecodeInt() will work with a familiar type: int

- DecodeString() doesn't check for the keys to be ordered alphabetically
  Reasoning:
  1. Because it is a waste of cpu cycles

- The Encode* functions assume they receive a value of type bencoding.B*. If this is not the case, they return an error
  Reasoning:
  1. Adding support for arbitrary values (like how json decoder does) is quite difficult and it requires treating a lot of edge cases. Given that I need this package just to perform basic I/O with other peers on the network, a simpler implementation is easier to code, maintain and debug

## MetaInfo Parsing

After I've implemented the bencoding decoder (and encoder) I started working on the parser for MetaInfo. I chose to simply parse the stream into a BDict and then map all the entries in there to an actual struct. That was tedious since you have to do typechecks at every step of the way and the code quickly becomes convoluted. I could've implemented a custom mapper that takes as input a `any` and a pointer to a struct and using the magic of reflection it populates it, but this is so error prone and so out of scope for this project that I dediced I better stick with the manual typechecking.

After some research I found `go-vip/mapstructure`, which seems to do what I need: map `any` to a concrete struct. If the need later arises, I will for sure use it, but at the moment the code I wrote is quite robust and ready-to-use, so I am not going to refactor it.

## Tracker Response Parsing

- HTTPTracker.RequestPeers() ignores the PeerID
  Reasoning:
  1. There is absolutely no need for that value whatsoever

## Errors

The current error hierarchy (as Gemini understood it because I was too tired to write it off):

```ascii
+-------------------------------------------------------------------------+
| Layer 3: Protocol Domain (protocol package)                             |
| High-level client domain errors:                                        |
|   - ErrUnsupportedTracker                                               |
|   - ErrTrackerReportedFailure (wraps tracker's failure message)         |
|   - ErrMalformedTrackerResponse (wraps underlying parser/format errors) |
|   - Network / IO errors (e.g., connection refused, read timeout)        |
+------------------------------------+------------------------------------+
                                     |
                                     v
+------------------------------------+------------------------------------+
| Layer 2: Semantic Serialization (serde package)                         |
| BitTorrent metadata mapping & schema validation:                        |
|   - ErrMalformedMetaInfo & MalformedMetaInfoError                       |
|   - ErrKeyNotFound & KeyNotFoundError                                   |
|   - ErrTypeMismatch                                                     |
+------------------------------------+------------------------------------+
                                     |
                                     v
+------------------------------------+------------------------------------+
| Layer 1: Primitive Bencoding (bencoding package)                        |
| Low-level byte-stream parser errors:                                    |
|   - ErrMalformedInput                                                   |
|   - ErrInvalidStringLength                                              |
|   - ErrInvalidIntegerValue                                              |
|   - ErrUnsupportedType                                                  |
|   - io.ErrUnexpectedEOF                                                 |
+-------------------------------------------------------------------------+
```
