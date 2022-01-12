package handler

import (
	"liquidator/conf"
	"liquidator/contract"
	"liquidator/log"

	"context"

	"github.com/machinebox/graphql"
	"github.com/robfig/cron"
)

var (
	ctx       context.Context
	client    *graphql.Client
	TokenChan chan AccountToken
)

func Start() {
	ctx = context.Background()
	client = graphql.NewClient(conf.Config.Subgraph)
	TokenChan = make(chan AccountToken, 1000)

	queryMarkets()
	approveAll()
	startCron()
}

func approveAll() {
	for _, market := range markets {
		contract.Approve(market.Id)
	}
}

func startCron() {
	c := cron.New()
	c.AddFunc("* 0/1 * * * ?", queryMarkets)
	c.AddFunc("0/30 * * * * ?", taskRun)
	c.Start()
}

func taskRun() {
	log.Print("cron task running")
	for _, market := range markets {
		go handleMarket(market.Symbol)
	}
}
