# BENCODING

Given the current scope of this project, I concern myself with only implementing the decoder.

There are two main approaches: a recursion-based decoder or a state-machine-based decoder.

The first approach is slower (due to the recursion) and can cause a stack overflow if the decoded structure is deeply nested. However, it is much easier to implement.

The second approach is much faster at runtime, but it is very difficult to implement (not to mention error-prone).

Despite all that, I choose to implement a recursion-based decoder. Why? Because in real life, only an attacker's payload would be so deep as to trigger an overflow and we can prevent that by simply tracking the recusion's depth and returning an error if it crosses a specific boundry. Regarding the performance penalties caused by recursion, the program will spend the majority of its time downloading chunks instead of parsing bencoded strings (if we were to build a high-performance tracker, the story would've been different). As such, a recursion-based decoder represents a good choice for this particular project.

## Tradeoffs

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
