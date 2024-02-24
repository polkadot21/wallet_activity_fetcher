package config

import (
	"github.com/joho/godotenv"
	"github.com/namsral/flag"
)

type Config struct {
	/* JSON-RPC */
	RpcEndpoint string
	NTopWallets int
	NBlocks     int64
	SavePath    string
}

func New() *Config {
	err := godotenv.Load()
	if err != nil {
		panic("can't load .env")
	}
	config := Config{}
	/* JSON-RPC */
	flag.StringVar(&config.RpcEndpoint, "rpc-endpoint", "", "JSON-RPC for address activity scraping")
	flag.IntVar(&config.NTopWallets, "n-top-wallets", 0, "Number of top wallets to parse")
	flag.Int64Var(&config.NBlocks, "n-blocks", 0, "Number of blocks to parse")
	flag.StringVar(&config.SavePath, "save-path", "default.json", "top active addresses are stored here")

	flag.Parse()
	return &config
}
