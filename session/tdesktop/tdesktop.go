// Package tdesktop reads Telegram Desktop session data from a tdata directory.
//
// It decrypts the local storage files and extracts auth keys, DC info, and
// user identity — everything needed to reconstruct an MTProto session.
//
// Usage:
//
//	accounts, err := tdesktop.Read("/path/to/tdata", nil)
//	for _, acc := range accounts {
//	    fmt.Printf("UserID=%d MainDC=%d\n", acc.UserID, acc.MainDC)
//	    for dc, key := range acc.Keys {
//	        fmt.Printf("  DC %d: auth_key=%x...\n", dc, key[:8])
//	    }
//	}
package tdesktop

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"  // #nosec G501
	"crypto/sha1" // #nosec G505
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// Account holds a single Telegram account's session data extracted from tdata.
type Account struct {
	// UserID is the Telegram user ID.
	UserID uint64
	// MainDC is the primary datacenter ID for this account.
	MainDC int
	// Keys maps DC ID to 256-byte auth key.
	Keys map[int][256]byte
}

// String returns the main DC auth key as a Telethon-format session string.
// Pass the result to session.TelethonSession() or session.StringSession().
func (a *Account) String() string {
	key, ok := a.Keys[a.MainDC]
	if !ok {
		return ""
	}
	addr, port := ResolveDC(a.MainDC)
	return encodeTelethonString(a.MainDC, addr, port, key[:])
}

// ResolveDC returns the (address, port) for a Telegram production DC.
func ResolveDC(dcID int) (string, int) {
	addrs := map[int]string{
		1: "149.154.175.50",
		2: "149.154.167.50",
		3: "149.154.175.100",
		4: "149.154.167.91",
		5: "91.108.56.190",
	}
	addr, ok := addrs[dcID]
	if !ok {
		addr = "149.154.167.50"
	}
	return addr, 443
}

func encodeTelethonString(dcID int, addr string, port int, authKey []byte) string {
	var ip net.IP
	if addr != "" {
		ip = net.ParseIP(addr)
	}
	if ip == nil {
		ip = net.IPv4(0, 0, 0, 0)
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	buf := new(bytes.Buffer)
	buf.WriteByte(uint8(dcID))
	buf.Write(ip)
	_ = binary.Write(buf, binary.BigEndian, uint16(port))
	buf.Write(authKey)
	return "1" + base64.URLEncoding.EncodeToString(buf.Bytes())
}

// Read reads all accounts from a tdata directory on disk.
// passcode should be nil for unencrypted sessions, or the user's local passcode.
func Read(root string, passcode []byte) ([]Account, error) {
	return ReadFS(os.DirFS(root), passcode)
}

// ReadFS reads all accounts from a tdata directory using the given filesystem.
func ReadFS(root fs.FS, passcode []byte) ([]Account, error) {
	keyDataFile, err := openFile(root, "key_data")
	if err != nil {
		return nil, fmt.Errorf("open key_data: %w", err)
	}

	kd, err := readKeyData(keyDataFile, passcode)
	if err != nil {
		return nil, fmt.Errorf("read key_data: %w", err)
	}
	if len(kd.accountsIdx) == 0 {
		return nil, ErrNoAccounts
	}

	result := make([]Account, 0, len(kd.accountsIdx))
	for _, idx := range kd.accountsIdx {
		keyFile := fileKey("data")
		if idx > 0 {
			keyFile = fileKey(fmt.Sprintf("data#%d", idx+1))
		}

		mtpDataFile, err := openFile(root, keyFile)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", keyFile, err)
		}

		auth, err := readMTPAuthorization(mtpDataFile, kd.localKey)
		if err != nil {
			return nil, fmt.Errorf("read mtp auth: %w", err)
		}

		result = append(result, Account(auth))
	}

	return result, nil
}

var (
	errChecksumMismatch  = errors.New("checksum mismatch")
	errDataNotAligned    = errors.New("data not aligned to block size")
	errKeyInnerTooShort  = errors.New("key inner too short")
	errKeyInnerTruncated = errors.New("key inner truncated")
	errInfoTooShort      = errors.New("info too short")
	errDecryptedTooShort = errors.New("decrypted too short")
)

var (
	ErrNoAccounts = errors.New("tdesktop: no accounts found")
	ErrKeyDecrypt = errors.New("tdesktop: key data decrypt failed (wrong passcode?)")
)

// ---------- internal types ----------

type keyData struct {
	localKey    [256]byte
	accountsIdx []uint32
}

type mtpAuthorization struct {
	UserID uint64
	MainDC int
	Keys   map[int][256]byte
}

// ---------- constants ----------

const (
	magicTDF   = "TDF$"
	saltSize   = 32
	wideIDsTag = ^uint64(0) // 0xFFFFFFFFFFFFFFFF
	dbiMtpAuth = 0x4b

	pbkdf2StrongIter = 100000
)

// ---------- file format ----------

// tdesktopFile represents a decrypted Telegram Desktop storage file.
type tdesktopFile struct {
	data    []byte
	offset  int
	version uint32
}

// openFile tries to open a tdesktop file with suffixes "0", "1", "s".
func openFile(root fs.FS, name string) (*tdesktopFile, error) {
	for _, suffix := range []string{"0", "1", "s"} {
		p := name + suffix
		f, err := root.Open(p)
		if err != nil {
			if isNotExist(err) || isPermission(err) {
				continue
			}
			return nil, err
		}
		tdf, err := fromReader(f)
		_ = f.Close()
		if err != nil {
			if err == io.ErrUnexpectedEOF || isWrongMagic(err) {
				continue
			}
			return nil, err
		}
		return tdf, nil
	}
	return nil, fmt.Errorf("file %q not found", name)
}

func fromReader(r io.Reader) (*tdesktopFile, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if string(buf[:4]) != magicTDF {
		return nil, &wrongMagicError{buf[:4]}
	}
	version := binary.LittleEndian.Uint32(buf[4:8])

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}
	if len(data) < 16 {
		return nil, fmt.Errorf("data too short (%d)", len(data))
	}

	// Last 16 bytes are the MD5 checksum.
	storedHash := data[len(data)-16:]
	data = data[:len(data)-16]

	computed := computeFileHash(data, version)
	if !bytes.Equal(computed[:], storedHash) {
		return nil, errChecksumMismatch
	}

	return &tdesktopFile{data: data, version: version}, nil
}

func (f *tdesktopFile) readArray() ([]byte, error) {
	if f.offset+4 > len(f.data) {
		return nil, io.ErrUnexpectedEOF
	}
	length := binary.BigEndian.Uint32(f.data[f.offset:])
	f.offset += 4
	if length == 0xFFFFFFFF {
		return nil, nil
	}
	if f.offset+int(length) > len(f.data) {
		return nil, io.ErrUnexpectedEOF
	}
	data := f.data[f.offset : f.offset+int(length)]
	f.offset += int(length)
	return data, nil
}

// ---------- key derivation ----------

func createLocalKey(passcode, salt []byte) [256]byte {
	iters := 1
	if len(passcode) > 0 {
		iters = pbkdf2StrongIter
	}
	h := sha512.New()
	h.Write(salt)
	h.Write(passcode)
	h.Write(salt)
	password := h.Sum(nil)

	key := pbkdf2.Key(password, salt, iters, 256, sha512.New)
	var result [256]byte
	copy(result[:], key)
	return result
}

// ---------- AES-IGE ----------

func decryptLocal(encrypted []byte, localKey [256]byte) ([]byte, error) {
	if len(encrypted) < 32 || len(encrypted)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid encrypted data length %d", len(encrypted))
	}
	// First 16 bytes are msg_key.
	var msgKey [16]byte
	copy(msgKey[:], encrypted[:16])
	ciphertext := encrypted[16:]

	aesKey, aesIV := deriveAESKeyIV(localKey, msgKey, 8)
	plaintext, err := igeDecrypt(aesKey, aesIV, ciphertext)
	if err != nil {
		return nil, err
	}

	// Verify msg_key.
	hash := sha1.Sum(plaintext)
	if !bytes.Equal(hash[:16], msgKey[:]) {
		return nil, ErrKeyDecrypt
	}
	return plaintext, nil
}

// deriveAESKeyIV computes the AES key and IV from auth_key and msg_key.
// x is the offset into auth_key (8 for server-client, 0 for client-server).
func deriveAESKeyIV(authKey [256]byte, msgKey [16]byte, x int) (key [32]byte, iv [32]byte) {
	// SHA1(msg_key + auth_key[x:x+32])
	sha1A := sha1.Sum(append(msgKey[:], authKey[x:x+32]...))
	// SHA1(auth_key[x+32:x+48] + msg_key + auth_key[x+48:x+64])
	sha1B := sha1.Sum(append(append(authKey[x+32:x+48], msgKey[:]...), authKey[x+48:x+64]...))
	// SHA1(auth_key[x+64:x+96] + msg_key)
	sha1C := sha1.Sum(append(authKey[x+64:x+96], msgKey[:]...))
	// SHA1(msg_key + auth_key[x+96:x+128])
	sha1D := sha1.Sum(append(msgKey[:], authKey[x+96:x+128]...))

	copy(key[0:8], sha1A[0:8])
	copy(key[8:20], sha1B[8:20])
	copy(key[20:32], sha1C[4:16])

	copy(iv[0:12], sha1A[8:20])
	copy(iv[12:20], sha1B[0:8])
	copy(iv[20:24], sha1C[16:20])
	copy(iv[24:32], sha1D[0:8])

	return key, iv
}

func igeDecrypt(key [32]byte, iv [32]byte, data []byte) ([]byte, error) {
	c, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errDataNotAligned
	}

	bs := aes.BlockSize
	plaintext := make([]byte, len(data))

	xPrev := iv[:bs]
	yPrev := iv[bs:]

	for i := 0; i < len(data); i += bs {
		ctBlock := data[i : i+bs]
		var temp [aes.BlockSize]byte
		for j := 0; j < bs; j++ {
			temp[j] = ctBlock[j] ^ yPrev[j]
		}
		dec := make([]byte, bs)
		c.Decrypt(dec, temp[:])
		for j := 0; j < bs; j++ {
			plaintext[i+j] = dec[j] ^ xPrev[j]
		}
		xPrev = ctBlock
		yPrev = plaintext[i : i+bs]
	}

	return plaintext, nil
}

// ---------- file hashing ----------

func computeFileHash(data []byte, version uint32) [16]byte {
	h := md5.New() // #nosec G401
	h.Write(data)
	var packedLen [4]byte
	binary.LittleEndian.PutUint32(packedLen[:], uint32(len(data)))
	h.Write(packedLen[:])
	var ver [4]byte
	binary.LittleEndian.PutUint32(ver[:], version)
	h.Write(ver[:])
	h.Write([]byte(magicTDF))
	var result [16]byte
	copy(result[:], h.Sum(nil))
	return result
}

// ---------- MD5 file naming ----------

func fileKey(s string) string {
	hash := md5.Sum([]byte(s)) // #nosec G401
	// Swap nibbles.
	for i := range hash {
		hash[i] = hash[i]<<4 | hash[i]>>4
	}
	return strings.ToUpper(fmt.Sprintf("%x", hash[:]))[:16]
}

// ---------- key_data parsing ----------

func readKeyData(tgf *tdesktopFile, passcode []byte) (keyData, error) {
	salt, err := tgf.readArray()
	if err != nil {
		return keyData{}, fmt.Errorf("read salt: %w", err)
	}
	if len(salt) != saltSize {
		return keyData{}, fmt.Errorf("invalid salt length %d", len(salt))
	}

	passcodeKey := createLocalKey(passcode, salt)

	keyEncrypted, err := tgf.readArray()
	if err != nil {
		return keyData{}, fmt.Errorf("read key_encrypted: %w", err)
	}

	keyInner, err := decryptLocal(keyEncrypted, passcodeKey)
	if err != nil {
		return keyData{}, fmt.Errorf("decrypt key_encrypted: %w", err)
	}

	// The decrypted key is a Qt-style byte array (4-byte BE length + data).
	if len(keyInner) < 4 {
		return keyData{}, errKeyInnerTooShort
	}
	keyLen := int(binary.BigEndian.Uint32(keyInner))
	if 4+keyLen > len(keyInner) {
		return keyData{}, errKeyInnerTruncated
	}
	keyBytes := keyInner[4 : 4+keyLen]

	var localKey [256]byte
	if len(keyBytes) < 256 {
		return keyData{}, fmt.Errorf("key too small (%d)", len(keyBytes))
	}
	copy(localKey[:], keyBytes)

	infoEncrypted, err := tgf.readArray()
	if err != nil {
		return keyData{}, fmt.Errorf("read info_encrypted: %w", err)
	}
	infoDecrypted, err := decryptLocal(infoEncrypted, localKey)
	if err != nil {
		return keyData{}, ErrKeyDecrypt
	}

	// Skip length prefix, read account count.
	if len(infoDecrypted) < 8 {
		return keyData{}, errInfoTooShort
	}
	infoDecrypted = infoDecrypted[4:] // skip uint32 length
	count := int(binary.BigEndian.Uint32(infoDecrypted))
	infoDecrypted = infoDecrypted[4:]

	accountsIdx := make([]uint32, 0, count)
	for i := 0; i < count && i*4+4 <= len(infoDecrypted); i++ {
		idx := binary.BigEndian.Uint32(infoDecrypted[i*4:])
		accountsIdx = append(accountsIdx, idx)
	}

	return keyData{localKey: localKey, accountsIdx: accountsIdx}, nil
}

// ---------- MTPAuthorization parsing ----------

func readMTPAuthorization(tgf *tdesktopFile, localKey [256]byte) (mtpAuthorization, error) {
	encrypted, err := tgf.readArray()
	if err != nil {
		return mtpAuthorization{}, fmt.Errorf("read encrypted: %w", err)
	}
	decrypted, err := decryptLocal(encrypted, localKey)
	if err != nil {
		return mtpAuthorization{}, fmt.Errorf("decrypt: %w", err)
	}

	// Skip uint32 length prefix.
	if len(decrypted) < 4 {
		return mtpAuthorization{}, errDecryptedTooShort
	}
	decrypted = decrypted[4:]

	return parseMTPAuthorization(decrypted)
}

func parseMTPAuthorization(data []byte) (mtpAuthorization, error) {
	r := newReader(data)

	// dbiMtpAuthorization marker.
	id, err := r.readUint32()
	if err != nil {
		return mtpAuthorization{}, fmt.Errorf("read id: %w", err)
	}
	if id != dbiMtpAuth {
		return mtpAuthorization{}, fmt.Errorf("unexpected id %d (expected %d)", id, dbiMtpAuth)
	}

	// Skip main length.
	if err := r.skip(4); err != nil {
		return mtpAuthorization{}, err
	}

	legacyUserID, err := r.readUint32()
	if err != nil {
		return mtpAuthorization{}, fmt.Errorf("read legacy user id: %w", err)
	}
	legacyMainDC, err := r.readUint32()
	if err != nil {
		return mtpAuthorization{}, fmt.Errorf("read legacy main dc: %w", err)
	}

	var auth mtpAuthorization
	if (uint64(legacyUserID)<<32)|uint64(legacyMainDC) == wideIDsTag {
		userID, err := r.readUint64()
		if err != nil {
			return mtpAuthorization{}, fmt.Errorf("read user id: %w", err)
		}
		mainDC, err := r.readUint32()
		if err != nil {
			return mtpAuthorization{}, fmt.Errorf("read main dc: %w", err)
		}
		auth.UserID = userID
		auth.MainDC = int(mainDC)
	} else {
		auth.UserID = uint64(legacyUserID)
		auth.MainDC = int(legacyMainDC)
	}

	keyCount, err := r.readUint32()
	if err != nil {
		return mtpAuthorization{}, fmt.Errorf("read key count: %w", err)
	}
	// keyCount is attacker-controlled untrusted input. Real tdata has only a
	// handful of DC keys; reject implausible counts and do not use the value as
	// an allocation hint, to avoid multi-GB map pre-allocation (DoS).
	const maxMTPKeys = 64
	if keyCount > maxMTPKeys {
		return mtpAuthorization{}, fmt.Errorf("implausible key count %d", keyCount)
	}

	auth.Keys = make(map[int][256]byte)
	for i := uint32(0); i < keyCount; i++ {
		dcID, err := r.readUint32()
		if err != nil {
			return mtpAuthorization{}, fmt.Errorf("read dc id %d: %w", i, err)
		}
		var key [256]byte
		if err := r.consumeN(key[:]); err != nil {
			return mtpAuthorization{}, fmt.Errorf("read auth key %d: %w", i, err)
		}
		auth.Keys[int(dcID)] = key
	}

	return auth, nil
}

// ---------- binary reader ----------

type reader struct {
	buf []byte
}

func newReader(data []byte) reader {
	return reader{buf: data}
}

func (r *reader) readUint32() (uint32, error) {
	if len(r.buf) < 4 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint32(r.buf)
	r.buf = r.buf[4:]
	return v, nil
}

func (r *reader) readUint64() (uint64, error) {
	if len(r.buf) < 8 {
		return 0, io.ErrUnexpectedEOF
	}
	v := binary.BigEndian.Uint64(r.buf)
	r.buf = r.buf[8:]
	return v, nil
}

func (r *reader) consumeN(target []byte) error {
	if len(r.buf) < len(target) {
		return io.ErrUnexpectedEOF
	}
	copy(target, r.buf[:len(target)])
	r.buf = r.buf[len(target):]
	return nil
}

func (r *reader) skip(n int) error {
	if len(r.buf) < n {
		return io.ErrUnexpectedEOF
	}
	r.buf = r.buf[n:]
	return nil
}

// ---------- error helpers ----------

type wrongMagicError struct {
	magic []byte
}

func (e *wrongMagicError) Error() string {
	return fmt.Sprintf("wrong magic %x", e.magic)
}

func isWrongMagic(err error) bool {
	_, ok := err.(*wrongMagicError)
	return ok
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "file does not exist") ||
		strings.Contains(msg, "no such file") ||
		strings.HasSuffix(msg, "not found")
}

func isPermission(err error) bool {
	return err != nil && strings.Contains(err.Error(), "permission")
}
