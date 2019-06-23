// Copyright 2018 The huayulei_2003@hotmail.com Authors
// This file is part of the airfk library.
//
// The airfk library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The airfk library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the airfk library. If not, see <http://www.gnu.org/licenses/>.
package common

import (
	"encoding/binary"
	"strconv"

	"airman.com/airfk/pkg/common/hexutil"
)

// A ItemID is a 64-bit ID.
type ItemID [8]byte

// EncodeItemID converts the given integer to a ID.
func EncodeItemID(i uint64) ItemID {
	var n ItemID
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// EncodeUint64 encodes i as a hex string with 0x prefix.
func EncodeUint64(i uint64) []byte {
	enc := make([]byte, 2, 10)
	copy(enc, "0x")
	return strconv.AppendUint(enc, i, 16)
}

// Uint64 returns the integer value of a ID.
func (n ItemID) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// Int64 returns the integer value of a ID.
func (n ItemID) Int64() int64 {
	return int64(binary.BigEndian.Uint64(n[:]))
}

// Bytes returns bytes of ItemID.
func (n ItemID) Bytes() []byte {
	return n[:]
}

// HexString returns hexadecimal string of ItemID.
func (n ItemID) HexString() string {
	return hexutil.Encode(n[:])
}

// Hex returns hexadecimal string of ItemID.
func (n ItemID) Hex() string {
	return hexutil.EncodeUint64(n.Uint64())
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n ItemID) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n ItemID) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("ItemID", input, n[:])
}
