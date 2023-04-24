package config

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/rollkit/rollkit/types"
)

const (
	flagAggregator        = "rollkit.aggregator"
	flagDALayer           = "rollkit.da_layer"
	flagDAConfig          = "rollkit.da_config"
	flagBlockTime         = "rollkit.block_time"
	flagDABlockTime       = "rollkit.da_block_time"
	flagDAStartHeight     = "rollkit.da_start_height"
	flagHeaderNamespaceID = "rollkit.header_namespace_id"
	flagDataNamespaceID   = "rollkit.data_namespace_id"
	flagFraudProofs       = "rollkit.experimental_insecure_fraud_proofs"
	flagLight             = "rollkit.light"
	flagTrustedHash       = "rollkit.trusted_hash"
	flagLazyAggregator    = "rollkit.lazy_aggregator"
)

// NodeConfig stores Rollkit node configuration.
type NodeConfig struct {
	// parameters below are translated from existing config
	RootDir string
	DBPath  string
	P2P     P2PConfig
	RPC     RPCConfig
	// parameters below are Rollkit specific and read from config
	Aggregator         bool `mapstructure:"aggregator"`
	BlockManagerConfig `mapstructure:",squash"`
	DALayer            string `mapstructure:"da_layer"`
	DAConfig           string `mapstructure:"da_config"`
	Light              bool   `mapstructure:"light"`
	HeaderConfig       `mapstructure:",squash"`
	LazyAggregator     bool `mapstructure:"lazy_aggregator"`
}

// HeaderConfig allows node to pass the initial trusted header hash to start the header exchange service
type HeaderConfig struct {
	TrustedHash string `mapstructure:"trusted_hash"`
}

// BlockManagerConfig consists of all parameters required by BlockManagerConfig
type BlockManagerConfig struct {
	// BlockTime defines how often new blocks are produced
	BlockTime time.Duration `mapstructure:"block_time"`
	// DABlockTime informs about block time of underlying data availability layer
	DABlockTime time.Duration `mapstructure:"da_block_time"`
	// DAStartHeight allows skipping first DAStartHeight-1 blocks when querying for blocks.
	DAStartHeight     uint64            `mapstructure:"da_start_height"`
	HeaderNamespaceID types.NamespaceID `mapstructure:"header_namespace_id"`
	DataNamespaceID   types.NamespaceID `mapstructure:"data_namespace_id"`
	FraudProofs       bool              `mapstructure:"fraud_proofs"`
}

// GetViperConfig reads configuration parameters from Viper instance.
//
// This method is called in cosmos-sdk.
func (nc *NodeConfig) GetViperConfig(v *viper.Viper) error {
	nc.Aggregator = v.GetBool(flagAggregator)
	nc.DALayer = v.GetString(flagDALayer)
	nc.DAConfig = v.GetString(flagDAConfig)
	nc.DAStartHeight = v.GetUint64(flagDAStartHeight)
	nc.DABlockTime = v.GetDuration(flagDABlockTime)
	nc.BlockTime = v.GetDuration(flagBlockTime)
	nc.LazyAggregator = v.GetBool(flagLazyAggregator)
	nc.FraudProofs = v.GetBool(flagFraudProofs)
	nc.Light = v.GetBool(flagLight)
	if err := getHexBytes(v, flagHeaderNamespaceID, nc.HeaderNamespaceID[:]); err != nil {
		return err
	}
	if err := getHexBytes(v, flagDataNamespaceID, nc.DataNamespaceID[:]); err != nil {
		return err
	}
	nc.TrustedHash = v.GetString(flagTrustedHash)
	return nil
}

// AddFlags adds Rollkit specific configuration options to cobra Command.
//
// This function is called in cosmos-sdk.
func AddFlags(cmd *cobra.Command) {
	def := DefaultNodeConfig
	cmd.Flags().Bool(flagAggregator, def.Aggregator, "run node in aggregator mode")
	cmd.Flags().Bool(flagLazyAggregator, def.LazyAggregator, "wait for transactions, don't build empty blocks")
	cmd.Flags().String(flagDALayer, def.DALayer, "Data Availability Layer Client name (mock or grpc")
	cmd.Flags().String(flagDAConfig, def.DAConfig, "Data Availability Layer Client config")
	cmd.Flags().Duration(flagBlockTime, def.BlockTime, "block time (for aggregator mode)")
	cmd.Flags().Duration(flagDABlockTime, def.DABlockTime, "DA chain block time (for syncing)")
	cmd.Flags().Uint64(flagDAStartHeight, def.DAStartHeight, "starting DA block height (for syncing)")
	cmd.Flags().BytesHex(flagHeaderNamespaceID, def.HeaderNamespaceID[:], "header namespace identifier (8 bytes in hex)")
	cmd.Flags().BytesHex(flagDataNamespaceID, def.DataNamespaceID[:], "data namespace identifier (8 bytes in hex)")
	cmd.Flags().Bool(flagFraudProofs, def.FraudProofs, "enable fraud proofs (experimental & insecure)")
	cmd.Flags().Bool(flagLight, def.Light, "run light client")
	cmd.Flags().String(flagTrustedHash, def.TrustedHash, "initial trusted hash to start the header exchange service")
}

func getHexBytes(v *viper.Viper, flag string, dst []byte) error {
	str := v.GetString(flag)
	bytes, err := hex.DecodeString(str)
	if err != nil {
		return err
	}
	if len(bytes) != len(types.NamespaceID{}) {
		return fmt.Errorf("invalid length of namespace ID for '%s', expected: %d, got: %d",
			flag, len(types.NamespaceID{}), len(bytes))
	}
	_ = copy(dst, bytes)
	return nil
}
