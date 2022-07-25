package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/forta-network/forta-core-go/utils"
	"github.com/hashicorp/go-multierror"
)

const (
	// IndicatorProxyAPIAccessible can connect to node
	IndicatorProxyAPIAccessible = "proxy.accessible"
	// IndicatorProxyAPIChainID which chain id the json-rpc provides
	IndicatorProxyAPIChainID = "proxy.chain-id"
	// IndicatorProxyAPIModuleWeb3 node supports web3 module.
	IndicatorProxyAPIModuleWeb3 = "proxy.module.web3"
	// IndicatorProxyAPIModuleEth node supports eth module.
	IndicatorProxyAPIModuleEth = "proxy.module.eth"
	// IndicatorProxyAPIModuleNet node supports net module.
	IndicatorProxyAPIModuleNet = "proxy.module.net"
	// IndicatorProxyAPIHistorySupport the earliest supported block height. The lower is better.
	IndicatorProxyAPIHistorySupport = "proxy.history-support"

	// MetadataProxyAPIBlockByNumberHash is the hash of the block data retrieved from the scan API.
	MetadataProxyAPIBlockByNumberHash = "proxy.block-by-number.hash"
)

const (
	// VeryOldBlockNumber is the number of a block which inspection logic considers
	// as a very old block.
	VeryOldBlockNumber = 5
)

// ProxyAPIInspector is an inspector implementation.
type ProxyAPIInspector struct{}

// compile time check: it should implement the interface
var _ Inspector = &ProxyAPIInspector{}

// Name returns the name of the inspector.
func (sai *ProxyAPIInspector) Name() string {
	return "proxy-api"
}

// Inspect checks given JSON-RPC node url supports web3, eth and net modules.
//
// it doesn't actually return any errors for now,
// because the point is to keep going and check if it supports the rest
// error return parameter is simply for keeping the function extensible without api changes in the future.
func (sai *ProxyAPIInspector) Inspect(ctx context.Context, inspectionCfg InspectionConfig) (results *InspectionResults, resultErr error) {
	results = NewInspectionResults()

	rpcClient, err := rpc.DialContext(ctx, inspectionCfg.ProxyAPIURL)
	if err != nil {
		resultErr = multierror.Append(resultErr, fmt.Errorf("can't dial json-rpc api %w", err))

		results.Indicators[IndicatorProxyAPIAccessible] = ResultFailure
		results.Indicators[IndicatorProxyAPIModuleWeb3] = ResultFailure
		results.Indicators[IndicatorProxyAPIModuleEth] = ResultFailure
		results.Indicators[IndicatorProxyAPIModuleNet] = ResultFailure
		results.Indicators[IndicatorProxyAPIHistorySupport] = ResultFailure
		results.Indicators[IndicatorProxyAPIChainID] = ResultFailure

		return
	}

	client := ethclient.NewClient(rpcClient)

	// arbitrary call to check node access
	if id, err := client.ChainID(ctx); err != nil {
		results.Indicators[IndicatorProxyAPIAccessible] = ResultFailure
		results.Indicators[IndicatorProxyAPIChainID] = ResultFailure
	} else {
		results.Indicators[IndicatorProxyAPIAccessible] = ResultSuccess
		results.Indicators[IndicatorProxyAPIChainID] = float64(id.Uint64())
	}

	currentHeight, err := client.BlockNumber(ctx)
	if err != nil {
		results.Indicators[IndicatorProxyAPIAccessible] = ResultFailure
		results.Indicators[IndicatorProxyAPIHistorySupport] = ResultFailure
		resultErr = multierror.Append(resultErr, err)
	} else {
		// check history support
		results.Indicators[IndicatorProxyAPIAccessible] = ResultSuccess
		results.Indicators[IndicatorProxyAPIHistorySupport] = checkHistorySupport(ctx, currentHeight, client)
	}

	err = checkSupportedModules(ctx, rpcClient, results)
	if err != nil {
		resultErr = multierror.Append(resultErr, fmt.Errorf("error checking module functionality %w", err))
	}

	// get configured block and include hash of the returned as metadata
	hash, err := getBlockResponseHash(ctx, rpcClient, inspectionCfg.BlockNumber)
	if err != nil {
		resultErr = multierror.Append(resultErr, fmt.Errorf("failed to get configured block %d: %v", inspectionCfg.BlockNumber, err))
	} else {
		results.Metadata[MetadataProxyAPIBlockByNumberHash] = hash
	}

	return
}

// checkSupportedModules double-checks the functionality of modules that were declared as supported by
// the node.
func checkSupportedModules(
	ctx context.Context, rpcClient *rpc.Client, results *InspectionResults,
) (resultError error) {
	client := ethclient.NewClient(rpcClient)

	// sends net_version under the hood. should prove the node supports net module
	_, err := client.NetworkID(ctx)
	if err != nil {
		results.Indicators[IndicatorProxyAPIModuleNet] = ResultFailure
		resultError = multierror.Append(resultError, err)
	} else {
		results.Indicators[IndicatorProxyAPIModuleNet] = ResultSuccess
	}

	// sends eth_chainId under the hood. should prove the node supports eth module
	_, err = client.ChainID(ctx)
	if err != nil {
		results.Indicators[IndicatorProxyAPIModuleEth] = ResultFailure
		resultError = multierror.Append(resultError, err)
	} else {
		results.Indicators[IndicatorProxyAPIModuleEth] = ResultSuccess
	}

	// ask for web3 client version to prove the node supports web3 module
	err = rpcClient.CallContext(ctx, nil, "web3_clientVersion")
	if err != nil {
		resultError = multierror.Append(resultError, err)
		results.Indicators[IndicatorProxyAPIModuleWeb3] = ResultFailure
	} else {
		results.Indicators[IndicatorProxyAPIModuleWeb3] = ResultSuccess
	}

	return resultError
}

// checkHistorySupport inspects block history supports. results earliest provided block
func checkHistorySupport(ctx context.Context, latestBlock uint64, client *ethclient.Client) float64 {
	// check for a very old block
	_, err := client.BlockByNumber(ctx, big.NewInt(VeryOldBlockNumber))
	if err == nil {
		return VeryOldBlockNumber
	}

	// check for earliest block if genesis wasn't available
	return float64(findOldestSupportedBlock(ctx, client, 0, latestBlock))
}

// findOldestSupportedBlock returns the earliest block provided by client
func findOldestSupportedBlock(ctx context.Context, client *ethclient.Client, low, high uint64) uint64 {
	memo := make(map[uint64]bool)

	// terminating condition, results merged
	for low < high {
		mid := (low + high) / 2

		// memoization trick.
		_, ok := memo[mid]
		if ok {
			continue
		}

		block := big.NewInt(int64(mid))

		_, err := client.BlockByNumber(ctx, block)
		isProvided := err == nil

		memo[mid] = isProvided
		// terminating condition, optimum solution
		if isProvided && mid == 0 {
			return 0
		}

		// left side of mid
		if isProvided {
			high = mid - 1
			continue
		}

		// right side of mid
		low = mid + 1
	}

	return low
}

func getBlockResponseHash(ctx context.Context, rpcClient *rpc.Client, blockNumber uint64) (string, error) {
	var blockData json.RawMessage
	err := rpcClient.CallContext(ctx, &blockData, "eth_getBlockByNumber", hexutil.EncodeUint64(blockNumber), true)
	if err != nil {
		return "", err
	}
	return utils.HashNormalizedJSON(blockData), nil
}

func getHost(apiURL string) string {
	if len(apiURL) == 0 {
		return "null"
	}
	u, err := url.Parse(apiURL)
	if err != nil {
		return "invalid"
	}
	return u.Host
}