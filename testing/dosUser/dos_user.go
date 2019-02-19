package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DOSNetwork/core/log"

	"github.com/DOSNetwork/core/configuration"
	"github.com/DOSNetwork/core/testing/dosUser/eth"
)

const (
	ENVQUERYTIMES = "QUERYTIMES"
	ENVQUERYTYPE  = "QUERYTYPE"
)

const (
	INVALIDQUERYINDEX = 17
	CHECKINTERVAL     = 3
	FINALREPORTDUE    = 10
)

type record struct {
	start   time.Time
	end     time.Time
	version uint8
}

type querySet struct {
	url      string
	selector string
}

var querySets = []querySet{
	{"https://api.coinbase.com/v2/prices/ETH-USD/spot", ""},
	{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "$"},
	{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "$.data"},
	{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "$.data.base"},
	{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "$.data.currency"},
	{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "$.data.amount"},
	//{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "$.data.NOTVALID"},
	{"https://api.coinmarketcap.com/v1/global/", ""},
	{"https://api.coinmarketcap.com/v1/global/", "$"},
	{"https://api.coinmarketcap.com/v1/global/", "$.total_market_cap_usd"},
	{"https://api.coinmarketcap.com/v1/global/", "$.total_24h_volume_usd"},
	{"https://api.coinmarketcap.com/v1/global/", "$.bitcoin_percentage_of_market_cap"},
	{"https://api.coinmarketcap.com/v1/global/", "$.active_currencies"},
	{"https://api.coinmarketcap.com/v1/global/", "$.active_assets"},
	{"https://api.coinmarketcap.com/v1/global/", "$.active_markets"},
	{"https://api.coinmarketcap.com/v1/global/", "$.last_updated"},
	//{"https://api.coinmarketcap.com/v1/global/", "$.NOTVALID"},

	//frequent-update queries
	//{"https://min-api.cryptocompare.com/data/price?fsym=BTC&tsyms=USD,JPY,EUR", "$"},

	//invalid queries
	//{"https://api.coinbase.com/v2/prices/ETH-USD/spot", "NOTVALID"},
	//{"https://api.coinmarketcap.com/v1/global/", "NOTVALID"},
}

var (
	envTimes        = ""
	envTypes        = ""
	userTestAdaptor = &eth.EthUserAdaptor{}
	counter         = 10
	totalQuery      = 0
	invalidQuery    = 0
	rMap            = make(map[string]string)
	canceled        = make(chan struct{})
	done            = make(chan struct{})
)
var logger log.Logger
var wg sync.WaitGroup

func main() {
	var err error

	envTypes = os.Getenv(ENVQUERYTYPE)
	if envTypes == "" {
		envTypes = "random"
	}

	envTimes = os.Getenv(ENVQUERYTIMES)
	if envTimes != "" {
		counter, err = strconv.Atoi(envTimes)
		if err != nil {
			log.Error(err)
		}
	}

	//It also need to connect to bootstrape node to get crential
	bootStrapIP := os.Getenv("BOOTSTRAPIP")
	s := strings.Split(bootStrapIP, ":")
	ip, _ := s[0], s[1]

	//
	config := eth.AMAConfig{}
	err = configuration.LoadConfig("./ama.json", &config)
	if err != nil {
		log.Fatal(err)
	}

	onChainConfig := configuration.Config{}
	if err = onChainConfig.LoadConfig(); err != nil {
		log.Fatal(err)
	}

	chainConfig := onChainConfig.GetChainConfig()

	//Wait until contract has group public key
	for {
		tServer := "http://" + ip + ":8080/hasGroupPubkey"
		// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
		resp, err := http.Get(tServer)
		for err != nil {
			log.Error(err)
			time.Sleep(10 * time.Second)
			resp, err = http.Get(tServer)
		}

		r, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
		}

		if string(r) == "yes" {
			err = resp.Body.Close()
			if err != nil {
				log.Error(err)
			}
			break
		}
	}
	if err = userTestAdaptor.SetAccount(onChainConfig.GetCredentialPath()); err != nil {
		log.Fatal(err)
	}

	log.Init(userTestAdaptor.GetId()[:])
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	if err = userTestAdaptor.Init(config.AskMeAnythingAddressPool[random.Intn(len(config.AskMeAnythingAddressPool))], chainConfig); err != nil {
		log.Fatal(err)
	}

	rootCredentialPath := "testAccounts/bootCredential/fundKey"
	if err := userTestAdaptor.BalanceMaintain(rootCredentialPath); err != nil {
		log.Fatal(err)
	}

	go func() {
		fmt.Println("regular balanceMaintain started")
		ticker := time.NewTicker(time.Hour * 8)
		for range ticker.C {
			if err := userTestAdaptor.BalanceMaintain(rootCredentialPath); err != nil {
				log.Fatal(err)
			}
		}
	}()

	logger = log.New("module", "AMAUser")
	events := make(chan interface{}, 5)
	if err = userTestAdaptor.SubscribeToAll(events); err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-events:
				switch i := event.(type) {
				case *eth.AskMeAnythingSetTimeout:

				case *eth.AskMeAnythingQueryResponseReady:
					f := map[string]interface{}{
						"RequestId": fmt.Sprintf("%x", i.QueryId),
						"Result":    i.Result,
						"Removed":   i.Removed}
					logger.Event("AMAResponseReady", f)
				case *eth.AskMeAnythingRequestSent:
					f := map[string]interface{}{
						"RequestId": fmt.Sprintf("%x", i.RequestId),
						"Removed":   i.Removed}
					logger.Event("AMARequestSent", f)
				case *eth.AskMeAnythingRandomReady:
					f := map[string]interface{}{
						"RequestId":       fmt.Sprintf("%x", i.RequestId),
						"GeneratedRandom": fmt.Sprintf("%x", i.GeneratedRandom),
						"Removed":         i.Removed}
					logger.Event("AMARandomReady", f)
				}
			}
		}
	}()
	timer := time.NewTimer(60 * time.Second)
	for {
		select {
		case <-timer.C:
			query(counter)
			counter--
			timer.Reset(60 * time.Second)
			if counter == 0 {
				timeout := time.After(60 * time.Second)
				<-timeout
				fmt.Println("There's no more time to this. Exiting!")
				return
			}
		default:
		}
	}
}

func query(counter int) {
	fmt.Println("query counter ", counter)
	f := map[string]interface{}{
		"Removed": false}
	logger.Event("AMAQueryCall", f)
	switch envTypes {
	case "url":
		lottery := rand.Intn(len(querySets))
		if err := userTestAdaptor.Query(uint8(counter), querySets[lottery].url, querySets[lottery].selector); err != nil {
			fmt.Println(err)
			return
		}
		if lottery >= INVALIDQUERYINDEX {
			invalidQuery++
		}
	case "random":
		if err := userTestAdaptor.GetSafeRandom(uint8(counter)); err != nil {
			fmt.Println(err)
			return
		}
	default:
		if counter%2 == 0 {
			lottery := rand.Intn(len(querySets))
			if err := userTestAdaptor.Query(uint8(counter), querySets[lottery].url, querySets[lottery].selector); err != nil {
				fmt.Println(err)
				return
			}
			if lottery >= INVALIDQUERYINDEX {
				invalidQuery++
			}
		} else {
			if err := userTestAdaptor.GetSafeRandom(uint8(counter)); err != nil {
				fmt.Println(err)
				return
			}
		}
	}
	totalQuery++
}
