package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	registry "github.com/ethereum-optimism/superchain-registry/superchain"
	"github.com/ethereum-optimism/supersim/genesis"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// Varsayılan mnemonic ve hesap sayısı
	DefaultSecretsConfig = SecretsConfig{
		Accounts:       10,
		Mnemonic:       "test test test test test test test test test test test junk",
		DerivationPath: accounts.DefaultRootDerivationPath,
	}
)

// Fork yapılandırması
type ForkConfig struct {
	RPCUrl      string
	BlockNumber uint64
}

// Secrets yapılandırması
type SecretsConfig struct {
	Accounts       uint64
	Mnemonic       string
	DerivationPath accounts.DerivationPath
}

// L2 zincir yapılandırması
type L2Config struct {
	L1ChainID     uint64
	L1Addresses   *registry.AddressList
	DependencySet []uint64
}

// Zincir yapılandırması
type ChainConfig struct {
	Name              string
	Port              uint64
	ChainID           uint64
	GenesisJSON       []byte
	SecretsConfig     SecretsConfig
	ForkConfig        *ForkConfig
	L2Config          *L2Config
	StartingTimestamp uint64
	LogsDirectory     string
}

// Ağ yapılandırması
type NetworkConfig struct {
	L1Config          ChainConfig
	L2StartingPort    uint64
	L2Configs         []ChainConfig
	InteropEnabled    bool
	InteropAutoRelay  bool
	InteropDelay      uint64
}

// Zincir arayüzü
type Chain interface {
	Endpoint() string
	LogPath() string
	Config() *ChainConfig
	EthClient() *ethclient.Client
	SimulatedLogs(ctx context.Context, tx *types.Transaction) ([]types.Log, error)
	SetCode(ctx context.Context, result interface{}, address common.Address, code string) error
	SetStorageAt(ctx context.Context, result interface{}, address common.Address, storageSlot string, storageValue string) error
	SetBalance(ctx context.Context, result interface{}, address common.Address, value *big.Int) error
	SetIntervalMining(ctx context.Context, result interface{}, interval int64) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Varsayılan ağ yapılandırmasını döndürür
func GetDefaultNetworkConfig(startingTimestamp uint64, logsDirectory string) (NetworkConfig, error) {
	if startingTimestamp == 0 || logsDirectory == "" {
		return NetworkConfig{}, errors.New("Geçersiz başlangıç zaman damgası veya log dizini")
	}

	return NetworkConfig{
		InteropEnabled: true,
		L1Config: ChainConfig{
			Name:              "Local",
			ChainID:           genesis.GeneratedGenesisDeployment.L1.ChainID,
			SecretsConfig:     DefaultSecretsConfig,
			GenesisJSON:       genesis.GeneratedGenesisDeployment.L1.GenesisJSON,
			StartingTimestamp: startingTimestamp,
			LogsDirectory:     logsDirectory,
		},
		L2Configs: []ChainConfig{
			{
				Name:          "OPChainA",
				ChainID:       genesis.GeneratedGenesisDeployment.L2s[0].ChainID,
				SecretsConfig: DefaultSecretsConfig,
				GenesisJSON:   genesis.GeneratedGenesisDeployment.L2s[0].GenesisJSON,
				L2Config: &L2Config{
					L1ChainID:     genesis.GeneratedGenesisDeployment.L1.ChainID,
					L1Addresses:   genesis.GeneratedGenesisDeployment.L2s[0].RegistryAddressList(),
					DependencySet: []uint64{genesis.GeneratedGenesisDeployment.L2s[1].ChainID},
				},
				StartingTimestamp: startingTimestamp,
				LogsDirectory:     logsDirectory,
			},
			{
				Name:          "OPChainB",
				ChainID:       genesis.GeneratedGenesisDeployment.L2s[1].ChainID,
				SecretsConfig: DefaultSecretsConfig,
				GenesisJSON:   genesis.GeneratedGenesisDeployment.L2s[1].GenesisJSON,
				L2Config: &L2Config{
					L1ChainID:     genesis.GeneratedGenesisDeployment.L1.ChainID,
					L1Addresses:   genesis.GeneratedGenesisDeployment.L2s[1].RegistryAddressList(),
					DependencySet: []uint64{genesis.GeneratedGenesisDeployment.L2s[0].ChainID},
				},
				StartingTimestamp: startingTimestamp,
				LogsDirectory:     logsDirectory,
			},
		},
	}, nil
}

// Varsayılan SecretsConfig'ı JSON olarak döndürür
func DefaultSecretsConfigAsJSON() (string, error) {
	keys, err := devkeys.NewMnemonicDevKeys(DefaultSecretsConfig.Mnemonic)
	if err != nil {
		return "", fmt.Errorf("Mnemonic anahtarlar oluşturulamadı: %w", err)
	}

	var accounts []map[string]string
	for i := uint64(0); i < DefaultSecretsConfig.Accounts; i++ {
		address, _ := keys.Address(devkeys.UserKey(i))
		privateKey, _ := keys.Secret(devkeys.UserKey(i))
		accounts = append(accounts, map[string]string{
			"Address":    address.Hex(),
			"PrivateKey": hexutil.Encode(crypto.FromECDSA(privateKey)),
		})
	}

	jsonData, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON oluşturulamadı: %w", err)
	}

	return string(jsonData), nil
}

// SecretsConfig'ı JSON dosyasından yükler
func LoadSecretsConfigFromJSON(filePath string) (SecretsConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return SecretsConfig{}, fmt.Errorf("JSON dosyası açılamadı: %w", err)
	}
	defer file.Close()

	var config SecretsConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return SecretsConfig{}, fmt.Errorf("JSON parse edilemedi: %w", err)
	}

	return config, nil
}
