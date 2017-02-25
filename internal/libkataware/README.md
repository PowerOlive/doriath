## simple library for accessing bits and pieces of Bitcoin data structures

libkataware is used throughout Doriath as a collection of simply implemented helper functions for parsing and operating on data structures found in Bitcoin. It aims to have no external dependencies, and it's not intended to be a complete "Bitcoin implementation" or anything like that. It *might* be useful for other people, but might not, which is why it is in `internal`.

*kataware* (片割れ) is the Japanese word for "fragment", as libkataware deals with picking apart blocks and transactions, the smallest fragments of the Bitcoin blockchain. It's also a reference to the anime film *Kimi no na wa*; libkataware makes it easier to implement Bitcoin logic correctly so that your transaction history does not suddenly get rewritten because of an implementation error.
