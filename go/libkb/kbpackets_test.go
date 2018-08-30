// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package libkb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBadMsgpack(t *testing.T) {
	var info NaclEncryptionInfo
	err := DecodeArmoredPacketBody(`
g6Rib2R5hqhkZXRhY2hlw6loYXNoX3R5cGUKo2tlecQjASBz6XLVJ/u0KKVpvp9QlcNvIopFsusm
wjFFHVUYo3ykIwqncGF5bG9hZMUD7XsiYm9keSI6eyJkZXZpY2UiOnsiaWQiOiIwNTc4NDNjMjAy
NTE5MjZhY2MwZDVkYjMxMjY5NzkxOCIsImtpZCI6IjAxMjEzNzlkNTM3MGFlYjRlOWY3YWE0YjUy
NzBmNWU0ODgwODVhNjQ3ZTBkYjI0NzdlZGQzZjNjNTBlM2QxNGY0MDRmMGEiLCJzdGF0dXMiOjF9
LCJrZXkiOnsiZWxkZXN0X2tpZCI6IjAxMjBjN2E5NGM0NDI3NjA5NWU3MzY1ZjQ5YjY3YWU2OGY3
YmNiYzgwNWM1ODI3OTdmNjkwYjY3YjA4ZjA1ZWRmZTFiMGEiLCJob3N0Ijoia2V5YmFzZS5pbyIs
ImtpZCI6IjAxMjA3M2U5NzJkNTI3ZmJiNDI4YTU2OWJlOWY1MDk1YzM2ZjIyOGE0NWIyZWIyNmMy
MzE0NTFkNTUxOGEzN2NhNDIzMGEiLCJ1aWQiOiJhMDkxNTUyMTA1MzJkOTYwODFjNGEzYTk5YzY0
ODQxOSIsInVzZXJuYW1lIjoibHdwcF80MjJhYjc5ODQ3In0sInN1YmtleSI6eyJraWQiOiIwMTIx
Mzc5ZDUzNzBhZWI0ZTlmN2FhNGI1MjcwZjVlNDg4MDg1YTY0N2UwZGIyNDc3ZWRkM2YzYzUwZTNk
MTRmNDA0ZjBhIiwicGFyZW50X2tpZCI6IjAxMjA3M2U5NzJkNTI3ZmJiNDI4YTU2OWJlOWY1MDk1
YzM2ZjIyOGE0NWIyZWIyNmMyMzE0NTFkNTUxOGEzN2NhNDIzMGEifSwidHlwZSI6InN1YmtleSIs
InZlcnNpb24iOjF9LCJjbGllbnQiOnsibmFtZSI6ImtleWJhc2UuaW8gZ28gY2xpZW50IiwidmVy
c2lvbiI6IjEuMC4wIn0sImN0aW1lIjoxNDQzNDY1OTQzLCJleHBpcmVfaW4iOjUwNDU3NjAwMCwi
bWVya2xlX3Jvb3QiOnsiY3RpbWUiOjE0NDM0NjU5NDMsImhhc2giOiI4ZWIzM2IwNmMxZTAyMjFh
Y2JhY2UzMDZkMjNlYTFkOTIyYmFhYWNjNmFiYmQyN2MzOWNmNjNiY2MyNjczNGVmNDg1MGNmODc2
YWZlNjkxNTc4NDk4NDEzZTJlOTc0M2I3OWI5N2JlMTYxZWMwOGRiNGNmMjk2YmQ1ZTk4YjVmZSIs
InNlcW5vIjo1NjN9LCJwcmV2IjoiZWUwMDc4NTgyNDZhZGY4ODU1OTc2NmYxNjRkMGIxOTUzMDMy
MDljYmQ4MmFmYTdmYzZkZWQxOGI0OWI3YjZiMiIsInNlcW5vIjo0LCJ0YWciOiJzaWduYXR1cmUi
faNzaWfEQG3uIt5g6X6NRAjnHdF1NSRO5UYJD1B0Ku1ixBIeS2zuSAGR0pts2Lbl+Cz3BGvu9isq
7MHrgCa2r1PEo4C/4ACoc2lnX3R5cGUgo3RhZ80CAqd2ZXJzaW9uAQ==
`, TagEncryption, &info)
	require.Error(t, err, "Malformed msgpack should fail to decode, but decoded to: %#v", info)
}

func TestFishyMsgpack(t *testing.T) {
	var info NaclSigInfo
	// This message has a duplicate key ("detached") in the top-level map
	err := DecodeArmoredPacketBody(`
hKRib2R5hqhkZXRhY2hlZMOoZGV0YWNoZWTCqWhhc2hfdHlwZQqja2V5xCMBIHPpctUn+7QopWm+
n1CVw28iikWy6ybCMUUdVRijfKQjCqdwYXlsb2FkxQPteyJib2R5Ijp7ImRldmljZSI6eyJpZCI6
IjA1Nzg0M2MyMDI1MTkyNmFjYzBkNWRiMzEyNjk3OTE4Iiwia2lkIjoiMDEyMTM3OWQ1MzcwYWVi
NGU5ZjdhYTRiNTI3MGY1ZTQ4ODA4NWE2NDdlMGRiMjQ3N2VkZDNmM2M1MGUzZDE0ZjQwNGYwYSIs
InN0YXR1cyI6MX0sImtleSI6eyJlbGRlc3Rfa2lkIjoiMDEyMGM3YTk0YzQ0Mjc2MDk1ZTczNjVm
NDliNjdhZTY4ZjdiY2JjODA1YzU4Mjc5N2Y2OTBiNjdiMDhmMDVlZGZlMWIwYSIsImhvc3QiOiJr
ZXliYXNlLmlvIiwia2lkIjoiMDEyMDczZTk3MmQ1MjdmYmI0MjhhNTY5YmU5ZjUwOTVjMzZmMjI4
YTQ1YjJlYjI2YzIzMTQ1MWQ1NTE4YTM3Y2E0MjMwYSIsInVpZCI6ImEwOTE1NTIxMDUzMmQ5NjA4
MWM0YTNhOTljNjQ4NDE5IiwidXNlcm5hbWUiOiJsd3BwXzQyMmFiNzk4NDcifSwic3Via2V5Ijp7
ImtpZCI6IjAxMjEzNzlkNTM3MGFlYjRlOWY3YWE0YjUyNzBmNWU0ODgwODVhNjQ3ZTBkYjI0Nzdl
ZGQzZjNjNTBlM2QxNGY0MDRmMGEiLCJwYXJlbnRfa2lkIjoiMDEyMDczZTk3MmQ1MjdmYmI0Mjhh
NTY5YmU5ZjUwOTVjMzZmMjI4YTQ1YjJlYjI2YzIzMTQ1MWQ1NTE4YTM3Y2E0MjMwYSJ9LCJ0eXBl
Ijoic3Via2V5IiwidmVyc2lvbiI6MX0sImNsaWVudCI6eyJuYW1lIjoia2V5YmFzZS5pbyBnbyBj
bGllbnQiLCJ2ZXJzaW9uIjoiMS4wLjAifSwiY3RpbWUiOjE0NDM0NjU5NDMsImV4cGlyZV9pbiI6
NTA0NTc2MDAwLCJtZXJrbGVfcm9vdCI6eyJjdGltZSI6MTQ0MzQ2NTk0MywiaGFzaCI6IjhlYjMz
YjA2YzFlMDIyMWFjYmFjZTMwNmQyM2VhMWQ5MjJiYWFhY2M2YWJiZDI3YzM5Y2Y2M2JjYzI2NzM0
ZWY0ODUwY2Y4NzZhZmU2OTE1Nzg0OTg0MTNlMmU5NzQzYjc5Yjk3YmUxNjFlYzA4ZGI0Y2YyOTZi
ZDVlOThiNWZlIiwic2Vxbm8iOjU2M30sInByZXYiOiJlZTAwNzg1ODI0NmFkZjg4NTU5NzY2ZjE2
NGQwYjE5NTMwMzIwOWNiZDgyYWZhN2ZjNmRlZDE4YjQ5YjdiNmIyIiwic2Vxbm8iOjQsInRhZyI6
InNpZ25hdHVyZSJ9o3NpZ8RAbe4i3mDpfo1ECOcd0XU1JE7lRgkPUHQq7WLEEh5LbO5IAZHSm2zY
tuX4LPcEa+72KyrsweuAJravU8SjgL/gAKhzaWdfdHlwZSCjdGFnzQICp3ZlcnNpb24B
`, TagSignature, &info)
	require.IsType(t, err, FishyMsgpackError{}, "info=%+v, err+%+v", info, err)
}

// Guard against unexpected codec encoding changes, in particular for
// ints.
func TestHardcodedPacketEncode(t *testing.T) {
	p, err := NewKeybasePacket(nil, TagSignature, KeybasePacketV1)
	require.NoError(t, err)

	p.Hash = nil

	bytes, err := p.Encode()
	require.NoError(t, err)
	// In particular, {0xcd, 0x2, 0x2} shouldn't change to
	// {0xd1, 0x2, 0x2}.
	expectedBytes := []byte{0x83, 0xa4, 0x62, 0x6f, 0x64, 0x79, 0xc0, 0xa3, 0x74, 0x61, 0x67, 0xcd, 0x2, 0x2, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x1}
	require.Equal(t, expectedBytes, bytes)
}

// This is a regression test for
// https://github.com/ugorji/go/issues/237 .
func TestMsgpackReencodeNilHash(t *testing.T) {
	var info NaclEncryptionInfo
	// This message has a nil hash.
	err := DecodeArmoredPacketBody(`
hKRib2R5hapjaXBoZXJ0ZXh0wKhlbmNfdHlwZQClbm9uY2XArHJlY2VpdmVyX2tlecCqc2VuZGVy
X2tlecCkaGFzaIKkdHlwZQildmFsdWXEIJZSZH19AzYud7qy9x3yx1hN2MooqnhjsytUSqTK+VMZ
o3RhZ80CA6d2ZXJzaW9uAQ==
`, TagEncryption, &info)
	// In particular, shouldn't return a FishyMsgpackError.
	require.NoError(t, err, "info=%+v", info)
}
