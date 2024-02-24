package container

import (
	"fmt"
	"go.uber.org/dig"
	"walletActivityParser/src/config"
	"walletActivityParser/src/logger"
	"walletActivityParser/src/services/wallet"
)

func CreateContainer() *dig.Container {
	container := dig.New()
	must(container.Provide(config.New))
	must(container.Provide(logger.New))
	must(container.Provide(wallet.New))
	return container
}

func MustInvoke(container *dig.Container, function interface{}, opts ...dig.InvokeOption) {
	must(container.Invoke(function, opts...))
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("failed to initialize DI: %s", err))
	}
}
