// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The tables below map geth's error messages to the exception names that
// execution-spec-tests (EEST) fixtures expect. They are a copy of EEST's
// GethExceptionMapper, the mapping its consume command applies to geth:
// packages/testing/src/execution_testing/client_clis/clis/geth.py, blob
// 5baff1852411516e6354fc97741043191ae61420 (execution-specs forks/amsterdam
// a87891f7e69eab1f903233c61c5514d8c94bd5d1). Keep them in sync with it.

type exceptionSubstring struct {
	name      string
	substring string
}

type exceptionRegexp struct {
	name    string
	pattern *regexp.Regexp
}

var exceptionSubstrings = []exceptionSubstring{
	{"TransactionException.SENDER_NOT_EOA", "sender not an eoa"},
	{"TransactionException.GAS_ALLOWANCE_EXCEEDED", "gas limit reached"},
	{"TransactionException.INSUFFICIENT_ACCOUNT_FUNDS", "insufficient funds for gas * price + value"},
	{"TransactionException.INTRINSIC_GAS_TOO_LOW", "intrinsic gas too low"},
	{"TransactionException.INTRINSIC_GAS_BELOW_FLOOR_GAS_COST", "insufficient gas for floor data gas cost"},
	{"TransactionException.NONCE_IS_MAX", "nonce has max value"},
	{"TransactionException.TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED", "would exceed maximum allowance"},
	{"TransactionException.INSUFFICIENT_MAX_FEE_PER_BLOB_GAS", "max fee per blob gas less than block blob gas fee"},
	{"TransactionException.INSUFFICIENT_MAX_FEE_PER_GAS", "max fee per gas less than block base fee"},
	{"TransactionException.PRIORITY_GREATER_THAN_MAX_FEE_PER_GAS", "max priority fee per gas higher than max fee per gas"},
	{"TransactionException.INVALID_CHAINID", "invalid chain id for signer"},
	{"TransactionException.INVALID_SIGNATURE_VRS", "invalid transaction v, r, s values"},
	{"TransactionException.TYPE_1_TX_PRE_FORK", "transaction type not supported"},
	{"TransactionException.TYPE_2_TX_PRE_FORK", "transaction type not supported"},
	{"TransactionException.TYPE_3_TX_PRE_FORK", "transaction type not supported"},
	{"TransactionException.TYPE_3_TX_INVALID_BLOB_VERSIONED_HASH", "has invalid hash version"},
	{"TransactionException.TYPE_3_TX_BLOB_COUNT_EXCEEDED", "blob transaction has too many blobs"},
	{"TransactionException.TYPE_3_TX_ZERO_BLOBS", "blob transaction missing blob hashes"},
	{"TransactionException.TYPE_3_TX_WITH_FULL_BLOBS", "unexpected blob sidecar in transaction at index"},
	{"TransactionException.TYPE_3_TX_CONTRACT_CREATION", "input string too short for common.Address, decoding into (types.BlobTx).To"},
	{"TransactionException.TYPE_4_EMPTY_AUTHORIZATION_LIST", "EIP-7702 transaction with empty auth list"},
	{"TransactionException.TYPE_4_TX_CONTRACT_CREATION", "input string too short for common.Address, decoding into (types.SetCodeTx).To"},
	{"TransactionException.GAS_LIMIT_EXCEEDS_MAXIMUM", "transaction gas limit too high"},
	{"TransactionException.TYPE_4_TX_PRE_FORK", "transaction type not supported"},
	{"TransactionException.INITCODE_SIZE_EXCEEDED", "max initcode size exceeded"},
	{"TransactionException.NONCE_MISMATCH_TOO_LOW", "nonce too low"},
	{"TransactionException.NONCE_MISMATCH_TOO_HIGH", "nonce too high"},
	{"BlockException.INCORRECT_BLOB_GAS_USED", "blob gas used mismatch"},
	{"BlockException.INCORRECT_EXCESS_BLOB_GAS", "invalid excessBlobGas"},
	{"BlockException.INVALID_VERSIONED_HASHES", "invalid number of versionedHashes"},
	{"BlockException.INVALID_REQUESTS", "invalid requests hash"},
	{"BlockException.SYSTEM_CONTRACT_EMPTY", "empty system contract"},
	{"BlockException.SYSTEM_CONTRACT_CALL_FAILED", "system call failed to execute:"},
	{"BlockException.INVALID_BLOCK_HASH", "blockhash mismatch"},
	{"BlockException.RLP_BLOCK_LIMIT_EXCEEDED", "block RLP-encoded size exceeds maximum"},
	{"BlockException.INVALID_BLOCK_ACCESS_LIST", "unequal"},
	{"BlockException.INVALID_BASEFEE_PER_GAS", "invalid baseFee"},
	{"BlockException.INVALID_BLOCK_TIMESTAMP_OLDER_THAN_PARENT", "invalid timestamp"},
	{"BlockException.INVALID_GASLIMIT", "invalid gas limit"},
	{"BlockException.INVALID_BLOCK_NUMBER", "invalid block number"},
	{"BlockException.EXTRA_DATA_TOO_BIG", "invalid extradata length"},
	{"BlockException.INVALID_RECEIPTS_ROOT", "invalid receipt root hash"},
	{"BlockException.INVALID_LOG_BLOOM", "invalid bloom"},
	{"BlockException.INVALID_STATE_ROOT", "invalid merkle root"},
	{"BlockException.GAS_USED_OVERFLOW", "bal validation failure"},
}

var exceptionRegexps = []exceptionRegexp{
	{"TransactionException.INVALID_SIGNATURE_VRS", regexp.MustCompile(`recovery failed`)},
	{"TransactionException.TYPE_3_TX_MAX_BLOB_GAS_ALLOWANCE_EXCEEDED", regexp.MustCompile(`blob gas used \d+ exceeds maximum allowance \d+`)},
	{"BlockException.BLOB_GAS_USED_ABOVE_LIMIT", regexp.MustCompile(`blob gas used \d+ exceeds maximum allowance \d+`)},
	{"BlockException.INVALID_GAS_USED_ABOVE_LIMIT", regexp.MustCompile(`invalid gasUsed: have \d+, gasLimit \d+`)},
	{"BlockException.INVALID_GAS_USED", regexp.MustCompile(`invalid gas used \(remote: \d+ local: \d+\)`)},
	{"BlockException.INVALID_DEPOSIT_EVENT_LAYOUT", regexp.MustCompile(`invalid requests hash|failed to parse deposit logs`)},
	{"BlockException.INVALID_BAL_HASH", regexp.MustCompile(`invalid block access list:|access list hash mismatch`)},
	{"BlockException.INVALID_BLOCK_ACCESS_LIST", regexp.MustCompile(`difference between computed state diff and BAL entry for account|invalid block access list:|computed state diff contained mutated accounts which weren't reported in BAL|BAL change not reported in computed|additional mutations compared to BAL|access list hash mismatch|failed to decode BAL|[bB][aA][lL] validation fail`)},
	{"BlockException.INCORRECT_BLOCK_FORMAT", regexp.MustCompile(`invalid block access list:`)},
	{"BlockException.BLOCK_ACCESS_LIST_GAS_LIMIT_EXCEEDED", regexp.MustCompile(`block access list exceeds gas limit|block access list exceeds size constraint`)},
	{"BlockException.GAS_USED_OVERFLOW", regexp.MustCompile(`gas limit reached`)},
	{"TransactionException.INTRINSIC_GAS_TOO_LOW", regexp.MustCompile(`insufficient gas for floor data gas cost`)},
}

// exceptionNames returns the names of the exceptions a geth error message maps
// to, matching EEST's mapper: every substring it contains and every pattern it
// matches.
func exceptionNames(msg string) []string {
	var names []string
	for _, e := range exceptionSubstrings {
		if strings.Contains(msg, e.substring) {
			names = append(names, e.name)
		}
	}
	for _, e := range exceptionRegexps {
		if e.pattern.MatchString(msg) {
			names = append(names, e.name)
		}
	}
	return names
}

// checkException reports whether a block or payload rejected with the error
// msg was rejected for the reason its fixture expects. The expectation may list
// several names separated by "|", any of which matches. An error that maps to
// no name fails too, so the tables stay complete. Expectations that are not
// EEST exception names, such as those of the legacy ethereum/tests fixtures,
// are not checked.
func checkException(expected, msg string) error {
	want := strings.Split(expected, "|")
	if !slices.ContainsFunc(want, isEESTException) {
		return nil
	}
	got := exceptionNames(msg)
	if len(got) == 0 {
		return fmt.Errorf("rejected with an error that maps to no exception: expected %s, got %q", expected, msg)
	}
	for _, name := range got {
		if slices.Contains(want, name) {
			return nil
		}
	}
	return fmt.Errorf("rejected for the wrong reason: expected %s, got %s (%q)", expected, strings.Join(got, "|"), msg)
}

func isEESTException(name string) bool {
	return strings.HasPrefix(name, "BlockException.") || strings.HasPrefix(name, "TransactionException.")
}
