package libkataware

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestHeaderSerialization(t *testing.T) {
	rawhdrs, err := base64.StdEncoding.DecodeString(
		`AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAO6Pt/Xp7ErJ6xyw+Z3aPYX/IG8OI
ilEyOp+4qkseXkopq19J//8AHR2sK3wBAAAAb+KMCrbxs3LBpqJGrmP3T5Meg2XhWgicaNYZAAAA
AACYIFH9HkunRLu+aA4f7hRne6Gjw1QL97HNtgboVyM+DmG8Zkn//wAdAeNimQ==`)
	if err != nil {
		panic(err.Error())
	}
	var h1, h2 Header
	h1.Deserialize(rawhdrs[:80])
	h2.Deserialize(rawhdrs[80:])
	if subtle.ConstantTimeCompare(append(h1.Serialize(), h2.Serialize()...), rawhdrs) != 1 {
		t.Error("serialize and deserialize did not give back the original")
	}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("did not panic when fed wrong length to deserialize")
			}
		}()
		h1.Deserialize(nil)
	}()
}

func TestMerkle(t *testing.T) {
	dehex := func(s string) []byte {
		res, _ := hex.DecodeString(s)
		return res
	}
	dehexle := func(s string) []byte {
		return SwapBytes(dehex(s))
	}
	var hdr Header
	hdr.HashMerkleRoot =
		dehexle("5463bb0d10a4ff6d91fbe7f7a084ed4b2f313cc08eb58808fdee259aa547f923")
	var tx Transaction
	tx.Unpack(bytes.NewReader(dehex(
		`0100000001e6b97bce8d7522de14a79e206a035ba328c9ba1a7ea6bbf8353efde217fdb5c5020000008a473044022011cd3406663fddba853090f621555e2d422e229d25fb245b5cc93ba85746e60402200ddba7b8d5ce1445577bb7641ddf875bf431e29fcbe8cc57e6873e80185645d9014104e9c4c6ad2be6d97bd4bbba18fc51e62a0b4b393735fb8c0fb859253aaf4427789ce6d4d1b653c29d67b99e9dcca1fbfaf1ab86aebe50b5b277ecd8ba8dbcac65ffffffff0354150000000000001976a914b22a5a0f48c42a0219a4bfe146e2a2432d9f9e1388ac31b80000000000001976a9140044b6662b972525f7fb6c2b40d51aa4cb56bc5d88acdbc20000000000001976a9140044b6662b972525f7fb6c2b40d51aa4cb56bc5d88ac00000000`)))
	res := func() bool {
		return hdr.CheckExists([][]byte{
			dehexle("d844420b0f01398953b809b844bc9a5987f41d1373dab3180b6b4fe4de8633c4"),
			dehexle("3c1f63dae13e84aaba94d6ee12c7b48fa7b470eacfbe6e183ba008b7cbc3725c"),
			dehexle("0c5141f62f1ad58a6ebcee98868451ff392bc49bfb6daa5e4f492b6d175812cf"),
			dehexle("e414fd7ee39a83b7bd524f98ed09efa279a97fecbeeca68087de384eda71fc31"),
			dehexle("47fbf181989f2549d0d7a9fee2337ea60295059df0c0267cca173fd43b8e8596"),
			dehexle("bc441c955a6d9bc629e3a8e428aa7670d2bf631a9d11887bb4ad6b534fcf4817"),
			dehexle("1985800e12a150e6196ac07119278ea84ad296f1d8b2c88b2c85f61035f6a7cd"),
			dehexle("fc02dfab3083ca9185bce4e8b9357b5aacacce2172c969ac543da04eaa1c0c1d"),
			dehexle("6a36f10f34da6d66fdf75d4d97e0796106e7594409d354b8419a248231ab6935"),
			dehexle("cbc02af67b294fe7bad08a69f9435b520f8f89fa221583e0a84c8d9c9dc469a0"),
			dehexle("1043c7a6b9b4c42243e84a49a1f40476692b5d36d2f64dfb14409490ca3dfad2"),
			dehexle("fbbaec1ca9d29ba9ee6a89214428556827b4e5a254de59de5adabbabb2621f54"),
		}, 2917, tx)
	}
	if res() == false {
		t.Fail()
	}
	rand.Read(hdr.HashMerkleRoot)
	if res() == true {
		t.Fail()
	}
}
