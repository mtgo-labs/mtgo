package telegram

type updateState struct {
	Pts  int32
	Qts  int32
	Date int32
	Seq  int32
}

type channelState struct {
	ChannelID  int64
	Pts        int32
	recovering bool
}

type updateMeta struct {
	Key        string
	Pts        int32
	PtsCount   int32
	Qts        int32
	SeqStart   int32
	Seq        int32
	Date       int32
	ChannelID  int64
	ChannelPts int32
	IsChannel  bool
}

type gapKind int

const (
	noGap gapKind = iota
	duplicateUpdate
	accountPtsGap
	accountSeqGap
	accountQtsGap
	channelPtsGap
)
