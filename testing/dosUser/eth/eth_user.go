package eth

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/DOSNetwork/core/configuration"
	"github.com/DOSNetwork/core/log"
	"github.com/DOSNetwork/core/onchain"
	"github.com/DOSNetwork/core/testing/dosUser/contract"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// TODO: Instead of hardcode, read from DOSAddressBridge.go
const (
	SubscribeAskMeAnythingSetTimeout = iota
	SubscribeAskMeAnythingQueryResponseReady
	SubscribeAskMeAnythingRequestSent
	SubscribeAskMeAnythingRandomReady
)

type AMAConfig struct {
	AskMeAnythingAddressPool []string
}

type EthUserAdaptor struct {
	onchain.EthCommon
	proxy     *dosUser.AskMeAnything
	lock      *sync.Mutex
	logFilter *sync.Map
	address   string
	logger    log.Logger
}

func (e *EthUserAdaptor) Init(address string, config configuration.ChainConfig) (err error) {
	if err = e.EthCommon.Init(config); err != nil {
		return
	}
	e.logFilter = new(sync.Map)
	go e.logMapTimeout()

	e.address = address
	fmt.Println("onChainConn initialization finished.")
	e.lock = new(sync.Mutex)
	e.dialToProxy()
	e.logger = log.New("module", "EthUser")
	return
}

func (e *EthUserAdaptor) dialToProxy() (err error) {
	e.lock.Lock()
	fmt.Println("dialing To Proxy...")
	addr := common.HexToAddress(e.address)
	e.proxy, err = dosUser.NewAskMeAnything(addr, e.Client)
	for err != nil {
		fmt.Println(err)
		fmt.Println("Connot Create new proxy, retrying...")
		e.proxy, err = dosUser.NewAskMeAnything(addr, e.Client)
	}
	e.lock.Unlock()
	return
}

func (e *EthUserAdaptor) resetOnChainConn() (err error) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if err = e.EthCommon.DialToEth(); err != nil {
		e.logger.Error(err)
		return
	}
	err = e.dialToProxy()
	return
}

func (e *EthUserAdaptor) GetId() (id []byte) {
	return e.GetAddress().Bytes()
}

func (e *EthUserAdaptor) SubscribeEvent(ch chan interface{}, subscribeType int) (err error) {
	opt := &bind.WatchOpts{}
	var cancel context.CancelFunc
	opt.Context, cancel = context.WithCancel(context.Background())
	done := make(chan bool)
	timer := time.NewTimer(onchain.SubscribeTimeout * time.Second)

	go e.subscribeEventAttempt(ch, opt, subscribeType, done)

	for {
		select {
		case succ := <-done:
			if succ {
				fmt.Println("subscribe done")
				return
			} else {
				fmt.Println("retry...")
				if err = e.resetOnChainConn(); err != nil {
					return
				} else {
					go e.subscribeEventAttempt(ch, opt, subscribeType, done)
				}
			}
		case <-timer.C:
			cancel()
			fmt.Println("subscribe timeout")
			return
		}
	}
}

func (e *EthUserAdaptor) fetchMatureLogs(ctx context.Context, subscribeType int, ch chan interface{}) (err error) {
	targetBlockN, err := e.GetCurrentBlock()
	if err != nil {
		return
	}

	timer := time.NewTimer(onchain.LogCheckingInterval * time.Second)
	go func() {
		for {
			select {
			case <-timer.C:
				currentBlockN, err := e.GetCurrentBlock()
				if err != nil {
					e.logger.Error(err)
				}
				for ; currentBlockN-onchain.LogBlockDiff >= targetBlockN; targetBlockN++ {
					fmt.Println("checking Block", currentBlockN)
					switch subscribeType {
					case SubscribeAskMeAnythingQueryResponseReady:
						logs, err := e.proxy.AskMeAnythingFilterer.FilterQueryResponseReady(&bind.FilterOpts{
							Start:   targetBlockN,
							End:     &targetBlockN,
							Context: ctx,
						})
						if err != nil {
							e.logger.Error(err)
						}
						for logs.Next() {
							ch <- &AskMeAnythingQueryResponseReady{
								QueryId: logs.Event.QueryId,
								Result:  logs.Event.Result,
								Tx:      logs.Event.Raw.TxHash.Hex(),
								BlockN:  logs.Event.Raw.BlockNumber,
								Removed: logs.Event.Raw.Removed,
							}
							f := map[string]interface{}{
								"RequestId": fmt.Sprintf("%x", logs.Event.QueryId),
								"Message":   logs.Event.Result,
								"Removed":   logs.Event.Raw.Removed,
								"Tx":        logs.Event.Raw.TxHash.Hex(),
								"BlockN":    logs.Event.Raw.BlockNumber,
								"Time":      time.Now()}
							e.logger.Event("EthUserQueryReady", f)
						}
						if logs.Error() != nil {
							e.logger.Error(logs.Error())
						}
						if logs.Close() != nil {
							e.logger.Error(logs.Error())
						}
					case SubscribeAskMeAnythingRequestSent:
						logs, err := e.proxy.AskMeAnythingFilterer.FilterRequestSent(&bind.FilterOpts{
							Start:   targetBlockN,
							End:     &targetBlockN,
							Context: ctx,
						}, []common.Address{e.GetAddress()})
						if err != nil {
							e.logger.Error(err)
						}
						for logs.Next() {
							f := map[string]interface{}{
								"Event":     "EthUserRequestSent",
								"RequestId": fmt.Sprintf("%x", logs.Event.RequestId),
								"Succ":      logs.Event.Succ,
								"Removed":   logs.Event.Raw.Removed,
								"Tx":        logs.Event.Raw.TxHash.Hex(),
								"BlockN":    logs.Event.Raw.BlockNumber,
								"Time":      time.Now()}
							e.logger.Event("EthUserRequestSent", f)
							ch <- &AskMeAnythingRequestSent{
								InternalSerial: logs.Event.InternalSerial,
								Succ:           logs.Event.Succ,
								RequestId:      logs.Event.RequestId,
								Tx:             logs.Event.Raw.TxHash.Hex(),
								BlockN:         logs.Event.Raw.BlockNumber,
								Removed:        logs.Event.Raw.Removed,
							}
						}
						if logs.Error() != nil {
							e.logger.Error(logs.Error())
						}
						if logs.Close() != nil {
							e.logger.Error(logs.Error())
						}
					case SubscribeAskMeAnythingRandomReady:
						logs, err := e.proxy.AskMeAnythingFilterer.FilterRandomReady(&bind.FilterOpts{
							Start:   targetBlockN,
							End:     &targetBlockN,
							Context: ctx,
						})
						if err != nil {
							e.logger.Error(err)
						}
						for logs.Next() {
							ch <- &AskMeAnythingRandomReady{
								GeneratedRandom: logs.Event.GeneratedRandom,
								RequestId:       logs.Event.RequestId,
								Tx:              logs.Event.Raw.TxHash.Hex(),
								BlockN:          logs.Event.Raw.BlockNumber,
								Removed:         logs.Event.Raw.Removed,
							}
							f := map[string]interface{}{
								"RequestId":       fmt.Sprintf("%x", logs.Event.RequestId),
								"GeneratedRandom": fmt.Sprintf("%x", logs.Event.GeneratedRandom),
								"Removed":         logs.Event.Raw.Removed,
								"Tx":              logs.Event.Raw.TxHash.Hex(),
								"BlockN":          logs.Event.Raw.BlockNumber,
								"Time":            time.Now()}
							e.logger.Event("EthUserRandomReady", f)
						}
						if logs.Error() != nil {
							e.logger.Error(logs.Error())
						}
						if logs.Close() != nil {
							e.logger.Error(logs.Error())
						}
					}
				}
				timer.Reset(onchain.LogCheckingInterval * time.Second)
			case <-ctx.Done():
				return
			}
		}
	}()
	return
}

func (e *EthUserAdaptor) GetCurrentBlock() (blknum uint64, err error) {
	var header *types.Header
	header, err = e.Client.HeaderByNumber(context.Background(), nil)
	if err == nil {
		blknum = header.Number.Uint64()
	}
	return
}

func (e *EthUserAdaptor) subscribeEventAttempt(ch chan interface{}, opt *bind.WatchOpts, subscribeType int, done chan bool) {
	fmt.Println("attempt to subscribe event...")
	switch subscribeType {
	case SubscribeAskMeAnythingSetTimeout:
		fmt.Println("subscribing AskMeAnythingSetTimeout event...")
		transitChan := make(chan *dosUser.AskMeAnythingSetTimeout)
		sub, err := e.proxy.AskMeAnythingFilterer.WatchSetTimeout(opt, transitChan)
		if err != nil {
			done <- false
			fmt.Println(err)
			fmt.Println("Network fail, will retry shortly")
			return
		}
		fmt.Println("AskMeAnythingSetTimeout event subscribed")

		done <- true
		for {
			select {
			case err := <-sub.Err():
				log.Fatal(err)
			case i := <-transitChan:
				if !e.filterLog(i.Raw) {
					ch <- &AskMeAnythingSetTimeout{
						PreviousTimeout: i.PreviousTimeout,
						NewTimeout:      i.NewTimeout,
					}
				}
			}
		}
	case SubscribeAskMeAnythingQueryResponseReady:
		if err := e.fetchMatureLogs(opt.Context, SubscribeAskMeAnythingQueryResponseReady, ch); err != nil {
			done <- false
			e.logger.Error(err)
			fmt.Println("Network fail, will retry shortly")
			return
		} else {
			done <- true
		}
	case SubscribeAskMeAnythingRequestSent:
		if err := e.fetchMatureLogs(opt.Context, SubscribeAskMeAnythingRequestSent, ch); err != nil {
			done <- false
			e.logger.Error(err)
			fmt.Println("Network fail, will retry shortly")
			return
		} else {
			done <- true
		}

	case SubscribeAskMeAnythingRandomReady:
		if err := e.fetchMatureLogs(opt.Context, SubscribeAskMeAnythingRandomReady, ch); err != nil {
			done <- false
			e.logger.Error(err)
			fmt.Println("Network fail, will retry shortly")
			return
		} else {
			done <- true
		}
	}
}

func (e *EthUserAdaptor) Query(internalSerial uint8, url, selector string) (err error) {
	auth, err := e.GetAuth()
	if err != nil {
		fmt.Println(" Query GetAuth err ", err)
		return
	}

	tx, err := e.proxy.AMA(auth, internalSerial, url, selector)
	for err != nil && (err.Error() == core.ErrNonceTooLow.Error() || err.Error() == core.ErrReplaceUnderpriced.Error()) {
		fmt.Println(err)
		time.Sleep(time.Second)
		fmt.Println("transaction retry...")
		tx, err = e.proxy.AMA(auth, internalSerial, url, selector)
	}
	if err != nil {
		fmt.Println(" Query AMAerr ", err)
		return
	}

	fmt.Println("tx sent: ", tx.Hash().Hex())
	fmt.Println("Querying ", url, "selector", selector, "waiting for confirmation...")

	//err = e.CheckTransaction(tx)

	return
}

func (e *EthUserAdaptor) GetSafeRandom(internalSerial uint8) (err error) {
	auth, err := e.GetAuth()
	if err != nil {
		return
	}

	tx, err := e.proxy.RequestSafeRandom(auth, internalSerial)
	for err != nil && (err.Error() == core.ErrNonceTooLow.Error() || err.Error() == core.ErrReplaceUnderpriced.Error()) {
		fmt.Println(err)
		time.Sleep(time.Second)
		fmt.Println("transaction retry...")
		tx, err = e.proxy.RequestSafeRandom(auth, internalSerial)
	}
	if err != nil {
		return
	}

	fmt.Println("tx sent: ", tx.Hash().Hex())
	fmt.Println("RequestSafeRandom ", " waiting for confirmation...")

	//err = e.CheckTransaction(tx)

	return
}

func (e *EthUserAdaptor) GetFastRandom() (err error) {
	auth, err := e.GetAuth()
	if err != nil {
		return
	}

	tx, err := e.proxy.RequestFastRandom(auth)
	for err != nil && (err.Error() == core.ErrNonceTooLow.Error() || err.Error() == core.ErrReplaceUnderpriced.Error()) {
		fmt.Println(err)
		time.Sleep(time.Second)
		fmt.Println("transaction retry...")
		tx, err = e.proxy.RequestFastRandom(auth)
	}
	if err != nil {
		return
	}

	fmt.Println("tx sent: ", tx.Hash().Hex())
	fmt.Println("RequestSafeRandom ", " waiting for confirmation...")

	//err = e.CheckTransaction(tx)

	return
}

func (e *EthUserAdaptor) SubscribeToAll(msgChan chan interface{}) (err error) {
	for i := SubscribeAskMeAnythingSetTimeout; i <= SubscribeAskMeAnythingRandomReady; i++ {
		err = e.SubscribeEvent(msgChan, i)
	}
	return
}

type logRecord struct {
	content       types.Log
	currTimeStamp time.Time
}

func (e *EthUserAdaptor) filterLog(raw types.Log) (duplicates bool) {
	fmt.Println("check duplicates")
	identityBytes := append(raw.Address.Bytes(), raw.Topics[0].Bytes()...)
	identityBytes = append(identityBytes, raw.Data...)
	identity := new(big.Int).SetBytes(identityBytes).String()

	var record interface{}
	if record, duplicates = e.logFilter.Load(identity); duplicates {
		fmt.Println("got duplicate event", record, "\n", raw)
	}
	e.logFilter.Store(identity, logRecord{raw, time.Now()})

	return
}

func (e *EthUserAdaptor) logMapTimeout() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		e.logFilter.Range(e.checkTime)
	}

}

func (e *EthUserAdaptor) checkTime(log, deliverTime interface{}) (okToDelete bool) {
	switch t := deliverTime.(type) {
	case logRecord:
		if time.Now().Sub(t.currTimeStamp).Seconds() > 60*10 {
			e.logFilter.Delete(log)
		}
	}
	return true
}
