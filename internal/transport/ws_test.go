package transport

import "testing"

func TestInvalidObfuscated2Nonce(t *testing.T) {
	tests := []struct {
		name  string
		nonce []byte
		want  bool
	}{
		{
			name:  "short",
			nonce: []byte{},
			want:  true,
		},
		{
			name: "abridged marker",
			nonce: testObfuscated2Nonce(func(nonce []byte) {
				nonce[0] = 0xEF
			}),
			want: true,
		},
		{
			name: "padded intermediate marker",
			nonce: testObfuscated2Nonce(func(nonce []byte) {
				copy(nonce[:4], []byte{0xDD, 0xDD, 0xDD, 0xDD})
			}),
			want: true,
		},
		{
			name: "intermediate marker",
			nonce: testObfuscated2Nonce(func(nonce []byte) {
				copy(nonce[:4], []byte{0xEE, 0xEE, 0xEE, 0xEE})
			}),
			want: true,
		},
		{
			name: "post method",
			nonce: testObfuscated2Nonce(func(nonce []byte) {
				copy(nonce[:4], "POST")
			}),
			want: true,
		},
		{
			name: "tls prefix",
			nonce: testObfuscated2Nonce(func(nonce []byte) {
				copy(nonce[:4], []byte{0x16, 0x03, 0x01, 0x02})
			}),
			want: true,
		},
		{
			name: "zero second word",
			nonce: testObfuscated2Nonce(func(nonce []byte) {
				copy(nonce[:4], "ABCD")
				copy(nonce[4:8], []byte{0, 0, 0, 0})
			}),
			want: true,
		},
		{
			name:  "valid",
			nonce: testObfuscated2Nonce(nil),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invalidObfuscated2Nonce(tt.nonce)
			if got != tt.want {
				t.Fatalf("invalidObfuscated2Nonce() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testObfuscated2Nonce(setup func([]byte)) []byte {
	nonce := make([]byte, 64)
	copy(nonce[:8], "ABCDEFGH")
	if setup != nil {
		setup(nonce)
	}
	return nonce
}
