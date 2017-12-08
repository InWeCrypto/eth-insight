package insight

import (
	"bytes"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Jeffail/gabs"
	log "github.com/Sirupsen/logrus"
)

var latestGasPrice atomic.Value

func updateGasTask() {
	for {
		if resp, err := http.Post(globalConfig.geth,
			"application/json",
			bytes.NewBufferString(`{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":0}`)); err == nil {
			jsonParsed, _ := gabs.ParseJSONBuffer(resp.Body)
			value, ok := jsonParsed.Path("result").Data().(string)
			if !ok {
				log.Println("cannot get gasPrice", jsonParsed)
			}
			latestGasPrice.Store(value)
		} else {
			log.Println(err)
		}
		<-time.After(globalConfig.gasUpdate)
	}
}
