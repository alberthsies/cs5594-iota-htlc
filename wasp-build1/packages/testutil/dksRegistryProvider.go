// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"fmt"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"

	"github.com/iotaledger/wasp/packages/tcrypto"
)

// DkgRegistryProvider stands for a mock for dkg.DKShareRegistryProvider.
type DkgRegistryProvider struct {
	DB    map[string][]byte
	Suite tcrypto.Suite
}

// NewDkgRegistryProvider creates new mocked DKG registry provider.
func NewDkgRegistryProvider(suite tcrypto.Suite) *DkgRegistryProvider {
	return &DkgRegistryProvider{
		DB:    map[string][]byte{},
		Suite: suite,
	}
}

// SaveDKShare implements dkg.DKShareRegistryProvider.
func (p *DkgRegistryProvider) SaveDKShare(dkShare *tcrypto.DKShare) error {
	p.DB[dkShare.Address.String()] = dkShare.Bytes()
	return nil
}

// LoadDKShare implements dkg.DKShareRegistryProvider.
func (p *DkgRegistryProvider) LoadDKShare(sharedAddress ledgerstate.Address) (*tcrypto.DKShare, error) {
	dkShareBytes := p.DB[sharedAddress.String()]
	if dkShareBytes == nil {
		return nil, fmt.Errorf("DKShare not found for %v", sharedAddress.Base58())
	}
	return tcrypto.DKShareFromBytes(dkShareBytes, p.Suite)
}
