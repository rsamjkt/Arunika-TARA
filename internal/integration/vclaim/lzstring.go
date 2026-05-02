package vclaim

import (
	"fmt"
	"strings"
)

// lzDecompressURIComponent mendekripsi output LZString.compressToEncodedURIComponent.
//
// BPJS mengompresi plaintext JSON dengan LZString sebelum AES encrypt.
// Ported dari LZString JS v1.5.0 (MIT) — pieroxy/lz-string.
func lzDecompressURIComponent(compressed string) (string, error) {
	if compressed == "" {
		return "", nil
	}
	compressed = strings.ReplaceAll(compressed, " ", "+")
	return lzDecompress(compressed, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+-$", 32)
}

func lzDecompress(compressed, alphabet string, resetValue int) (string, error) {
	// Build reverse-lookup table: char → bit value
	revMap := [256]int{}
	for i := range revMap {
		revMap[i] = -1
	}
	for i := 0; i < len(alphabet); i++ {
		revMap[alphabet[i]] = i
	}

	// Bit-stream state (mirrors JS data object)
	dsVal := revMap[compressed[0]]
	dsPos := resetValue
	dsIdx := 1

	readBits := func(n int) (int, error) {
		bits := 0
		power := 1
		maxPow := 1 << n
		for power < maxPow {
			resb := dsVal & dsPos
			dsPos >>= 1
			if dsPos == 0 {
				dsPos = resetValue
				if dsIdx < len(compressed) {
					dsVal = revMap[compressed[dsIdx]]
				}
				dsIdx++
			}
			if resb > 0 {
				bits |= power
			}
			power <<= 1
		}
		return bits, nil
	}

	// dictionary[0..2] are placeholder ints (never used as string refs)
	// dictionary[3..] are actual string entries
	dict := make([]string, 4) // pre-allocate first 4 slots
	dictSize := 4
	numBits := 3
	enlargeIn := 4

	// Read first token: 2 bits → type of first character
	next, _ := readBits(2)
	var firstChar string
	switch next {
	case 0:
		b, _ := readBits(8)
		firstChar = string(rune(b))
	case 1:
		b, _ := readBits(16)
		firstChar = string(rune(b))
	case 2:
		return "", nil
	default:
		return "", fmt.Errorf("lz: invalid first token %d", next)
	}
	dict[3] = firstChar
	w := firstChar

	var result strings.Builder
	result.WriteString(firstChar)

	for {
		if dsIdx > len(compressed)+1 {
			return "", fmt.Errorf("lz: premature end")
		}

		cv, _ := readBits(numBits)

		var entry string
		switch cv {
		case 0: // new 8-bit literal
			b, _ := readBits(8)
			ns := string(rune(b))
			dict = append(dict, ns)
			dictSize++
			cv = dictSize - 1
			enlargeIn--
			if enlargeIn == 0 {
				enlargeIn = 1 << numBits
				numBits++
			}
			entry = ns

		case 1: // new 16-bit literal
			b, _ := readBits(16)
			ns := string(rune(b))
			dict = append(dict, ns)
			dictSize++
			cv = dictSize - 1
			enlargeIn--
			if enlargeIn == 0 {
				enlargeIn = 1 << numBits
				numBits++
			}
			entry = ns

		case 2: // end of stream
			return result.String(), nil

		default: // dictionary reference
			if cv < len(dict) && dict[cv] != "" {
				entry = dict[cv]
			} else if cv == dictSize {
				// w[0] repeated
				runes := []rune(w)
				if len(runes) == 0 {
					return "", fmt.Errorf("lz: w is empty at ref %d", cv)
				}
				entry = w + string(runes[0])
			} else {
				return "", fmt.Errorf("lz: invalid ref cv=%d dictSize=%d", cv, dictSize)
			}
		}

		result.WriteString(entry)

		// Add new dictionary entry: w + entry[0]
		entryRunes := []rune(entry)
		if len(entryRunes) == 0 {
			return "", fmt.Errorf("lz: empty entry")
		}
		dict = append(dict, w+string(entryRunes[0]))
		dictSize++
		w = entry

		enlargeIn--
		if enlargeIn == 0 {
			enlargeIn = 1 << numBits
			numBits++
		}
	}
}
