package inspect

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

var testTraceEnv struct {
	TraceAPI string `envconfig:"trace_api" default:"https://rpcapi-tracing.testnet.fantom.network/"`
}

func init() {
	envconfig.MustProcess("test", &testTraceEnv)
}

func TestTraceAPIInspection(t *testing.T) {
	//TODO: this endpoint times out now, so we can't run this test
	t.SkipNow()
	r := require.New(t)

	recentBlockNumber := testGetRecentBlockNumber(r, testTraceEnv.TraceAPI)

	inspector := &TraceAPIInspector{}
	results, err := inspector.Inspect(
		context.Background(), InspectionConfig{
			TraceAPIURL: testTraceEnv.TraceAPI,
			BlockNumber: recentBlockNumber,
			CheckTrace:  true,
		},
	)
	r.NoError(err)

	r.Equal(
		map[string]float64{
			IndicatorTraceAccessible: ResultSuccess,
			IndicatorTraceSupported:  ResultSuccess,
			IndicatorTraceAPIChainID: 4002,
			IndicatorTraceAPIIsETH2:  ResultFailure,
		}, results.Indicators,
	)

	r.NotEmpty(results.Metadata[MetadataTraceAPIBlockByNumberHash])
	r.NotEmpty(results.Metadata[MetadataTraceAPITraceBlockHash])
}

func testGetRecentBlockNumber(r *require.Assertions, apiURL string) uint64 {
	client, err := ethclient.Dial(apiURL)
	r.NoError(err)
	block, err := client.BlockByNumber(context.Background(), nil)
	r.NoError(err)
	return block.NumberU64() - 100
}
