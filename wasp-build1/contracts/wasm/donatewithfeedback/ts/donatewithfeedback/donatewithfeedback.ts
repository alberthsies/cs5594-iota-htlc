// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

import * as wasmlib from "wasmlib"
import * as wasmtypes from "wasmlib/wasmtypes"
import * as sc from "./index";

export function funcDonate(ctx: wasmlib.ScFuncContext, f: sc.DonateContext): void {
    const amount = ctx.incoming().balance(wasmtypes.IOTA);
    let donation = new sc.Donation();
    donation.amount = amount;
    donation.donator = ctx.caller();
    donation.error = "";
    donation.feedback = f.params.feedback().value();
    donation.timestamp = ctx.timestamp();
    if (donation.amount == 0 || donation.feedback.length == 0) {
        donation.error = "error: empty feedback or donated amount = 0".toString();
        if (donation.amount > 0) {
            ctx.send(donation.donator.address(), wasmlib.ScTransfers.iotas(donation.amount));
            donation.amount = 0;
        }
    }
    let log = f.state.log();
    log.appendDonation().setValue(donation);

    let largestDonation = f.state.maxDonation();
    let totalDonated = f.state.totalDonation();
    if (donation.amount > largestDonation.value()) {
        largestDonation.setValue(donation.amount);
    }
    totalDonated.setValue(totalDonated.value() + donation.amount);
}

export function funcWithdraw(ctx: wasmlib.ScFuncContext, f: sc.WithdrawContext): void {
    let balance = ctx.balances().balance(wasmtypes.IOTA);
    let amount = f.params.amount().value();
    if (amount == 0 || amount > balance) {
        amount = balance;
    }
    if (amount == 0) {
        ctx.log("dwf.withdraw: nothing to withdraw");
        return;
    }

    let scCreator = ctx.contractCreator().address();
    ctx.send(scCreator, wasmlib.ScTransfers.iotas(amount));
}

export function viewDonation(ctx: wasmlib.ScViewContext, f: sc.DonationContext): void {
    let nr = f.params.nr().value();
    let donation = f.state.log().getDonation(nr).value();
    f.results.amount().setValue(donation.amount);
    f.results.donator().setValue(donation.donator);
    f.results.error().setValue(donation.error);
    f.results.feedback().setValue(donation.feedback);
    f.results.timestamp().setValue(donation.timestamp);
}

export function viewDonationInfo(ctx: wasmlib.ScViewContext, f: sc.DonationInfoContext): void {
    f.results.maxDonation().setValue(f.state.maxDonation().value());
    f.results.totalDonation().setValue(f.state.totalDonation().value());
    f.results.count().setValue(f.state.log().length());
}
