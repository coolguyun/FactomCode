// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	// "github.com/davecgh/go-spew/spew"
	"github.com/FactomProject/FactomCode/common"
)

// Ack Type
const (
	AckFactoidTx uint8 = iota
	EndMinute1
	EndMinute2
	EndMinute3
	EndMinute4
	EndMinute5
	EndMinute6
	EndMinute7
	EndMinute8
	EndMinute9
	EndMinute10
	AckRevealEntry
	AckCommitChain
	AckRevealChain
	AckCommitEntry

	EndMinute
	NonEndMinute
	Unknown
)

// MsgAck is the message sent out by the leader to the followers for
// message it receives and puts into process list.
type MsgAck struct {
	Height            uint32
	ChainID           *common.Hash
	Index             uint32
	Type              byte
	DBlockTimestamp   uint32   // timestamp from leader used for DBlock.Timestamp for followers
	CoinbaseTimestamp uint64   // timestamp from leader used for FBlock coinbase.MilliTimestamp
	Affirmation       *ShaHash // affirmation value -- hash of the message/object in question
	SerialHash        [32]byte
	Signature         [64]byte
	SourceNodeID	  string
	SourceAddr		  string // the ip address of source peer in case of non-mesh network
}

// Sign is used to sign this message
func (msg *MsgAck) Sign(priv *common.PrivateKey) error {
	bytes, err := msg.GetBinaryForSignature()
	if err != nil {
		return err
	}
	msg.Signature = *priv.Sign(bytes).Sig
	return nil
}

//func (msg *MsgAck) Verify()

// GetBinaryForSignature Writes out the MsgAck (excluding Signature) to binary.
func (msg *MsgAck) GetBinaryForSignature() (data []byte, err error) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg.Height)
	if msg.ChainID != nil {
		data, err = msg.ChainID.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	binary.Write(&buf, binary.BigEndian, msg.Index)
	buf.WriteByte(msg.Type)
	binary.Write(&buf, binary.BigEndian, msg.DBlockTimestamp)
	binary.Write(&buf, binary.BigEndian, msg.CoinbaseTimestamp)
	buf.Write(msg.Affirmation.Bytes())
	buf.Write(msg.SerialHash[:])
	buf.WriteByte(byte(len(msg.SourceNodeID)))
	buf.Write([]byte(msg.SourceNodeID))
	buf.WriteByte(byte(len(msg.SourceAddr)))
	buf.Write([]byte(msg.SourceAddr))
	return buf.Bytes(), err
}

// MsgDecode is part of the Message interface implementation.
func (msg *MsgAck) MsgDecode(r io.Reader, pver uint32) error {
	newData, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("MsgAck.MsgDecode reader is invalid")
	}

	msg.Height, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]
	msg.ChainID = common.NewHash()
	newData, _ = msg.ChainID.UnmarshalBinaryData(newData)

	msg.Index, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]
	msg.Type, newData = newData[0], newData[1:]
	msg.DBlockTimestamp, newData = binary.BigEndian.Uint32(newData[0:4]), newData[4:]
	msg.CoinbaseTimestamp, newData = binary.BigEndian.Uint64(newData[0:8]), newData[8:]
	msg.Affirmation, _ = NewShaHash(newData[:32])

	newData = newData[32:]
	copy(msg.SerialHash[:], newData[0:32])
	newData = newData[32:]
	copy(msg.Signature[:], newData[0:64])

	var slen byte
	var s []byte
	slen, newData = newData[64], newData[65:]
	s, newData = newData[:slen], newData[slen:]
	msg.SourceNodeID = string(s)

	slen, newData = newData[0], newData[1:]
	msg.SourceAddr = string(newData[:slen])
	return nil
}

// MsgEncode is part of the Message interface implementation.
func (msg *MsgAck) MsgEncode(w io.Writer, pver uint32) error {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msg.Height)
	buf.Write(msg.ChainID.Bytes())
	binary.Write(&buf, binary.BigEndian, msg.Index)
	buf.WriteByte(msg.Type)
	binary.Write(&buf, binary.BigEndian, msg.DBlockTimestamp)
	binary.Write(&buf, binary.BigEndian, msg.CoinbaseTimestamp)
	buf.Write(msg.Affirmation.Bytes())
	buf.Write(msg.SerialHash[:])
	buf.Write(msg.Signature[:])
	buf.WriteByte(byte(len(msg.SourceNodeID)))
	buf.Write([]byte(msg.SourceNodeID))
	buf.WriteByte(byte(len(msg.SourceAddr)))
	buf.Write([]byte(msg.SourceAddr))
	w.Write(buf.Bytes())
	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgAck) Command() string {
	return CmdAck
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgAck) MaxPayloadLength(pver uint32) uint32 {
	return 300 //4 + 32 + 4 + 1 + 4 + 8 + 32 + 32 + 64 = 181 + len(str)
}

// NewMsgAck returns a new ack message that conforms to the Message
// interface.  See MsgAck for details.
func NewMsgAck(height uint32, index uint32, affirm *ShaHash, ackType byte, timestamp uint32, 
	coinbaseTS uint64, sid string, addr string) *MsgAck {

	if affirm == nil {
		affirm = new(ShaHash)
	}
	return &MsgAck{
		Height:            height,
		ChainID:           common.NewHash(), //TODO: get the correct chain id from processor
		Index:             index,
		DBlockTimestamp:   timestamp,
		CoinbaseTimestamp: coinbaseTS,
		Affirmation:       affirm,
		Type:              ackType,
		SourceNodeID:	   sid,
		SourceAddr:		   addr,
	}
}

// Sha Creates a sha hash from the message binary (output of MsgEncode)
func (msg *MsgAck) Sha() (ShaHash, error) {
	buf := bytes.NewBuffer(nil)
	msg.MsgEncode(buf, ProtocolVersion)
	var sha ShaHash
	_ = sha.SetBytes(Sha256(buf.Bytes()))
	return sha, nil
}

// Clone creates a new MsgAck with the same value
func (msg *MsgAck) Clone() *MsgAck {
	return &MsgAck{
		Height:            msg.Height,
		ChainID:           msg.ChainID,
		Index:             msg.Index,
		DBlockTimestamp:   msg.DBlockTimestamp,
		CoinbaseTimestamp: msg.CoinbaseTimestamp,
		Affirmation:       msg.Affirmation,
		Type:              msg.Type,
		SourceNodeID:	   msg.SourceNodeID,
		SourceAddr:		   msg.SourceAddr,
	}
}

// IsEomAck checks if it's a EOM ack
func (msg *MsgAck) IsEomAck() bool {
	if EndMinute1 <= msg.Type && msg.Type <= EndMinute10 {
		return true
	}
	return false
}

// Equals check if two MsgAcks are the same
func (msg *MsgAck) Equals(ack *MsgAck) bool {
	return msg.Height == ack.Height &&
		msg.Index == ack.Index &&
		msg.Type == ack.Type &&
		msg.DBlockTimestamp == ack.DBlockTimestamp &&
		msg.CoinbaseTimestamp == ack.CoinbaseTimestamp &&
		msg.Affirmation.IsEqual(ack.Affirmation) &&
		msg.ChainID.IsSameAs(ack.ChainID) &&
		bytes.Equal(msg.SerialHash[:], ack.SerialHash[:]) &&
		bytes.Equal(msg.Signature[:], ack.Signature[:]) && 
		msg.SourceNodeID == ack.SourceNodeID &&
		msg.SourceAddr == ack.SourceAddr
}

// String returns its string value
func (msg *MsgAck) String() string {
	return fmt.Sprintf("Ack(h=%d, idx=%d, type=%v, from=%s [%s])", 
		msg.Height, msg.Index, msg.Type, msg.SourceNodeID, msg.SourceAddr)
}
