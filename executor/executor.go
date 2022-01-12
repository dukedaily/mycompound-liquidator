package executor

import (
	"liquidator/contract"
	"liquidator/handler"
	"liquidator/log"
	"math/big"
)

func Run() {
	log.Println("executor running")
	for {
		token := <-handler.TokenChan
		log.Printf("receive token: %+v", token)
		borrower := token.Account.Id
		marketId := token.Market.Id
		if contract.IsHighRisk(borrower) {
			walletUnderlyingBalance := contract.GetWalletUnderlyingBalance(marketId)
			repayAmount, collateral := calculateRepayAmountAndCollateral(token)
			if walletUnderlyingBalance.Cmp(repayAmount) < 0 {
				log.Printf("Wallet not enough balance of %s", token.Market.UnderlyingSymbol)
			} else {
				tx, err := contract.LiquidateBorrow(marketId, borrower, collateral, repayAmount)
				if err == nil {
					log.Printf("LiquidateBorrow tx: %s", tx)
				}
			}
		}
	}
}

func calculateRepayAmountAndCollateral(token handler.AccountToken) (*big.Int, string) {
	borrower := token.Account.Id
	marketId := token.Market.Id
	repayAmount := contract.GetLiquidateRepayAmount(marketId, borrower)
	collaterals := contract.GetCollaterals(borrower)
loop:
	for _, collateral := range collaterals {
		seizeAmount := contract.LiquidateCalculateSeizeTokens(marketId, collateral, repayAmount)
		balance := contract.GetAssetBalance(collateral, borrower)
		if balance.Cmp(seizeAmount) >= 0 {
			return repayAmount, collateral
		}
	}
	// 如果执行到这里，说明单个抵押物无法一次偿还当前的repayAmount
	repayAmount = repayAmount.Div(repayAmount, big.NewInt(2))
	goto loop
}
