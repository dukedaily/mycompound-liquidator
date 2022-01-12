package handler

import (
	"liquidator/log"
	"liquidator/utils"

	"github.com/machinebox/graphql"
	"github.com/shopspring/decimal"
)

type Account struct {
	Id string
}

type AccountToken struct {
	Id                  string
	Symbol              string
	AccrualBlockNumber  string
	PTokenBalance       decimal.Decimal
	StoredBorrowBalance decimal.Decimal
	Market              Market
	Account             Account
}

type QueryAccountTokensResp struct {
	AccountPTokens []AccountToken
}

func queryAccountTokens(symbol string, lastBlockNumber uint) []AccountToken {
	req := graphql.NewRequest(`
	query ($symbol: String!, $lastBlockNumber: BigInt) {
		accountPTokens(first: 1000, orderBy: accrualBlockNumber, 
			where: {accrualBlockNumber_gt: $lastBlockNumber, storedBorrowBalance_gt: 0, symbol: $symbol}) {
			id
			symbol
			pTokenBalance
			accrualBlockNumber
			storedBorrowBalance
			market {
				id
				underlyingAddress
				underlyingSymbol
			}
			account {
				id
			}
		}
	}
	`)
	req.Var("symbol", symbol)
	req.Var("lastBlockNumber", lastBlockNumber)

	var respData QueryAccountTokensResp
	if err := client.Run(ctx, req, &respData); err != nil {
		log.Print(err)
	}
	tokens := respData.AccountPTokens
	tokensLen := len(tokens)
	if tokensLen == 1000 {
		blockNumber := utils.String2Uint(tokens[tokensLen-1].AccrualBlockNumber)
		tokens2 := queryAccountTokens(symbol, blockNumber)
		tokens = append(tokens, tokens2...)
	}

	return tokens
}
