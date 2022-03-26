// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

// (Re-)generated by schema tool
// >>>> DO NOT CHANGE THIS FILE! <<<<
// Change the json schema instead

#![allow(dead_code)]
#![allow(unused_imports)]

use wasmlib::*;

use crate::*;

#[derive(Clone)]
pub struct MapAgentIDToImmutableAllowancesForAgent {
	pub(crate) proxy: Proxy,
}

impl MapAgentIDToImmutableAllowancesForAgent {
    pub fn get_allowances_for_agent(&self, key: &ScAgentID) -> ImmutableAllowancesForAgent {
        ImmutableAllowancesForAgent { proxy: self.proxy.key(&agent_id_to_bytes(key)) }
    }
}

#[derive(Clone)]
pub struct ImmutableErc20State {
	pub(crate) proxy: Proxy,
}

impl ImmutableErc20State {
    pub fn all_allowances(&self) -> MapAgentIDToImmutableAllowancesForAgent {
		MapAgentIDToImmutableAllowancesForAgent { proxy: self.proxy.root(STATE_ALL_ALLOWANCES) }
	}

    pub fn balances(&self) -> MapAgentIDToImmutableUint64 {
		MapAgentIDToImmutableUint64 { proxy: self.proxy.root(STATE_BALANCES) }
	}

    pub fn supply(&self) -> ScImmutableUint64 {
		ScImmutableUint64::new(self.proxy.root(STATE_SUPPLY))
	}
}

#[derive(Clone)]
pub struct MapAgentIDToMutableAllowancesForAgent {
	pub(crate) proxy: Proxy,
}

impl MapAgentIDToMutableAllowancesForAgent {
    pub fn clear(&self) {
        self.proxy.clear_map();
    }

    pub fn get_allowances_for_agent(&self, key: &ScAgentID) -> MutableAllowancesForAgent {
        MutableAllowancesForAgent { proxy: self.proxy.key(&agent_id_to_bytes(key)) }
    }
}

#[derive(Clone)]
pub struct MutableErc20State {
	pub(crate) proxy: Proxy,
}

impl MutableErc20State {
    pub fn as_immutable(&self) -> ImmutableErc20State {
		ImmutableErc20State { proxy: self.proxy.root("") }
	}

    pub fn all_allowances(&self) -> MapAgentIDToMutableAllowancesForAgent {
		MapAgentIDToMutableAllowancesForAgent { proxy: self.proxy.root(STATE_ALL_ALLOWANCES) }
	}

    pub fn balances(&self) -> MapAgentIDToMutableUint64 {
		MapAgentIDToMutableUint64 { proxy: self.proxy.root(STATE_BALANCES) }
	}

    pub fn supply(&self) -> ScMutableUint64 {
		ScMutableUint64::new(self.proxy.root(STATE_SUPPLY))
	}
}