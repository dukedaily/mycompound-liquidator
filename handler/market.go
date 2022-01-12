package handler

import (
	"github.com/machinebox/graphql"

	"liquidator/contract"
	"liquidator/log"
)

type Market struct {
	Id                 string
	Name               string
	Symbol             string
	UnderlyingAddress  string
	UnderlyingName     string
	UnderlyingSymbol   string
	AccrualBlockNumber uint
	BlockTimestamp     uint
}

type QueryMarketsResp struct {
	Markets []Market
}

var markets []Market

func queryMarkets() {
	req := graphql.NewRequest(`
	query {
		markets(orderBy: accrualBlockNumber, orderDirection: desc) {
			id
			name
			symbol
			underlyingAddress
    		underlyingName
			underlyingSymbol
			accrualBlockNumber
			blockTimestamp
		}
	}
	`)

	var respData QueryMarketsResp
	if err := client.Run(ctx, req, &respData); err != nil {
		log.Print(err)
	}
	markets = respData.Markets
	log.Printf("markets: %+v", markets)
}

func handleMarket(symbol string) {
	log.Printf("handle market start %s", symbol)
	tokens := queryAccountTokens(symbol, 0)
	log.Printf("%s tokens len: %d", symbol, len(tokens))
	for _, token := range tokens {
		log.Printf("Token: %+v", token)
		if contract.IsHighRisk(token.Account.Id) {
			TokenChan <- token
		}
	}
}
