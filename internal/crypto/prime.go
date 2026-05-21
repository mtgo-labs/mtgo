package crypto

import (
	"crypto/rand"
	"math/big"
)

// CurrentDHPrime is the 2048-bit safe prime used for Diffie-Hellman key exchange
// in MTProto authorization. See https://core.telegram.org/mtproto/auth_key.
var CurrentDHPrime = newBigIntFromHex(
	"C71CAEB9C6B1C9048E6C522F70F13F73980D40238E3E21C14934D037563D930F" +
		"48198A0AA7C14058229493D22530F4DBFA336F6E0AC925139543AED44CCE7C37" +
		"20FD51F69458705AC68CD4FE6B6B13ABDC9746512969328454F18FAF8C595F64" +
		"2477FE96BB2A941D5BCD1D4AC8CC49880708FA9B378E3C4F3A9060BEE67CF9A4" +
		"A4A695811051907E162753B56B0F6B410DBA74D8A84B2A14B3144E0EF1284754" +
		"FD17ED950D5965B4B9DD46582DB1178D169C6BC465B0D6FF9CA3928FEF5B9AE4" +
		"E418FC15E83EBEA0F87FA9FF5EED70050DED2849F47BF959D956850CE929851F" +
		"0D8115F635B105EE2E4E15D04B2454BF6F4FADF034B10403119CD8E3B92FCC5B",
)

var smallPrimes = []int64{
	3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47,
	53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109,
	113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179,
	181, 191, 193, 197, 199, 211, 223, 227, 229, 233, 239, 251,
}

func newBigIntFromHex(hex string) *big.Int {
	n, ok := new(big.Int).SetString(hex, 16)
	if !ok {
		panic("crypto/prime: invalid hex")
	}
	return n
}

// Decompose factorizes a semiprime pq (product of two primes) into its smaller
// prime factor using Pollard's rho algorithm. It returns the smaller prime factor
// of pq, or 0 if factorization fails after 64 attempts.
//
// See https://core.telegram.org/mtproto/pq.
func Decompose(pq int64) int64 {
	if pq%2 == 0 {
		return 2
	}

	for _, p := range smallPrimes {
		if pq%p == 0 {
			return p
		}
	}

	n := big.NewInt(pq)
	nMinus1 := new(big.Int).Sub(n, big.NewInt(1))
	tmp := new(big.Int)
	bigOne := big.NewInt(1)

	for attempt := 0; attempt < 64; attempt++ {
		c, err := rand.Int(rand.Reader, nMinus1)
		if err != nil {
			continue
		}
		c.Add(c, bigOne)
		if c.Cmp(bigOne) == 0 {
			c.SetInt64(2)
		}
		y, err := rand.Int(rand.Reader, nMinus1)
		if err != nil {
			continue
		}
		y.Add(y, bigOne)
		g := big.NewInt(1)
		q := big.NewInt(1)
		r := int64(1)
		x := new(big.Int).Set(y)
		ys := new(big.Int)

		for g.Cmp(bigOne) == 0 {
			x.Set(y)
			for i := int64(0); i < r; i++ {
				tmp.Mul(y, y).Add(tmp, c).Mod(tmp, n)
				y.Set(tmp)
			}
			k := int64(0)
			for k < r && g.Cmp(bigOne) == 0 {
				ys.Set(y)
				limit := int64(128)
				if r-k < limit {
					limit = r - k
				}
				for i := int64(0); i < limit; i++ {
					tmp.Mul(y, y).Add(tmp, c).Mod(tmp, n)
					y.Set(tmp)
					tmp.Sub(x, y)
					if tmp.Sign() < 0 {
						tmp.Add(tmp, n)
					}
					q.Mul(q, tmp).Mod(q, n)
				}
				g.GCD(nil, nil, q, n)
				k += limit
			}
			r *= 2
		}

		if g.Cmp(n) == 0 {
			for {
				tmp.Mul(ys, ys).Add(tmp, c).Mod(tmp, n)
				ys.Set(tmp)
				tmp.Sub(x, ys)
				if tmp.Sign() < 0 {
					tmp.Add(tmp, n)
				}
				g.GCD(nil, nil, tmp, n)
				if g.Cmp(bigOne) > 0 {
					break
				}
			}
		}

		if g.Cmp(n) != 0 && g.Cmp(bigOne) != 0 {
			factor := g.Int64()
			other := pq / factor
			if other < factor {
				return other
			}
			return factor
		}
	}

	return 0
}
