// Copyright 2023 The AthanorLabs/atomic-swap Authors
// SPDX-License-Identifier: LGPL-3.0-only

package relayer

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/athanorlabs/atomic-swap/common/types"
	contracts "github.com/athanorlabs/atomic-swap/ethereum"
	"github.com/athanorlabs/atomic-swap/tests"
)

// Speed up tests a little by giving deployContracts(...) a package-level cache.
// These variables should not be accessed by other functions.
var _forwarderAddr *ethcommon.Address
var _swapCreatorAddr *ethcommon.Address

// deployContracts deploys and returns the swapCreator and forwarder addresses.
func deployContracts(t *testing.T, ec *ethclient.Client, key *ecdsa.PrivateKey) (ethcommon.Address, ethcommon.Address) {
	ctx := context.Background()

	if _forwarderAddr == nil || _swapCreatorAddr == nil {
		forwarderAddr, err := contracts.DeployGSNForwarderWithKey(ctx, ec, key)
		require.NoError(t, err)
		_forwarderAddr = &forwarderAddr

		swapCreatorAddr, _, err := contracts.DeploySwapCreatorWithKey(ctx, ec, key, forwarderAddr)
		require.NoError(t, err)
		_swapCreatorAddr = &swapCreatorAddr
	}

	return *_swapCreatorAddr, *_forwarderAddr
}

func createTestSwap(claimer ethcommon.Address) *contracts.SwapCreatorSwap {
	return &contracts.SwapCreatorSwap{
		Owner:        ethcommon.Address{0x1},
		Claimer:      claimer,
		PubKeyClaim:  [32]byte{0x1},
		PubKeyRefund: [32]byte{0x1},
		Timeout0:     big.NewInt(time.Now().Add(30 * time.Minute).Unix()),
		Timeout1:     big.NewInt(time.Now().Add(60 * time.Minute).Unix()),
		Asset:        ethcommon.Address(types.EthAssetETH),
		Value:        big.NewInt(1e18),
		Nonce:        big.NewInt(1),
	}
}

func TestCreateRelayClaimRequest(t *testing.T) {
	ctx := context.Background()
	ethKey := tests.GetMakerTestKey(t)
	claimer := crypto.PubkeyToAddress(*ethKey.Public().(*ecdsa.PublicKey))
	ec, _ := tests.NewEthClient(t)
	secret := [32]byte{0x1}
	swapCreatorAddr, forwarderAddr := deployContracts(t, ec, ethKey)

	// success path
	swap := createTestSwap(claimer)
	req, err := CreateRelayClaimRequest(ctx, ethKey, ec, swapCreatorAddr, forwarderAddr, swap, &secret)
	require.NoError(t, err)
	require.NotNil(t, req)

	// change the ethkey to not match the claimer address to trigger the error path
	ethKey = tests.GetTakerTestKey(t)
	_, err = CreateRelayClaimRequest(ctx, ethKey, ec, swapCreatorAddr, forwarderAddr, swap, &secret)
	require.ErrorContains(t, err, "signing key does not match claimer")
}