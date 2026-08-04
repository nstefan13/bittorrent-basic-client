# BENCODING

Given the current scope of this project, I concern myself with only implementing the decoder.

There are two main approaches: a recursion-based decoder or a state-machine-based decoder.

The first approach is slower (due to the recursion) and can cause a stack overflow if the decoded structure is deeply nested. However, it is much easier to implement.

The second approach is much faster at runtime, but it is very difficult to implement (not to mention error-prone).

Despite all that, I choose to implement a recursion-based decoder. Why? Because in real life, only an attacker's payload would be so deep as to trigger an overflow and we can prevent that by simply tracking the recusion's depth and returning an error if it crosses a specific boundry. Regarding the performance penalties caused by recursion, the program will spend the majority of its time downloading chunks instead of parsing bencoded strings (if we were to build a high-performance tracker, the story would've been different). As such, a recursion-based decoder represents a good choice for this particular project.
