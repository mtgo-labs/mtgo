package tg

import "bytes"

// This file adds two TL constructors that are present in the MTProto schema
// (telegram_api.tl) but were omitted from mtgo's generated TL layer because
// they are deprecated/legacy and not used by the high-level client:
//
//	inputPeerPhotoFileLocationLegacy#27d69997 flags:# big:flags.0?true peer:InputPeer volume_id:long local_id:int = InputFileLocation;
//	inputStickerSetThumbLegacy#dbaeae9 stickerset:InputStickerSet volume_id:long local_id:int = InputFileLocation;
//
// They are hand-written alongside the generated code to avoid regenerating the
// entire package for two constructors. Each follows the exact generated pattern:
// struct → SetFlags (if flags) → ConstructorID → Encode → Decode → Registry init.

// isInputFileLocation marks InputPeerPhotoFileLocationLegacy as implementing the InputFileLocationClass interface.
func (*InputPeerPhotoFileLocationLegacy) isInputFileLocation() {}

// isInputFileLocation marks InputStickerSetThumbLegacy as implementing the InputFileLocationClass interface.
func (*InputStickerSetThumbLegacy) isInputFileLocation() {}

// --- InputPeerPhotoFileLocationLegacy ---

// InputPeerPhotoFileLocationLegacyTypeID is the constructor ID for
// inputPeerPhotoFileLocationLegacy (0x27d69997).
const InputPeerPhotoFileLocationLegacyTypeID = 0x27d69997

// InputPeerPhotoFileLocationLegacy represents the TL constructor
// inputPeerPhotoFileLocationLegacy (0x27d69997). It is the legacy variant of
// inputPeerPhotoFileLocation that carries a volume_id + local_id instead of a
// photo_id, used for dialog/profile photos from the pre-photo-id era.
type InputPeerPhotoFileLocationLegacy struct {
	Flags    Fields         `json:"-"`
	Big      bool           `json:"big,omitempty"`
	Peer     InputPeerClass `json:"peer,omitempty"`
	VolumeID int64          `json:"volume_id,omitempty"`
	LocalID  int32          `json:"local_id,omitempty"`
}

// SetFlags computes flags from non-zero optional fields.
func (v *InputPeerPhotoFileLocationLegacy) SetFlags() {
	if v.Big {
		v.Flags.Set(0)
	}
}

// ConstructorID returns the TL constructor identifier 0x27d69997.
func (v *InputPeerPhotoFileLocationLegacy) ConstructorID() uint32 {
	return InputPeerPhotoFileLocationLegacyTypeID
}

// Encode serializes InputPeerPhotoFileLocationLegacy to a bytes.Buffer.
func (v *InputPeerPhotoFileLocationLegacy) Encode(b *bytes.Buffer) error {
	WriteInt(b, InputPeerPhotoFileLocationLegacyTypeID)
	v.SetFlags()
	WriteInt(b, uint32(v.Flags))
	if err := EncodeTLObject(b, v.Peer); err != nil {
		return err
	}
	WriteLong(b, v.VolumeID)
	WriteInt(b, uint32(v.LocalID))
	return nil
}

// DecodeInputPeerPhotoFileLocationLegacy deserializes from a reader.
func DecodeInputPeerPhotoFileLocationLegacy(r *Reader) (*InputPeerPhotoFileLocationLegacy, error) {
	v := &InputPeerPhotoFileLocationLegacy{}
	_f, err := r.ReadUint32()
	if err != nil {
		return nil, err
	}
	v.Flags = Fields(_f)
	v.Big = v.Flags.Has(0)
	obj, err := ReadTLObject(r)
	if err != nil {
		return nil, err
	}
	v.Peer = obj.(InputPeerClass)
	vid, err := r.ReadInt64()
	if err != nil {
		return nil, err
	}
	v.VolumeID = vid
	lid, err := r.ReadInt32()
	if err != nil {
		return nil, err
	}
	v.LocalID = lid
	return v, nil
}

func init() {
	Registry[InputPeerPhotoFileLocationLegacyTypeID] = func(r *Reader) (TLObject, error) {
		return DecodeInputPeerPhotoFileLocationLegacy(r)
	}
}

// --- InputStickerSetThumbLegacy ---

// InputStickerSetThumbLegacyTypeID is the constructor ID for
// inputStickerSetThumbLegacy (0x0dbaeae9).
const InputStickerSetThumbLegacyTypeID = 0x0dbaeae9

// InputStickerSetThumbLegacy represents the TL constructor
// inputStickerSetThumbLegacy (0x0dbaeae9). It is the legacy variant of
// inputStickerSetThumb that carries a volume_id + local_id instead of a
// thumb_version, used for sticker-set thumbnails from the pre-version era.
type InputStickerSetThumbLegacy struct {
	Stickerset InputStickerSetClass `json:"stickerset,omitempty"`
	VolumeID   int64                `json:"volume_id,omitempty"`
	LocalID    int32                `json:"local_id,omitempty"`
}

// ConstructorID returns the TL constructor identifier 0x0dbaeae9.
func (v *InputStickerSetThumbLegacy) ConstructorID() uint32 {
	return InputStickerSetThumbLegacyTypeID
}

// Encode serializes InputStickerSetThumbLegacy to a bytes.Buffer.
func (v *InputStickerSetThumbLegacy) Encode(b *bytes.Buffer) error {
	WriteInt(b, InputStickerSetThumbLegacyTypeID)
	if err := EncodeTLObject(b, v.Stickerset); err != nil {
		return err
	}
	WriteLong(b, v.VolumeID)
	WriteInt(b, uint32(v.LocalID))
	return nil
}

// DecodeInputStickerSetThumbLegacy deserializes from a reader.
func DecodeInputStickerSetThumbLegacy(r *Reader) (*InputStickerSetThumbLegacy, error) {
	v := &InputStickerSetThumbLegacy{}
	obj, err := ReadTLObject(r)
	if err != nil {
		return nil, err
	}
	v.Stickerset = obj.(InputStickerSetClass)
	vid, err := r.ReadInt64()
	if err != nil {
		return nil, err
	}
	v.VolumeID = vid
	lid, err := r.ReadInt32()
	if err != nil {
		return nil, err
	}
	v.LocalID = lid
	return v, nil
}

func init() {
	Registry[InputStickerSetThumbLegacyTypeID] = func(r *Reader) (TLObject, error) {
		return DecodeInputStickerSetThumbLegacy(r)
	}
}
