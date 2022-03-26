// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package governance

import (
	"bytes"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/iotaledger/wasp/packages/iscp"
	"github.com/iotaledger/wasp/packages/kv/codec"
	"github.com/iotaledger/wasp/packages/kv/collections"
	"github.com/iotaledger/wasp/packages/kv/dict"
	"github.com/iotaledger/wasp/packages/kv/kvdecoder"
	"github.com/iotaledger/wasp/packages/util"
	"golang.org/x/xerrors"
)

// NodeOwnershipCertificate is a proof that a specified address is an owner of the specified node.
// It is implemented as a signature over the node pub key concatenated with the owner address.
type NodeOwnershipCertificate []byte

func NewNodeOwnershipCertificate(nodeKeyPair *ed25519.KeyPair, ownerAddress ledgerstate.Address) NodeOwnershipCertificate {
	certData := bytes.Buffer{}
	certData.Write(nodeKeyPair.PublicKey.Bytes())
	certData.Write(ownerAddress.Bytes())
	return nodeKeyPair.PrivateKey.Sign(certData.Bytes()).Bytes()
}

func NewNodeOwnershipCertificateFromBytes(data []byte) NodeOwnershipCertificate {
	return NodeOwnershipCertificate(data)
}

func (c NodeOwnershipCertificate) Verify(nodePubKey ed25519.PublicKey, ownerAddress ledgerstate.Address) bool {
	certData := bytes.Buffer{}
	certData.Write(nodePubKey.Bytes())
	certData.Write(ownerAddress.Bytes())
	signature, _, err := ed25519.SignatureFromBytes(c.Bytes())
	if err != nil {
		return false
	}
	return nodePubKey.VerifySignature(certData.Bytes(), signature)
}

func (c NodeOwnershipCertificate) Bytes() []byte {
	return c
}

// AccessNodeInfo conveys all the information that is maintained
// on the governance SC about a specific node.
type AccessNodeInfo struct {
	NodePubKey    []byte // Public Key of the node. Stored as a key in the SC State and Params.
	ValidatorAddr []byte // Address of the validator owning the node. Not sent via parameters.
	Certificate   []byte // Proof that Validator owns the Node.
	ForCommittee  bool   // true, if Node should be a candidate to a committee.
	AccessAPI     string // API URL, if any.
}

func NewAccessNodeInfoFromBytes(pubKey, value []byte) (*AccessNodeInfo, error) {
	var a AccessNodeInfo
	var err error
	r := bytes.NewReader(value)
	a.NodePubKey = pubKey // NodePubKey stored as a map key.
	if a.ValidatorAddr, err = util.ReadBytes16(r); err != nil {
		return nil, xerrors.Errorf("failed to read AccessNodeInfo.ValidatorAddr: %v", err)
	}
	if a.Certificate, err = util.ReadBytes16(r); err != nil {
		return nil, xerrors.Errorf("failed to read AccessNodeInfo.Certificate: %v", err)
	}
	if err := util.ReadBoolByte(r, &a.ForCommittee); err != nil {
		return nil, xerrors.Errorf("failed to read AccessNodeInfo.ForCommittee: %v", err)
	}
	if a.AccessAPI, err = util.ReadString16(r); err != nil {
		return nil, xerrors.Errorf("failed to read AccessNodeInfo.AccessAPI: %v", err)
	}
	return &a, nil
}

func NewAccessNodeInfoListFromMap(infoMap *collections.ImmutableMap) ([]*AccessNodeInfo, error) {
	res := make([]*AccessNodeInfo, 0)
	var accErr error
	err := infoMap.Iterate(func(elemKey, value []byte) bool {
		var a *AccessNodeInfo
		if a, accErr = NewAccessNodeInfoFromBytes(elemKey, value); accErr != nil {
			return false
		}
		res = append(res, a)
		return true
	})
	if accErr != nil {
		return nil, xerrors.Errorf("failed to iterate over AccessNodeInfo list: %v", accErr)
	}
	if err != nil {
		return nil, xerrors.Errorf("failed to iterate over AccessNodeInfo list: %v", err)
	}
	return res, nil
}

func (a *AccessNodeInfo) Bytes() []byte {
	w := bytes.Buffer{}
	// NodePubKey stored as a map key.
	if err := util.WriteBytes16(&w, a.ValidatorAddr); err != nil {
		panic(xerrors.Errorf("failed to write AccessNodeInfo.ValidatorAddr: %v", err))
	}
	if err := util.WriteBytes16(&w, a.Certificate); err != nil {
		panic(xerrors.Errorf("failed to write AccessNodeInfo.Certificate: %v", err))
	}
	if err := util.WriteBoolByte(&w, a.ForCommittee); err != nil {
		panic(xerrors.Errorf("failed to write AccessNodeInfo.ForCommittee: %v", err))
	}
	if err := util.WriteString16(&w, a.AccessAPI); err != nil {
		panic(xerrors.Errorf("failed to write AccessNodeInfo.AccessAPI: %v", err))
	}
	return w.Bytes()
}

func NewAccessNodeInfoFromAddCandidateNodeParams(ctx iscp.Sandbox) *AccessNodeInfo {
	params := kvdecoder.New(ctx.Params(), ctx.Log())
	ani := AccessNodeInfo{
		NodePubKey:    params.MustGetBytes(ParamAccessNodeInfoPubKey),
		ValidatorAddr: ctx.Request().SenderAddress().Bytes(), // Not from params, to have it validated.
		Certificate:   params.MustGetBytes(ParamAccessNodeInfoCertificate),
		ForCommittee:  params.MustGetBool(ParamAccessNodeInfoForCommittee, false),
		AccessAPI:     params.MustGetString(ParamAccessNodeInfoAccessAPI, ""),
	}
	return &ani
}

func (a *AccessNodeInfo) ToAddCandidateNodeParams() dict.Dict {
	d := dict.New()
	d.Set(ParamAccessNodeInfoForCommittee, codec.EncodeBool(a.ForCommittee))
	d.Set(ParamAccessNodeInfoPubKey, a.NodePubKey)
	d.Set(ParamAccessNodeInfoCertificate, a.Certificate)
	d.Set(ParamAccessNodeInfoAccessAPI, codec.EncodeString(a.AccessAPI))
	return d
}

func NewAccessNodeInfoFromRevokeAccessNodeParams(ctx iscp.Sandbox) *AccessNodeInfo {
	params := kvdecoder.New(ctx.Params(), ctx.Log())
	ani := AccessNodeInfo{
		NodePubKey:    params.MustGetBytes(ParamAccessNodeInfoPubKey),
		ValidatorAddr: ctx.Request().SenderAddress().Bytes(), // Not from params, to have it validated.
		Certificate:   params.MustGetBytes(ParamAccessNodeInfoCertificate),
	}
	return &ani
}

func (a *AccessNodeInfo) ToRevokeAccessNodeParams() dict.Dict {
	d := dict.New()
	d.Set(ParamAccessNodeInfoPubKey, a.NodePubKey)
	d.Set(ParamAccessNodeInfoCertificate, a.Certificate)
	return d
}

func (a *AccessNodeInfo) AddCertificate(nodeKeyPair *ed25519.KeyPair, ownerAddress ledgerstate.Address) *AccessNodeInfo {
	a.Certificate = NewNodeOwnershipCertificate(nodeKeyPair, ownerAddress).Bytes()
	return a
}

func (a *AccessNodeInfo) ValidateCertificate(ctx iscp.Sandbox) bool {
	nodePubKey, _, err := ed25519.PublicKeyFromBytes(a.NodePubKey)
	if err != nil {
		return false
	}
	validatorAddr, _, err := ledgerstate.AddressFromBytes(a.ValidatorAddr)
	if err != nil {
		return false
	}
	cert := NewNodeOwnershipCertificateFromBytes(a.Certificate)
	return cert.Verify(nodePubKey, validatorAddr)
}

//
// GetChainNodesRequest
//
type GetChainNodesRequest struct{}

func (req GetChainNodesRequest) AsDict() dict.Dict {
	return dict.New()
}

//
// GetChainNodesResponse
//
type GetChainNodesResponse struct {
	AccessNodeCandidates []*AccessNodeInfo   // Application info for the AccessNodes.
	AccessNodes          []ed25519.PublicKey // Public Keys of Access Nodes.
}

func NewGetChainNodesResponseFromDict(d dict.Dict) *GetChainNodesResponse {
	res := GetChainNodesResponse{
		AccessNodeCandidates: make([]*AccessNodeInfo, 0),
		AccessNodes:          make([]ed25519.PublicKey, 0),
	}

	ac := collections.NewMapReadOnly(d, ParamGetChainNodesAccessNodeCandidates)
	ac.MustIterate(func(pubKey, value []byte) bool {
		ani, err := NewAccessNodeInfoFromBytes(pubKey, value)
		if err != nil {
			panic(xerrors.Errorf("unable to decode access node info: %v", err))
		}
		res.AccessNodeCandidates = append(res.AccessNodeCandidates, ani)
		return true
	})

	an := collections.NewMapReadOnly(d, ParamGetChainNodesAccessNodes)
	an.MustIterate(func(pubKeyBin, value []byte) bool {
		pubKey, _, err := ed25519.PublicKeyFromBytes(pubKeyBin)
		if err != nil {
			panic(xerrors.Errorf("failed to decode pub key: %v", err))
		}
		res.AccessNodes = append(res.AccessNodes, pubKey)
		return true
	})
	return &res
}

//
//	ChangeAccessNodesRequest
//

type ChangeAccessNodeAction byte

const (
	ChangeAccessNodeActionRemove = ChangeAccessNodeAction(iota)
	ChangeAccessNodeActionAccept
	ChangeAccessNodeActionDrop
)

type ChangeAccessNodesRequest struct {
	actions map[ed25519.PublicKey]ChangeAccessNodeAction
}

func NewChangeAccessNodesRequest() *ChangeAccessNodesRequest {
	return &ChangeAccessNodesRequest{
		actions: make(map[ed25519.PublicKey]ChangeAccessNodeAction),
	}
}

func (req *ChangeAccessNodesRequest) Remove(pubKey ed25519.PublicKey) *ChangeAccessNodesRequest {
	req.actions[pubKey] = ChangeAccessNodeActionRemove
	return req
}

func (req *ChangeAccessNodesRequest) Accept(pubKey ed25519.PublicKey) *ChangeAccessNodesRequest {
	req.actions[pubKey] = ChangeAccessNodeActionAccept
	return req
}

func (req *ChangeAccessNodesRequest) Drop(pubKey ed25519.PublicKey) *ChangeAccessNodesRequest {
	req.actions[pubKey] = ChangeAccessNodeActionDrop
	return req
}

func (req *ChangeAccessNodesRequest) AsDict() dict.Dict {
	d := dict.New()
	actionsMap := collections.NewMap(d, ParamChangeAccessNodesActions)
	for pubKey, action := range req.actions {
		actionsMap.MustSetAt(pubKey.Bytes(), []byte{byte(action)})
	}
	return d
}
