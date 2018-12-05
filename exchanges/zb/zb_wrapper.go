package zb

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

// Start starts the OKEX go routine
func (z *ZB) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		z.Run()
		wg.Done()
	}()
}

// Run implements the OKEX wrapper
func (z *ZB) Run() {
	if z.Verbose {
		log.Printf("%s Websocket: %s. (url: %s).\n", z.GetName(), common.IsEnabled(z.Websocket.IsEnabled()), z.WebsocketURL)
		log.Printf("%s polling delay: %ds.\n", z.GetName(), z.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", z.GetName(), len(z.EnabledPairs), z.EnabledPairs)
	}

	markets, err := z.GetMarkets()
	if err != nil {
		log.Printf("%s Unable to fetch symbols.\n", z.GetName())
	} else {
		var currencies []string
		for x := range markets {
			currencies = append(currencies, x)
		}

		err = z.UpdateCurrencies(currencies, false, false)
		if err != nil {
			log.Printf("%s Failed to update available currencies.\n", z.GetName())
		}
	}
}

// UpdateTicker updates and returns the ticker for a currency pair
func (z *ZB) UpdateTicker(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	var tickerPrice ticker.Price

	result, err := z.GetTickers()
	if err != nil {
		return tickerPrice, err
	}

	for _, x := range z.GetEnabledCurrencies() {
		currencySplit := common.SplitStrings(exchange.FormatExchangeCurrency(z.Name, x).String(), "_")
		currency := currencySplit[0] + currencySplit[1]
		var tp ticker.Price
		tp.Pair = x
		tp.High = result[currency].High
		tp.Last = result[currency].Last
		tp.Ask = result[currency].Sell
		tp.Bid = result[currency].Buy
		tp.Last = result[currency].Last
		tp.Low = result[currency].Low
		tp.Volume = result[currency].Vol
		ticker.ProcessTicker(z.Name, x, tp, assetType)
	}

	return ticker.GetTicker(z.Name, p, assetType)
}

// GetTickerPrice returns the ticker for a currency pair
func (z *ZB) GetTickerPrice(p pair.CurrencyPair, assetType string) (ticker.Price, error) {
	tickerNew, err := ticker.GetTicker(z.GetName(), p, assetType)
	if err != nil {
		return z.UpdateTicker(p, assetType)
	}
	return tickerNew, nil
}

// GetOrderbookEx returns orderbook base on the currency pair
func (z *ZB) GetOrderbookEx(currency pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	ob, err := orderbook.GetOrderbook(z.GetName(), currency, assetType)
	if err != nil {
		return z.UpdateOrderbook(currency, assetType)
	}
	return ob, nil
}

// UpdateOrderbook updates and returns the orderbook for a currency pair
func (z *ZB) UpdateOrderbook(p pair.CurrencyPair, assetType string) (orderbook.Base, error) {
	var orderBook orderbook.Base
	currency := exchange.FormatExchangeCurrency(z.Name, p).String()

	orderbookNew, err := z.GetOrderbook(currency)
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

	orderbook.ProcessOrderbook(z.GetName(), p, orderBook, assetType)
	return orderbook.GetOrderbook(z.Name, p, assetType)
}

// GetAccountInfo retrieves balances for all enabled currencies for the
// ZB exchange
func (z *ZB) GetAccountInfo() (exchange.AccountInfo, error) {
	var response exchange.AccountInfo
	return response, errors.New("not implemented")
}

// GetFundingHistory returns funding history, deposits and
// withdrawals
func (z *ZB) GetFundingHistory() ([]exchange.FundHistory, error) {
	var fundHistory []exchange.FundHistory
	return fundHistory, common.ErrFunctionNotSupported
}

// GetExchangeHistory returns historic trade data since exchange opening.
func (z *ZB) GetExchangeHistory(p pair.CurrencyPair, assetType string) ([]exchange.TradeHistory, error) {
	var resp []exchange.TradeHistory

	return resp, common.ErrNotYetImplemented
}

// SubmitOrder submits a new order
func (z *ZB) SubmitOrder(p pair.CurrencyPair, side exchange.OrderSide, orderType exchange.OrderType, amount, price float64, clientID string) (exchange.SubmitOrderResponse, error) {
	var submitOrderResponse exchange.SubmitOrderResponse
	var oT SpotNewOrderRequestParamsType

	if side == exchange.Buy {
		oT = SpotNewOrderRequestParamsTypeBuy
	} else {
		oT = SpotNewOrderRequestParamsTypeSell
	}

	var params = SpotNewOrderRequestParams{
		Amount: amount,
		Price:  price,
		Symbol: common.StringToLower(p.Pair().String()),
		Type:   oT,
	}
	response, err := z.SpotNewOrder(params)

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
func (z *ZB) ModifyOrder(orderID int64, action exchange.ModifyOrder) (int64, error) {
	return 0, common.ErrNotYetImplemented
}

// CancelOrder cancels an order by its corresponding ID number
func (z *ZB) CancelOrder(order exchange.OrderCancellation) error {
	orderIDInt, err := strconv.ParseInt(order.OrderID, 10, 64)

	if err != nil {
		return err
	}

	return z.CancelExistingOrder(orderIDInt, exchange.FormatExchangeCurrency(z.Name, order.CurrencyPair).String())
}

// CancelAllOrders cancels all orders associated with a currency pair
func (z *ZB) CancelAllOrders(orderCancellation exchange.OrderCancellation) error {
	return common.ErrNotYetImplemented
}

// GetOrderInfo returns information on a current open order
func (z *ZB) GetOrderInfo(orderID int64) (exchange.OrderDetail, error) {
	var orderDetail exchange.OrderDetail
	return orderDetail, common.ErrNotYetImplemented
}

// GetDepositAddress returns a deposit address for a specified currency
func (z *ZB) GetDepositAddress(cryptocurrency pair.CurrencyItem) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawCryptocurrencyFunds returns a withdrawal ID when a withdrawal is
// submitted
func (z *ZB) WithdrawCryptocurrencyFunds(address string, cryptocurrency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFunds returns a withdrawal ID when a
// withdrawal is submitted
func (z *ZB) WithdrawFiatFunds(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// WithdrawFiatFundsToInternationalBank returns a withdrawal ID when a
// withdrawal is submitted
func (z *ZB) WithdrawFiatFundsToInternationalBank(currency pair.CurrencyItem, amount float64) (string, error) {
	return "", common.ErrNotYetImplemented
}

// GetWebsocket returns a pointer to the exchange websocket
func (z *ZB) GetWebsocket() (*exchange.Websocket, error) {
	return nil, common.ErrNotYetImplemented
}

// GetFeeByType returns an estimate of fee based on type of transaction
func (z *ZB) GetFeeByType(feeBuilder exchange.FeeBuilder) (float64, error) {
	return z.GetFee(feeBuilder)
}

// GetWithdrawCapabilities returns the types of withdrawal methods permitted by the exchange
func (z *ZB) GetWithdrawCapabilities() uint32 {
	return z.GetWithdrawPermissions()
}
