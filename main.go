package main

import (
	"walletActivityParser/src/config"
	"walletActivityParser/src/container"
	"walletActivityParser/src/logger"
	"walletActivityParser/src/services/wallet"
)

func main() {
	di := container.CreateContainer()
	container.MustInvoke(di, func(
		config *config.Config,
		logger *logger.Logger,
		fetcher *wallet.Fetcher,
	) {
		fetcher.FetchAndStore()
	})
}
