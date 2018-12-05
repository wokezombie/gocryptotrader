package okcoin

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// Start starts the OKCoin go routine
func (o *OKCoin) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		o.Run()
		wg.Done()
	}()
}

// Run implements the OKCoin wrapper
func (o *OKCoin) Run() {
	if o.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", o.GetName(), common.IsEnabled(o.Websocket.IsEnabled()), o.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", o.GetName(), o.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", o.GetName(), len(o.EnabledPairs), o.EnabledPairs)
	}

	if o.APIUrl == okcoinAPIURL {
		// OKCoin International
		forceUpgrade := false
		if !common.StringDataContains(o.EnabledPairs, "_") || !common.StringDataContains(o.AvailablePairs, "_") {
			forceUpgrade = true
		}

		prods, err := o.GetSpotInstruments()
		if err != nil {
			log.Printf("OKEX failed to obtain available spot instruments. Err: %d", err)
		} else {
			var pairs []string
			for x := range prods {
				pairs = append(pairs, prods[x].BaseCurrency+"_"+prods[x].QuoteCurrency)
			}

			err = o.UpdateCurrencies(pairs, false, forceUpgrade)
			if err != nil {
				log.Printf("OKEX failed to update available currencies. Err: %s", err)
			}
		}

		if forceUpgrade {
			enabledPairs := []string{"btc_usd"}
			log.Println("WARNING: Available pairs for OKCoin International reset due to config upgrade, please enable the pairs you would like again.")

			err := o.UpdateCurrencies(enabledPairs, true, true)
			if err != nil {
				log.Printf("%s failed to update currencies. Err: %s", o.Name, err)
			}
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (o *OKCoin) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	currency := exchange.FormatExchangeCurrency(o.Name, p).String()
	var tickerPrice ticker.Price

	if assetType != ticker.Spot && o.APIUrl == okcoinAPIURL {
		tick, err := o.GetFuturesTicker(currency, assetType)
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Sell
		tickerPrice.Bid = tick.Buy
		tickerPrice.Low = tick.Low
		tickerPrice.Last = tick.Last
		tickerPrice.Volume = tick.Vol
		tickerPrice.High = tick.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, assetType)
	} else {
		tick, err := o.GetTicker(currency)
		if err != nil {
			return tickerPrice, err
		}
		tickerPrice.Pair = p
		tickerPrice.Ask = tick.Sell
		tickerPrice.Bid = tick.Buy
		tickerPrice.Low = tick.Low
		tickerPrice.Last = tick.Last
		tickerPrice.Volume = tick.Vol
		tickerPrice.High = tick.High
		ticker.ProcessTicker(o.GetName(), p, tickerPrice, ticker.Spot)

	}
	return ticker.GetTicker(o.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (o *OKCoin) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(o.GetName(), p, assetType)
	if err != nil {
		return o.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (o *OKCoin) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(o.GetName(), currency, assetType)
	if err != nil {
		return o.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (o *OKCoin) UpdateOrderbook(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	orderbookNew, err := o.GetOrderBook(exchange.FormatExchangeCurrency(o.Name, currency).String(), 200, false)
	if err != nil {
		return orderBook, err
	}

	for x := range orderbookNew.Bids {
		data := orderbookNew.Bids[x]
		orderBook.Bids = append(orderBook.Bids, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	for x := range orderbookNew.Asks {
		data := orderbookNew.Asks[x]
		orderBook.Asks = append(orderBook.Asks, orderbook.Item{Amount: data[1], Price: data[0]})
	}

	orderbook.ProcessOrderbook(o.GetName(), currency, orderBook, assetType)
	return orderbook.GetOrderbook(o.Name, currency, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// OKCoin exchange
func (o *OKCoin) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	response.ExchangeName = o.GetName()
	assets, err := o.GetUserInfo()
	if err != nil {
		return response, err
	}

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "BTC",
		TotalValue:   assets.Info.Funds.Free.BTC,
		Hold:         assets.Info.Funds.Freezed.BTC,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "LTC",
		TotalValue:   assets.Info.Funds.Free.LTC,
		Hold:         assets.Info.Funds.Freezed.LTC,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "USD",
		TotalValue:   assets.Info.Funds.Free.USD,
		Hold:         assets.Info.Funds.Freezed.USD,
	})

	response.Currencies = append(response.Currencies, exchange.AccountCurrencyInfo{
		CurrencyName: "CNY",
		TotalValue:   assets.Info.Funds.Free.CNY,
		Hold:         assets.Info.Funds.Freezed.CNY,
	})

	return response, nil
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (o *OKCoin) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (o *OKCoin) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (o *OKCoin) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var oT string
	if orderType == exchange.Limit {
		if side == exchange.Buy {
			oT = "buy"
		} else {
			oT = "sell"
		}
	} else if orderType == exchange.Market {
		if side == exchange.Buy {
			oT = "buy_market"
		} else {
			oT = "sell_market"
		}
	} else {
		return submitOrderResponse, errors.New("Unsupported order type")
	}

	response, err := o.Trade(amount, price, p.Pair().String(), oT)

	if response > 0 {
		submitOrderResponse.OrderID = fmt.Sprintf("%v", response)
	}

	if err == nil {
		submitOrderResponse.IsOrderPlaced = true
	}

	return submitOrderResponse, err
}

// ModifyOrder will allow of changing orderbook placement and limit to
// market conversion
func (o *OKCoin) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (o *OKCoin) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)
	orders := []int64{orderIDInt}

	if err != nil {
		return err
	}

	_, err = o.CancelExistingOrder(orders, exchange.FormatExchangeCurrency(o.Name, order.CurrencyPair).String())

	return err
}

// CancelAllOrders cancels all orders associated with a currency pair
func (o *OKCoin) CancelAllOrders(orderCancellation exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (o *OKCoin) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (o *OKCoin) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (o *OKCoin) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (o *OKCoin) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (o *OKCoin) GetWebsocket() (*exchange.Websocket, error) {
	return o.Websocket, nil
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (o *OKCoin) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return o.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (o *OKCoin) GetWithdrawCapabilities() uint32 {
	return o.GetWithdrawPermissions()
}
