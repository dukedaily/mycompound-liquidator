package contract

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"

	"liquidator/conf"
	"liquidator/log"
)

var (
	client              *ethclient.Client
	comptrollerInstance *Comptroller
	closeFactor         decimal.Decimal
	auth                *bind.TransactOpts
	walletAddress       common.Address
)

func Init() {
	c, err := ethclient.Dial(conf.Config.Infura)
	if err != nil {
		panic(err)
	}
	client = c

	comptrollerInstance = newComptroller()
	closeFactor = getCloseFactor()

	privateKey, err := crypto.HexToECDSA(conf.Config.Wallet)
	if err != nil {
		log.Print(err)
	}
	auth, err = bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(conf.Config.Chainid))
	if err != nil {
		log.Print(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Print("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	walletAddress = crypto.PubkeyToAddress(*publicKeyECDSA)
}

func getNonce() uint64 {
	nonce, err := client.PendingNonceAt(context.Background(), walletAddress)
	if err != nil {
		log.Printf("get nonce error: %s", err)
		return 0
	}
	return nonce
}

func getGasPrice() *big.Int {
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Printf("get gas price error: %s", err)
		return common.Big0
	}
	return gasPrice
}

func newComptroller() *Comptroller {
	comptrollerAddress := common.HexToAddress(conf.Config.Comptroller)
	instance, err := NewComptroller(comptrollerAddress, client)
	if err != nil {
		log.Print(err)
	}
	return instance
}

func exponentToDecimal(decimals int) decimal.Decimal {
	ten, _ := decimal.NewFromString("10")
	result, _ := decimal.NewFromString("1")
	for i := 0; i < decimals; i++ {
		result = result.Mul(ten)
	}
	return result
}

func getCloseFactor() decimal.Decimal {
	closeFactorMantissa, err := comptrollerInstance.CloseFactorMantissa(nil)
	if err != nil {
		log.Print(err)
	}
	return decimal.NewFromBigInt(closeFactorMantissa, 0).Div(exponentToDecimal(18))
}

func IsHighRisk(address string) bool {
	account := common.HexToAddress(address)
	_, _, shortfall, _ := comptrollerInstance.GetAccountLiquidity(nil, account)
	if shortfall.Cmp(common.Big0) > 0 {
		return true
	} else {
		return false
	}
}

func GetLiquidateRepayAmount(pToken string, borrower string) *big.Int {
	pTokenInstance, err := NewPtoken(common.HexToAddress(pToken), client)
	if err != nil {
		log.Printf("NewPToken error: %s", err)
		return common.Big0
	}
	borrowBalance, err := pTokenInstance.BorrowBalanceStored(nil, common.HexToAddress(borrower))
	if err != nil {
		log.Printf("Get BorrowBalanceStored error: %s", err)
		return common.Big0
	}
	return borrowBalance.Mul(borrowBalance, closeFactor.BigInt())
}

func LiquidateCalculateSeizeTokens(pTokenBorrowed, pTokenCollateral string, actualRepayAmount *big.Int) *big.Int {
	borrowed := common.HexToAddress(pTokenBorrowed)
	collateral := common.HexToAddress(pTokenCollateral)
	_, amount, err := comptrollerInstance.LiquidateCalculateSeizeTokens(nil, borrowed, collateral, actualRepayAmount)
	if err != nil {
		log.Printf("Get LiquidateCalculateSeizeTokens error: %s", err)
		return common.Big0
	}
	return amount
}

func GetCollaterals(borrower string) []string {
	assets, err := comptrollerInstance.GetAssetsIn(nil, common.HexToAddress(borrower))
	if err != nil {
		log.Printf("GetAssetsIn error: %s", err)
		return nil
	}
	result := make([]string, 0)
	for _, asset := range assets {
		result = append(result, asset.String())
	}
	return result
}

func GetAssetBalance(asset, account string) *big.Int {
	pTokenInstance, err := NewPtoken(common.HexToAddress(asset), client)
	if err != nil {
		log.Printf("NewPToken error: %s", err)
		return common.Big0
	}
	balance, _ := pTokenInstance.BalanceOf(nil, common.HexToAddress(account))
	return balance
}

func GetWalletUnderlyingBalance(pToken string) *big.Int {
	pTokenInstance, err := NewPtoken(common.HexToAddress(pToken), client)
	if err != nil {
		log.Printf("NewPToken error: %s", err)
		return common.Big0
	}
	underlying, _ := pTokenInstance.Underlying(nil)
	erc20Instance, _ := NewErc20(underlying, client)
	balance, _ := erc20Instance.BalanceOf(nil, walletAddress)
	return balance
}

func LiquidateBorrow(asset, borrower, collateral string, repayAmount *big.Int) (string, error) {
	pTokenInstance, err := NewPtoken(common.HexToAddress(asset), client)
	if err != nil {
		log.Printf("NewPToken error: %s", err)
		return "", err
	}

	auth.Nonce = big.NewInt(int64(getNonce()))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	auth.GasPrice = getGasPrice()

	tx, err := pTokenInstance.LiquidateBorrow(auth, common.HexToAddress(borrower), repayAmount, common.HexToAddress(collateral))
	if err != nil {
		log.Printf("LiquidateBorrow error: %s", err)
		return "", err
	}

	return tx.Hash().String(), nil
}

func Approve(pToken string) (string, error) {
	pTokenAddress := common.HexToAddress(pToken)
	pTokenInstance, err := NewPtoken(pTokenAddress, client)
	if err != nil {
		log.Printf("NewPToken error: %s", err)
		return "", err
	}
	erc20Address, err := pTokenInstance.Underlying(nil)
	if err != nil {
		log.Printf("Get underlying error: %s", err)
		return "", err
	}
	erc20Instance, err := NewErc20(erc20Address, client)
	if err != nil {
		log.Printf("NewErc20 error: %s", err)
		return "", err
	}

	auth.Nonce = big.NewInt(int64(getNonce()))
	auth.Value = big.NewInt(0)      // in wei
	auth.GasLimit = uint64(3000000) // in units
	auth.GasPrice = getGasPrice()

	totalSupply, err := erc20Instance.TotalSupply(nil)
	if err != nil {
		log.Printf("Get totalSupply error: %s", err)
		return "", err
	}
	tx, err := erc20Instance.Approve(auth, pTokenAddress, totalSupply)
	if err != nil {
		log.Printf("Approve error: %s", err)
		return "", err
	}
	return tx.Hash().String(), nil
}
