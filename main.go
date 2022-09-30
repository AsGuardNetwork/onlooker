package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/containrrr/shoutrrr"
	"github.com/hako/durafmt"
	lens "github.com/strangelove-ventures/lens/client"
	registry "github.com/strangelove-ventures/lens/client/chain_registry"
	"gopkg.in/yaml.v2"
)

var wg sync.WaitGroup

type walletInfo struct {
	ChainName     string   `yaml:"chainName"`
	ChainID       string   `yaml:"chainId"`
	WalletAddress string   `yaml:"walletAddress"`
	Notify        []string `yaml:"notify"`
	Amount        int      `yaml:"amount"`
	Duration      string   `yaml:"duration"`
}

type walletsInfo struct {
	walletsInfo []walletInfo
}

func main() {
	filename := ""
	if _, err := os.Stat(strings.Trim(os.Getenv("HOME"), "/") + "/onlooker.yaml"); err == nil {
		filename = strings.Trim(os.Getenv("HOME"), "/") + "/onlooker.yaml"
	} else {
		filename, _ = filepath.Abs("./onlooker.yaml")
		if err != nil {
			fmt.Printf("Failed to get absolute path. Err: %v \n", err)
			panic(err)
		}
	}

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read yaml. Err: %v \n", err)
	}
	walletsInfo := walletsInfo{}
	if yaml.Unmarshal(yamlFile, &walletsInfo.walletsInfo) != nil {
		fmt.Printf("Failed to Unmarshal yaml. Err: %v \n", err)
	}

	wg.Add(len(walletsInfo.walletsInfo))
	for _, wallet := range walletsInfo.walletsInfo {
		fmt.Println(wallet)
		go checkBalance(wallet)
	}
	wg.Wait()
}

func checkBalance(wallet walletInfo) {
	// Get goroutine ID
	routineId := Goid()

	// Fetches chain info from chain registry
	chainInfo, err := registry.DefaultChainRegistry().GetChain(wallet.ChainName)
	if err != nil {
		fmt.Printf("[%d] Failed to get chain info. Err: %v \n", routineId, err)
	}

	// Fetched assetList from chain registry
	assetInfo, err := chainInfo.GetAssetList()
	if err != nil {
		fmt.Printf("[%d] Failed to get assetlist info. Err: %v \n", routineId, err)
	}

	// Get zero Exponent denom i.e. assetmantle -> mantle-1 -> umntl
	chainDenom, err := GetZeroExponentDenom(assetInfo)
	if err != nil {
		fmt.Printf("[%d] Failed get denom. Err: %v", routineId, err)
	}

	// Get healthy RPC
	rpc, err := chainInfo.GetRandomRPCEndpoint()
	if err != nil {
		fmt.Printf("[%d] Failed to get RPC endpoints on chain %s. Err: %v \n", routineId, chainInfo.ChainName, err)
	}

	// For this simple example, only two fields from the config are needed
	chainConfig := lens.ChainClientConfig{
		RPCAddr: rpc,
		// GRPCAddr:       lens.GetTestClient().Config.GRPCAddr,
		KeyringBackend: "test",
		ChainID:        wallet.ChainID,
	}

	// Creates client object to pull chain info
	chainClient, err := lens.NewChainClient(&chainConfig, os.Getenv("HOME"), os.Stdin, os.Stdout)
	if err != nil {
		fmt.Printf("[%d] Failed to build new chain client for %s. Err: %v \n", routineId, chainInfo.ChainID, err)
	}

	// Unmarshal timeforamt
	duration, err := durafmt.ParseString(wallet.Duration)
	if err != nil {
		fmt.Println(err)
	}
	// short wallet address
	// TODO: fix short address length
	shortWalletAddress := fmt.Sprintf("[%d] %s...%s",
		routineId, string(wallet.WalletAddress)[:len(chainInfo.Bech32Prefix)+6], string(wallet.WalletAddress)[len(wallet.WalletAddress)-5:])

	// Checking checkBalance
	for {
		balance, err := chainClient.QueryBalanceWithAddress(wallet.WalletAddress)
		if err != nil {
			fmt.Printf("[%d] Failed to query balance. Err: %v \n", routineId, err)
		}
		// get balancer of proper denome
		walletBalance := balance.AmountOf(chainDenom)
		fmt.Printf("[%d] %s: balance %s %s\n",
			routineId, shortWalletAddress, walletBalance, chainDenom)

		// Balance condition
		if int(walletBalance.Int64()) < wallet.Amount {
			alertBody := fmt.Sprintf(`
⚠️LOW balance⚠️
wallet: %s
amount: %v %s
https://www.mintscan.io/%s/account/%s`,
				wallet.WalletAddress, walletBalance, chainDenom, wallet.ChainName, wallet.WalletAddress)

			// Send notification to each notification url
			// TODO: hide sensitive info
			for _, notify := range wallet.Notify {
				if shoutrrr.Send(notify, alertBody) != nil {
					fmt.Printf("Failed to notify %v. Err: %v \n", notify, err)
				}
			}
		}

		// TODO: env based
		fmt.Printf("[%d] Waiting %v\n", routineId, duration)
		time.Sleep(duration.Duration())
	}
}

func GetZeroExponentDenom(chainAssetList registry.AssetList) (string, error) {
	for _, asset := range chainAssetList.Assets {
		for _, denom := range asset.DenomUnits {
			if denom.Exponent == 0 {
				return denom.Denom, nil
			}
		}
	}
	return "", fmt.Errorf("No Zero Exponent denom found")
}

func Goid() int {
	var buf [64]byte
	id, _ := strconv.Atoi(strings.Fields(strings.TrimPrefix(string(buf[:runtime.Stack(buf[:], false)]), "goroutine "))[0])
	return id
}
